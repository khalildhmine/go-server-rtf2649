package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"repair-service-server/config"
	"repair-service-server/database"
	"repair-service-server/jobs"
	"repair-service-server/middleware"
	"repair-service-server/models"
	"repair-service-server/routes"
	"repair-service-server/services"
	ws "repair-service-server/websocket"
)

// Global chat hub for WebSocket notifications
var globalChatHub *ws.Hub

// GetGlobalChatHub returns the global chat hub instance
func GetGlobalChatHub() *ws.Hub {
	return globalChatHub
}

// BroadcastServiceRequest sends a service request ID to the broadcast channel
func BroadcastServiceRequest(serviceRequestID uint) {
	if serviceRequestBroadcastChan != nil {
		select {
		case serviceRequestBroadcastChan <- serviceRequestID:
			log.Printf("📡 Service request %d queued for WebSocket broadcast", serviceRequestID)
		default:
			log.Printf("⚠️ Service request broadcast channel is full, dropping request %d", serviceRequestID)
		}
	} else {
		log.Printf("⚠️ Service request broadcast channel not initialized")
	}
}

// serviceRequestBroadcastChan is a channel for broadcasting service requests via WebSocket
var serviceRequestBroadcastChan chan uint

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Load configuration
	config.Load()

	// Initialize database
	if err := database.Initialize(); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	// Register models for auto-migration
	database.DB.AutoMigrate(
		&models.User{},
		&models.ServiceCategory{},
		&models.ServiceOption{},
		&models.WorkerProfile{},
		&models.CustomerServiceRequest{},
		&models.Service{},
		&models.Address{},
		// Chat models
		&models.ChatRoom{},
		&models.ChatMessage{},
		&models.ChatNotification{},
		&models.UserDeviceToken{},
		// Rating and service history models
		&models.WorkerRating{},
		&models.ServiceHistory{},
		// Worker analytics models
		&models.WorkerStats{},
		&models.WorkerDailyStats{},
		&models.WorkerMonthlyStats{},
		// Security models
		&models.RefreshToken{},
		// Notification models
		&models.Notification{},
		&models.PushToken{},
	)

	// Set Gin mode
	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create router
	router := gin.New()
	
	// Enterprise-grade security middleware stack
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	
	// Disable automatic redirects for trailing slashes
	router.RedirectTrailingSlash = false
	router.RedirectFixedPath = false

	// Security headers (must be first)
	router.Use(middleware.SecurityHeadersMiddleware())
	
	// Input validation
	router.Use(middleware.InputValidationMiddleware())
	
	// Rate limiting
	router.Use(middleware.RateLimitMiddleware())
	
	// Secure CORS
	router.Use(middleware.CORSMiddleware())
	
	// Audit logging
	router.Use(middleware.AuditLogMiddleware())

	// Global middleware
	router.Use(middleware.Logger())
	router.Use(middleware.Recovery())

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"message": "Repair Service Server is running",
			"time":    time.Now().UTC(),
		})
	})

	// AI Chat WebSocket endpoint
	aiChatHandler := ws.NewAIChatHandler()
	router.GET("/api/v1/ws/ai-chat", aiChatHandler.HandleAIChat)

	// Worker WebSocket endpoint for notifications
	workerHandler := ws.NewWorkerHandler()
	router.GET("/api/v1/ws/worker", workerHandler.HandleWorker)


	// Initialize chat hub and routes
	globalChatHub = ws.NewHub()
	go globalChatHub.Run()
	
	// Initialize service request broadcast channel
	serviceRequestBroadcastChan = make(chan uint, 100)
	
	// Start service request broadcasting goroutine
	go func() {
		for serviceRequestID := range serviceRequestBroadcastChan {
			if globalChatHub == nil {
				log.Printf("⚠️ WebSocket hub not available for service request broadcast")
				continue
			}
			
			// Load service request with relationships for complete data
			var fullRequest models.CustomerServiceRequest
			if err := database.DB.
				Preload("Customer").
				Preload("Category").
				Preload("ServiceOption").
				First(&fullRequest, serviceRequestID).Error; err != nil {
				log.Printf("❌ Failed to load service request details: %v", err)
				continue
			}

			// Create WebSocket message for service request
			websocketMessage := &ws.Message{
				Type: "service_request",
				Data: map[string]interface{}{
					"request_id":           fullRequest.ID,
					"title":                fullRequest.Title,
					"description":          fullRequest.Description,
					"category_id":          fullRequest.CategoryID,
					"service_option_id":    fullRequest.ServiceOptionID,
					"location_address":     fullRequest.LocationAddress,
					"location_city":        fullRequest.LocationCity,
					"location_lat":         fullRequest.LocationLat,
					"location_lng":         fullRequest.LocationLng,
					"priority":             fullRequest.Priority,
					"budget":               fullRequest.Budget,
					"estimated_duration":   fullRequest.EstimatedDuration,
					"customer_name":        fullRequest.Customer.FullName,
					"category_name":        fullRequest.Category.Name,
					"created_at":           fullRequest.CreatedAt,
					"status":               fullRequest.Status,
				},
				Timestamp: time.Now(),
			}

			// Broadcast to all connected workers
			globalChatHub.Broadcast <- websocketMessage
			
			log.Printf("📡 Service request %d broadcasted via WebSocket to all connected workers", serviceRequestID)
		}
	}()

	routes.InitChatHub()
	routes.ChatRoutes(router, globalChatHub)

	// API routes
	api := router.Group("/api/v1")
	{
		// Auth routes (no authentication required) - with strict rate limiting
		authRoutes := api.Group("/auth")
		authRoutes.Use(middleware.AuthRateLimitMiddleware()) // Stricter rate limiting for auth
		routes.RegisterSecureAuthRoutes(authRoutes) // Use secure auth routes

		// Service routes (public)
		serviceRoutes := api.Group("/services")
		routes.RegisterServiceRoutes(serviceRoutes)

		// Category routes (public)
		routes.RegisterCategoryRoutes(api)
		routes.RegisterServiceOptionRoutes(api) // Add this line

		// Note: Rating and service history routes are now protected and require authentication

		// Protected routes
		protected := api.Group("")
		protected.Use(middleware.AuthMiddleware())
		{
			// Debug route to test protected group
			protected.GET("/debug", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{
					"message": "Protected route working",
					"user_id": c.GetUint("user_id"),
				})
			})
			
			// Auth routes that require authentication (handled in RegisterSecureAuthRoutes)
			
			// Address routes (protected) - need authentication for user_id
			addressRoutes := protected.Group("/addresses")
			routes.RegisterAddressRoutes(addressRoutes)
			
			// Location routes (protected) - need authentication
			locationRoutes := protected.Group("/location")
			routes.RegisterLocationRoutes(locationRoutes)
			
			// Service request routes (protected) - need authentication
			log.Printf("🔧 Registering service request routes...")
			serviceRequestRoutes := protected.Group("/service-requests")
			routes.RegisterServiceRequestRoutes(serviceRequestRoutes)
			log.Printf("✅ Service request routes registered successfully")
			
			// Test route to verify protected group is working
			protected.GET("/test-service-requests", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{
					"message": "Service requests test route working",
					"user_id": c.GetUint("user_id"),
				})
			})

			// Debug route to check database state
			protected.GET("/debug/database", func(c *gin.Context) {
				// Check worker profiles
				var workerCount int64
				database.DB.Model(&models.WorkerProfile{}).Count(&workerCount)
				
				var availableWorkerCount int64
				database.DB.Model(&models.WorkerProfile{}).Where("is_available = ?", true).Count(&availableWorkerCount)
				
				var serviceRequestCount int64
				database.DB.Model(&models.CustomerServiceRequest{}).Count(&serviceRequestCount)
				
				var broadcastRequestCount int64
				database.DB.Model(&models.CustomerServiceRequest{}).Where("status = ?", "broadcast").Count(&broadcastRequestCount)
				
				c.JSON(http.StatusOK, gin.H{
					"message": "Database debug info",
					"total_workers": workerCount,
					"available_workers": availableWorkerCount,
					"total_service_requests": serviceRequestCount,
					"broadcast_requests": broadcastRequestCount,
				})
			})
			
			// Debug route to check specific worker's requests
			protected.GET("/debug/worker/:id/requests", func(c *gin.Context) {
				workerID := c.Param("id")
				workerIDInt, err := strconv.Atoi(workerID)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid worker ID"})
					return
				}
				
				// Get worker profile
				var workerProfile models.WorkerProfile
				if err := database.DB.First(&workerProfile, workerIDInt).Error; err != nil {
					c.JSON(http.StatusNotFound, gin.H{"error": "Worker not found"})
					return
				}
				
				// Get all requests for this worker
				var requests []models.CustomerServiceRequest
				if err := database.DB.Where("assigned_worker_id = ?", workerIDInt).Find(&requests).Error; err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch requests"})
					return
				}
				
				// Get available requests in worker's category
				var availableRequests []models.CustomerServiceRequest
				if err := database.DB.Where("category_id = ? AND status = ? AND assigned_worker_id IS NULL", 
					workerProfile.CategoryID, "broadcast").Find(&availableRequests).Error; err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch available requests"})
					return
				}
				
				c.JSON(http.StatusOK, gin.H{
					"message": "Worker requests debug info",
					"worker": gin.H{
						"id": workerProfile.ID,
						"user_id": workerProfile.UserID,
						"category_id": workerProfile.CategoryID,
						"is_available": workerProfile.IsAvailable,
						"current_lat": workerProfile.CurrentLat,
						"current_lng": workerProfile.CurrentLng,
					},
					"assigned_requests": requests,
					"available_requests_in_category": availableRequests,
				})
			})
			
			// Worker routes
			routes.RegisterWorkerRoutes(protected)
			
			// Worker service request routes (protected)
			protected.GET("/worker/available-requests", routes.GetAvailableServiceRequests)
			protected.GET("/worker/scheduled-requests", routes.GetScheduledServiceRequests)
			protected.GET("/worker/active-requests", routes.GetWorkerActiveRequests)
			protected.POST("/worker/requests/:id/respond", routes.RespondToServiceRequest)
			protected.POST("/worker/requests/:id/start", routes.StartServiceRequest)
			protected.POST("/worker/requests/:id/complete", routes.CompleteServiceRequest)
			
			// Rating routes (protected - require authentication)
			routes.RegisterRatingRoutes(protected)
			
			// Service history routes (protected - require authentication)
			routes.RegisterServiceHistoryRoutes(protected)
			
			// Worker analytics routes (protected - require authentication)
			routes.RegisterWorkerAnalyticsRoutes(protected)

			// Worker media upload routes (protected)
			routes.RegisterWorkerMediaRoutes(protected)
			
			// Service request routes already registered above
			
			// Notification routes (protected)
			notifications := api.Group("/notifications")
			notifications.Use(middleware.AuthMiddleware())
			notifications.POST("/register-token", routes.RegisterPushToken)
			notifications.GET("/has-token", routes.HasPushToken)
			notifications.GET("/", routes.GetUserNotifications)
			notifications.GET("", routes.GetUserNotifications)
			notifications.GET("/test", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Notification routes working!"})
			})
			notifications.GET("/unread-count", routes.GetUnreadCount)
			notifications.POST("/mark-read/:id", routes.MarkNotificationAsRead)
			notifications.POST("/mark-all-read", routes.MarkAllNotificationsAsRead)
			
			// Campaign notifications
			notifications.POST("/send-campaign", routes.SendCampaignNotification)
			notifications.POST("/schedule-campaign", routes.ScheduleCampaignNotification)
			
			// User activity tracking
			notifications.POST("/user-activity", routes.TrackUserActivity)
			
			// Feedback submission
			notifications.POST("/feedback", routes.SubmitFeedback)
			
			// Test notifications (development only)
			notifications.POST("/create-test", routes.CreateTestNotifications)
		}

		// Admin authentication routes (no authentication required)
		adminAuth := api.Group("/admin/auth")
		adminAuth.POST("/login", routes.AdminLogin)
		adminAuth.POST("/refresh", routes.AdminRefreshToken)

		// Admin routes (protected with admin authentication)
		adminRoutes := api.Group("/admin")
		adminRoutes.Use(routes.AdminAuthMiddleware())
		{
			// Admin current user
			adminRoutes.GET("/auth/me", routes.GetCurrentAdmin)

			// Admin dashboard
			adminRoutes.GET("/dashboard/stats", routes.GetDashboardStats)

			// Admin user management
			adminRoutes.GET("/users", routes.GetAllUsers)
			adminRoutes.GET("/users/:id", routes.GetUserById)
			adminRoutes.PATCH("/users/:id/status", routes.UpdateUserStatus)
			adminRoutes.DELETE("/users/:id", routes.DeleteUser)

			// Admin worker management
			adminRoutes.GET("/workers", routes.GetAllWorkers)
			adminRoutes.GET("/workers/:id", routes.GetWorkerById)
			adminRoutes.GET("/workers/:id/stats", routes.GetWorkerStatsForAdmin)
			adminRoutes.PATCH("/workers/:id/verify", routes.VerifyWorker)
			adminRoutes.PATCH("/workers/:id/availability", routes.UpdateWorkerAvailability)

			// Admin service request management
			adminRoutes.GET("/service-requests", routes.GetAllServiceRequests)
			adminRoutes.GET("/service-requests/:id", routes.GetServiceRequestById)

			// Admin services management
			adminRoutes.GET("/services", routes.GetAllServices)
			adminRoutes.POST("/services", routes.CreateService)
			adminRoutes.PUT("/services/:id", routes.UpdateService)
			adminRoutes.DELETE("/services/:id", routes.DeleteService)

			// Admin service options management
			adminRoutes.GET("/service-options", routes.GetAllServiceOptionsForAdmin)
			adminRoutes.POST("/service-options", routes.CreateServiceOptionForAdmin)
			adminRoutes.PUT("/service-options/:id", routes.UpdateServiceOptionForAdmin)
			adminRoutes.DELETE("/service-options/:id", routes.DeleteServiceOptionForAdmin)

			// Admin categories
			adminRoutes.GET("/categories", routes.GetServiceCategories)
			adminRoutes.POST("/categories", routes.CreateCategory)
			adminRoutes.PUT("/categories/:id", routes.UpdateCategory)
			adminRoutes.DELETE("/categories/:id", routes.DeleteCategory)
		}
	}

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Start background jobs
	expirationJob := jobs.NewExpirationJob()
	expirationJob.Start()
	defer expirationJob.Stop()

	// Start token cleanup job
	go func() {
		ticker := time.NewTicker(24 * time.Hour) // Run daily
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				jwtService := services.NewJWTService()
				if err := jwtService.CleanupExpiredTokens(); err != nil {
					log.Printf("❌ Token cleanup failed: %v", err)
				}
			}
		}
	}()

	log.Printf("Server starting on port %s", port)
	if err := router.Run("0.0.0.0:" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
