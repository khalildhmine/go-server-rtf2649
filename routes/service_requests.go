package routes

import (
	"log"
	"net/http"
	"repair-service-server/database"
	"repair-service-server/models"
	"repair-service-server/services"
	"repair-service-server/utils"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// RegisterServiceRequestRoutes registers all service request-related routes
func RegisterServiceRequestRoutes(router *gin.RouterGroup) {
	log.Printf("🔧 RegisterServiceRequestRoutes called with router: %v", router)
	
	// Create a new service request
	router.POST("/", createServiceRequest)

	// Urgent service request (priority=urgent, broadcast immediately)
	router.POST("/urgent", createUrgentServiceRequest)

	// Scheduled service request (status=scheduled, scheduled_for set)
	router.POST("/scheduled", createScheduledServiceRequest)
	log.Printf("✅ POST / route registered")
	
	// Get customer's service requests
	router.GET("/my-requests", getMyServiceRequests)
	log.Printf("✅ GET /my-requests route registered")
	
	// Get a specific service request
	router.GET("/:id", getServiceRequest)
	log.Printf("✅ GET /:id route registered")
	
	// Update service request status
	router.PUT("/:id/status", updateServiceRequestStatus)
	log.Printf("✅ PUT /:id/status route registered")
	
	// Cancel a service request
	router.POST("/:id/cancel", cancelServiceRequest)
	log.Printf("✅ POST /:id/cancel route registered")
	
	// Rate and review a completed service
	router.POST("/:id/review", reviewService)
	log.Printf("✅ POST /:id/review route registered")
	
	log.Printf("🎯 All service request routes registered successfully")
}
// createUrgentServiceRequest creates a high-priority request and broadcasts it
func createUrgentServiceRequest(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req models.CustomerServiceRequestCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Force urgent priority
	req.Priority = "urgent"

	if !utils.IsLocationValid(req.LocationLat, req.LocationLng) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid location coordinates"})
		return
	}

	expiresAt := time.Now().Add(3 * time.Minute)

	serviceRequest := models.CustomerServiceRequest{
		CustomerID:        userID,
		CategoryID:        req.CategoryID,
		ServiceOptionID:   req.ServiceOptionID,
		Title:             req.Title,
		Description:       req.Description,
		Priority:          req.Priority,
		Budget:            req.Budget,
		EstimatedDuration: req.EstimatedDuration,
		LocationLat:       &req.LocationLat,
		LocationLng:       &req.LocationLng,
		LocationAddress:   req.LocationAddress,
		LocationCity:      req.LocationCity,
		Status:            models.RequestStatusBroadcast,
		ExpiresAt:         &expiresAt,
	}

	if err := database.DB.Create(&serviceRequest).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create service request"})
		return
	}

	go broadcastServiceRequest(serviceRequest)

	c.JSON(http.StatusCreated, gin.H{
		"message": "Urgent service request created",
		"service_request": serviceRequest,
	})
}

// createScheduledServiceRequest creates a scheduled request without immediate broadcast
func createScheduledServiceRequest(c *gin.Context) {
	userID := c.GetUint("user_id")

	var body struct {
		models.CustomerServiceRequestCreate
		ScheduledFor string `json:"scheduled_for" binding:"required"` // ISO8601
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !utils.IsLocationValid(body.LocationLat, body.LocationLng) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid location coordinates"})
		return
	}

	schedTime, err := time.Parse(time.RFC3339, body.ScheduledFor)
	if err != nil || schedTime.Before(time.Now()) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "scheduled_for must be a future ISO time"})
		return
	}

	serviceRequest := models.CustomerServiceRequest{
		CustomerID:        userID,
		CategoryID:        body.CategoryID,
		ServiceOptionID:   body.ServiceOptionID,
		Title:             body.Title,
		Description:       body.Description,
		Priority:          ifEmpty(body.Priority, "normal"),
		Budget:            body.Budget,
		EstimatedDuration: body.EstimatedDuration,
		LocationLat:       &body.LocationLat,
		LocationLng:       &body.LocationLng,
		LocationAddress:   body.LocationAddress,
		LocationCity:      body.LocationCity,
		Status:            models.RequestStatusScheduled,
		ScheduledFor:      &schedTime,
	}

	if err := database.DB.Create(&serviceRequest).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create scheduled request"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Scheduled service request created",
		"service_request": serviceRequest,
	})
}

func ifEmpty(s string, def string) string {
	if s == "" {
		return def
	}
	return s
}

// Worker Service Functions (exported for use in main.go)
func GetAvailableServiceRequests(c *gin.Context) {
	getAvailableServiceRequests(c)
}

func GetWorkerActiveRequests(c *gin.Context) {
	getWorkerActiveRequests(c)
}

func RespondToServiceRequest(c *gin.Context) {
	respondToServiceRequest(c)
}

func StartServiceRequest(c *gin.Context) {
	startServiceRequest(c)
}

func CompleteServiceRequest(c *gin.Context) {
	completeServiceRequest(c)
}

// createServiceRequest creates a new service request and broadcasts it to nearby workers
func createServiceRequest(c *gin.Context) {
	userID := c.GetUint("user_id")
	
	var req models.CustomerServiceRequestCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Validate location coordinates
	if !utils.IsLocationValid(req.LocationLat, req.LocationLng) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid location coordinates"})
		return
	}
	
	// Set expiration time (3 minutes from now)
	expiresAt := time.Now().Add(3 * time.Minute)
	
	// Create service request
	serviceRequest := models.CustomerServiceRequest{
		CustomerID:        userID,
		CategoryID:        req.CategoryID,
		ServiceOptionID:   req.ServiceOptionID, // New: Include service option ID
		Title:             req.Title,
		Description:       req.Description,
		Priority:          req.Priority,
		Budget:            req.Budget,
		EstimatedDuration: req.EstimatedDuration,
		LocationLat:       &req.LocationLat,
		LocationLng:       &req.LocationLng,
		LocationAddress:   req.LocationAddress,
		LocationCity:      req.LocationCity,
		Status:            models.RequestStatusBroadcast,
		ExpiresAt:         &expiresAt,
	}
	
	if err := database.DB.Create(&serviceRequest).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create service request"})
		return
	}
	
	// Broadcast to nearby workers
	go broadcastServiceRequest(serviceRequest)
	
	// Track analytics for all workers in this category (they received a job opportunity)
	analyticsService := services.NewWorkerAnalyticsService()
	var workersInCategory []models.WorkerProfile
	if err := database.DB.Where("category_id = ? AND is_active = ?", req.CategoryID, true).Find(&workersInCategory).Error; err == nil {
		for _, worker := range workersInCategory {
			if err := analyticsService.TrackJobReceived(worker.ID, serviceRequest.ID); err != nil {
				log.Printf("⚠️ Failed to track job received analytics for worker %d: %v", worker.ID, err)
			}
		}
	}
	
	c.JSON(http.StatusCreated, gin.H{
		"message": "Service request created successfully",
		"service_request": serviceRequest,
	})
}

// getMyServiceRequests returns all service requests created by the current user
func getMyServiceRequests(c *gin.Context) {
	userID := c.GetUint("user_id")
	
	var serviceRequests []models.CustomerServiceRequest
	if err := database.DB.Where("customer_id = ?", userID).
		Preload("AssignedWorker.User").
		Preload("Category").
		Preload("ServiceOption"). // New: Preload service option details
		Order("created_at DESC").
		Find(&serviceRequests).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch service requests"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"service_requests": serviceRequests,
		"total_count": len(serviceRequests),
	})
}

// getServiceRequest returns a specific service request by ID
func getServiceRequest(c *gin.Context) {
	requestID := c.Param("id")
	userID := c.GetUint("user_id")
	
	var serviceRequest models.CustomerServiceRequest
	if err := database.DB.Where("id = ?", requestID).
		Preload("Customer").
		Preload("AssignedWorker.User").
		Preload("AssignedWorker.Category").
		Preload("Category").
		Preload("ServiceOption"). // New: Preload service option details
		First(&serviceRequest).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service request not found"})
		return
	}
	
	// Check if user has access to this request
	if serviceRequest.CustomerID != userID {
		// Check if user is the assigned worker
		if serviceRequest.AssignedWorkerID == nil || *serviceRequest.AssignedWorkerID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}
	}
	
	c.JSON(http.StatusOK, gin.H{
		"service_request": serviceRequest,
	})
}

// getAvailableServiceRequests returns available service requests for workers
func getAvailableServiceRequests(c *gin.Context) {
	userID := c.GetUint("user_id")
	
	log.Printf("🔍 getAvailableServiceRequests called for user %d", userID)
	
	// Get worker profile
	var workerProfile models.WorkerProfile
	if err := database.DB.Where("user_id = ?", userID).First(&workerProfile).Error; err != nil {
		log.Printf("❌ Worker profile not found for user %d: %v", userID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Worker profile not found"})
		return
	}
	
	log.Printf("🔍 Worker profile loaded: ID=%d, CategoryID=%d, IsAvailable=%v", 
		workerProfile.ID, workerProfile.CategoryID, workerProfile.IsAvailable)
	
	// Check if worker is available
	if !workerProfile.IsAvailable {
		log.Printf("❌ Worker %d is not available", workerProfile.ID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Worker is not available"})
		return
	}

	// Check if worker has active work (only in-progress requests should block new requests)
	var activeRequestCount int64
	if err := database.DB.Model(&models.CustomerServiceRequest{}).
		Where("assigned_worker_id = ? AND status = ?", 
			workerProfile.ID, 
			models.RequestStatusInProgress).
		Count(&activeRequestCount).Error; err != nil {
		log.Printf("❌ Failed to check active requests for worker %d: %v", workerProfile.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check active requests"})
		return
	}

	log.Printf("🔍 Worker %d has %d in-progress requests", workerProfile.ID, activeRequestCount)

	if activeRequestCount > 0 {
		log.Printf("❌ Worker %d has active in-progress work and cannot accept new requests", workerProfile.ID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Worker has active in-progress work and cannot accept new requests"})
		return
	}
	
	// Check if worker has recent location data (optional for now)
	hasLocationData := workerProfile.CurrentLat != nil && workerProfile.CurrentLng != nil && utils.IsLocationRecent(workerProfile.LastLocationUpdate)
	log.Printf("🔍 Worker %d has location data: %v (lat=%v, lng=%v)", 
		workerProfile.ID, hasLocationData, workerProfile.CurrentLat, workerProfile.CurrentLng)
	
	// Get available service requests in worker's category
	var serviceRequests []models.CustomerServiceRequest
	if err := database.DB.Where("category_id = ? AND status = ? AND assigned_worker_id IS NULL", 
		workerProfile.CategoryID, models.RequestStatusBroadcast).
		Preload("Customer").
		Preload("Category").
		Preload("ServiceOption").
		Find(&serviceRequests).Error; err != nil {
		log.Printf("❌ Failed to fetch service requests for category %d: %v", workerProfile.CategoryID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch service requests"})
		return
	}
	
	log.Printf("🔍 Found %d broadcast requests in category %d", len(serviceRequests), workerProfile.CategoryID)
	
	// Filter requests by distance and add distance information
	var availableRequests []gin.H
	for _, request := range serviceRequests {
		if hasLocationData {
			distance := utils.HaversineDistance(
				*workerProfile.CurrentLat, *workerProfile.CurrentLng,
				*request.LocationLat, *request.LocationLng,
			)
			
			// Use a default broadcast radius of 10km if not specified
			broadcastRadius := 10.0
			
			if distance <= broadcastRadius {
				eta := utils.CalculateETA(
					utils.Location{Latitude: *workerProfile.CurrentLat, Longitude: *workerProfile.CurrentLng},
					utils.Location{Latitude: *request.LocationLat, Longitude: *request.LocationLng},
					30.0, // Assume average speed of 30 km/h
				)
				
				// Get customer name separately to avoid preload issues
				var customer models.User
				var customerName string
				if err := database.DB.First(&customer, request.CustomerID).Error; err == nil {
					customerName = customer.FullName
				} else {
					customerName = "Unknown Customer"
				}
				
				availableRequests = append(availableRequests, gin.H{
					"id": request.ID,
					"title": request.Title,
					"description": request.Description,
					"category_id": request.CategoryID,
					"service_option_id": request.ServiceOptionID,
					"location_address": request.LocationAddress,
					"location_city": request.LocationCity,
					"location_lat": request.LocationLat,
					"location_lng": request.LocationLng,
					"priority": request.Priority,
					"budget": request.Budget,
					"estimated_duration": request.EstimatedDuration,
					"distance": distance,
					"eta_minutes": int(eta.Minutes()),
					"customer_name": customerName,
					"created_at": request.CreatedAt,
					"status": request.Status,
				})
			}
		} else {
			// For workers without location data, show all requests in their category
			// Get customer name separately to avoid preload issues
			var customer models.User
			var customerName string
			if err := database.DB.First(&customer, request.CustomerID).Error; err == nil {
				customerName = customer.FullName
			} else {
				customerName = "Unknown Customer"
			}
			
			availableRequests = append(availableRequests, gin.H{
				"id": request.ID,
				"title": request.Title,
				"description": request.Description,
				"category_id": request.CategoryID,
				"service_option_id": request.ServiceOptionID,
				"location_address": request.LocationAddress,
				"location_city": request.LocationCity,
				"location_lat": request.LocationLat,
				"location_lng": request.LocationLng,
				"priority": request.Priority,
				"budget": request.Budget,
				"estimated_duration": request.EstimatedDuration,
				"distance": nil,
				"eta_minutes": nil,
				"customer_name": customerName,
				"created_at": request.CreatedAt,
				"status": request.Status,
			})
		}
	}
	
	log.Printf("✅ Returning %d available requests for worker %d", len(availableRequests), workerProfile.ID)
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"available_requests": availableRequests,
		"total_count": len(availableRequests),
		"worker_category": workerProfile.CategoryID,
	})
}

// getWorkerActiveRequests returns active requests assigned to the worker
func getWorkerActiveRequests(c *gin.Context) {
	userID := c.GetUint("user_id")
	
	// Get worker profile
	var workerProfile models.WorkerProfile
	if err := database.DB.Where("user_id = ?", userID).First(&workerProfile).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Worker profile not found"})
		return
	}
	
	// Get active requests (accepted and in-progress)
	var serviceRequests []models.CustomerServiceRequest
	if err := database.DB.Where(
		"assigned_worker_id = ? AND status IN (?, ?)", 
		workerProfile.ID, 
		models.RequestStatusAccepted,
		models.RequestStatusInProgress,
	).
	Order("created_at DESC").
	Find(&serviceRequests).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch active requests"})
		return
	}
	
	// Format response
	var activeRequests []gin.H
	for _, request := range serviceRequests {
		// Get customer name separately to avoid preload issues
		var customer models.User
		var customerName string
		if err := database.DB.First(&customer, request.CustomerID).Error; err == nil {
			customerName = customer.FullName
		} else {
			customerName = "Unknown Customer"
		}
		
		activeRequests = append(activeRequests, gin.H{
			"id": request.ID,
			"title": request.Title,
			"description": request.Description,
			"location_address": request.LocationAddress,
			"location_city": request.LocationCity,
			"priority": request.Priority,
			"budget": request.Budget,
			"estimated_duration": request.EstimatedDuration,
			"status": request.Status,
			"started_at": request.StartedAt,
			"completed_at": request.CompletedAt,
			"customer_name": customerName,
			"created_at": request.CreatedAt,
		})
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"active_requests": activeRequests,
		"total_count": len(activeRequests),
	})
}

// respondToServiceRequest allows workers to respond to service requests
func respondToServiceRequest(c *gin.Context) {
	requestID := c.Param("id")
	userID := c.GetUint("user_id")
	
	var req models.WorkerResponseCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Get service request
	var serviceRequest models.CustomerServiceRequest
	if err := database.DB.Where("id = ?", requestID).First(&serviceRequest).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service request not found"})
		return
	}
	
	// Check if request is still available
	if serviceRequest.Status != models.RequestStatusBroadcast {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Service request is no longer available"})
		return
	}
	
	// Get worker profile
	var workerProfile models.WorkerProfile
	if err := database.DB.Where("user_id = ?", userID).First(&workerProfile).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Worker profile not found"})
		return
	}
	
	// Check if worker category matches
	if workerProfile.CategoryID != serviceRequest.CategoryID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Service category does not match worker's category"})
		return
	}
	
	// Calculate distance
	var distance float64
	if workerProfile.CurrentLat != nil && workerProfile.CurrentLng != nil && serviceRequest.LocationLat != nil && serviceRequest.LocationLng != nil {
		distance = utils.HaversineDistance(
			*workerProfile.CurrentLat, *workerProfile.CurrentLng,
			*serviceRequest.LocationLat, *serviceRequest.LocationLng,
		)
	}
	
	// Create worker response
	workerResponse := models.WorkerResponse{
		ServiceRequestID: serviceRequest.ID,
		WorkerID:         workerProfile.ID,
		Response:         req.Response,
		Message:          req.Message,
		ProposedPrice:   req.ProposedPrice,
		ProposedTime:    req.ProposedTime,
		Distance:         distance,
		RespondedAt:      time.Now(),
	}
	
	if err := database.DB.Create(&workerResponse).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create response"})
		return
	}
	
	// If worker accepts, assign them to the request
	if req.Response == "accept" {
		serviceRequest.Status = models.RequestStatusAccepted
		serviceRequest.AssignedWorkerID = &workerProfile.ID
		
		if err := database.DB.Save(&serviceRequest).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to assign worker"})
			return
		}
		
		// Send notification to customer about acceptance
		if err := SendServiceStatusNotification(serviceRequest.CustomerID, serviceRequest.ID, "accepted"); err != nil {
			log.Printf("⚠️ Failed to send acceptance notification: %v", err)
		}
		
		// Track analytics for job response
		analyticsService := services.NewWorkerAnalyticsService()
		responseTime := time.Since(serviceRequest.CreatedAt).Minutes()
		
		if err := analyticsService.TrackJobResponse(workerProfile.ID, serviceRequest.ID, responseTime); err != nil {
			log.Printf("⚠️ Failed to track job response analytics: %v", err)
			// Don't fail the response, just log the error
		}
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Response submitted successfully",
		"response": workerResponse,
		"request_status": serviceRequest.Status,
	})
}

// Worker response to service request
func workerRespondToRequest(c *gin.Context) {
	// Get worker profile
	workerID := c.GetUint("user_id")
	var workerProfile models.WorkerProfile
	if err := database.DB.Where("user_id = ?", workerID).First(&workerProfile).Error; err != nil {
		log.Printf("❌ Worker profile not found for user %d: %v", workerID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Worker profile not found"})
		return
	}

	log.Printf("🔍 Worker profile found: ID=%d, UserID=%d, CategoryID=%d", 
		workerProfile.ID, workerProfile.UserID, workerProfile.CategoryID)

	// Parse request
	var req struct {
		Response       string  `json:"response" binding:"required,oneof=accept decline"`
		Message        string  `json:"message"`
		ProposedPrice *float64 `json:"proposed_price"`
		ProposedTime  string  `json:"proposed_time"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("❌ JSON binding error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	log.Printf("🔍 Worker %d responding to request with: %s", workerID, req.Response)

	// Get service request ID from URL
	requestID := c.Param("id")
	if requestID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Service request ID is required"})
		return
	}

	// Parse request ID
	requestIDInt, err := strconv.Atoi(requestID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid service request ID"})
		return
	}

	// Get service request
	var serviceRequest models.CustomerServiceRequest
	if err := database.DB.Preload("Customer").First(&serviceRequest, requestIDInt).Error; err != nil {
		log.Printf("❌ Service request %d not found: %v", requestIDInt, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Service request not found"})
		return
	}

	log.Printf("🔍 Service request %d found: status=%s, category_id=%d", 
		requestIDInt, serviceRequest.Status, serviceRequest.CategoryID)

	// Check if request is still available
	if serviceRequest.Status != models.RequestStatusBroadcast {
		log.Printf("❌ Service request %d status is %s, expected %s", 
			requestIDInt, serviceRequest.Status, models.RequestStatusBroadcast)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Service request is no longer available"})
		return
	}

	// Check if worker category matches
	if workerProfile.CategoryID != serviceRequest.CategoryID {
		log.Printf("❌ Worker category %d does not match service request category %d", 
			workerProfile.CategoryID, serviceRequest.CategoryID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Worker category does not match service request category"})
		return
	}

	// Handle response
	if req.Response == "accept" {
		log.Printf("✅ Worker %d accepting service request %d", workerID, requestIDInt)
		
		// Update service request status to accepted
		serviceRequest.Status = models.RequestStatusAccepted
		serviceRequest.AssignedWorkerID = &workerProfile.ID
		
		if err := database.DB.Save(&serviceRequest).Error; err != nil {
			log.Printf("❌ Failed to update service request %d: %v", requestIDInt, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update service request"})
			return
		}
		
		log.Printf("✅ Service request %d assigned to worker %d (profile ID: %d)", 
			requestIDInt, workerID, workerProfile.ID)
		
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Service request accepted successfully",
			"request_status": serviceRequest.Status,
			"assigned_worker_id": serviceRequest.AssignedWorkerID,
		})
	} else {
		log.Printf("❌ Worker %d declining service request %d", workerID, requestIDInt)
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Response submitted successfully",
		})
	}
}

// Helper function to broadcast service request to nearby workers
func broadcastServiceRequest(serviceRequest models.CustomerServiceRequest) {
	// Update status to broadcast
	serviceRequest.Status = models.RequestStatusBroadcast
	
	if err := database.DB.Save(&serviceRequest).Error; err != nil {
		log.Printf("❌ Failed to update service request status: %v", err)
		return
	}
	
	log.Printf("📡 Broadcasting service request %d to category %d workers", 
		serviceRequest.ID, serviceRequest.CategoryID)
	
	// Send real-time WebSocket notification to workers
	broadcastServiceRequestViaWebSocket(serviceRequest)
	
	// Find available workers in the same category within broadcast radius
	// Exclude workers who are already working on other requests
	var availableWorkers []models.WorkerProfile
	err := database.DB.Where(
		"category_id = ? AND is_available = ? AND current_lat IS NOT NULL AND current_lng IS NOT NULL AND id NOT IN (SELECT DISTINCT assigned_worker_id FROM customer_service_requests WHERE assigned_worker_id IS NOT NULL AND status IN (?, ?))",
		serviceRequest.CategoryID, true, models.RequestStatusAccepted, models.RequestStatusInProgress,
	).Preload("User").Find(&availableWorkers).Error
	
	if err != nil {
		log.Printf("❌ Failed to find available workers: %v", err)
		return
	}
	
	log.Printf("👷 Found %d available category workers", len(availableWorkers))
	
	// If no workers found, let's check what's in the database
	if len(availableWorkers) == 0 {
		log.Printf("🔍 No workers found. Let's check what workers exist:")
		
		// Check all workers in this category
		var allWorkersInCategory []models.WorkerProfile
		if err := database.DB.Where("category_id = ?", serviceRequest.CategoryID).Find(&allWorkersInCategory).Error; err == nil {
			log.Printf("📊 Total workers in category %d: %d", serviceRequest.CategoryID, len(allWorkersInCategory))
			for _, w := range allWorkersInCategory {
				log.Printf("👷 Worker %d: available=%v, has_location=%v, lat=%v, lng=%v", 
					w.ID, w.IsAvailable, w.CurrentLat != nil && w.CurrentLng != nil, w.CurrentLat, w.CurrentLng)
			}
		}
		
		// Check all available workers regardless of category
		var allAvailableWorkers []models.WorkerProfile
		if err := database.DB.Where("is_available = ?", true).Find(&allAvailableWorkers).Error; err == nil {
			log.Printf("📊 Total available workers: %d", len(allAvailableWorkers))
			for _, w := range allAvailableWorkers {
				log.Printf("👷 Available Worker %d: category_id=%d, has_location=%v", 
					w.ID, w.CategoryID, w.CurrentLat != nil && w.CurrentLng != nil)
			}
		}
	}
	
	// Filter workers by distance and notify them
	for _, worker := range availableWorkers {
		if worker.CurrentLat != nil && worker.CurrentLng != nil && serviceRequest.LocationLat != nil && serviceRequest.LocationLng != nil {
			distance := utils.HaversineDistance(
				*worker.CurrentLat, *worker.CurrentLng,
				*serviceRequest.LocationLat, *serviceRequest.LocationLng,
			)
			
			// Check if worker is within broadcast radius (default 10km)
			broadcastRadius := 10.0
			if distance <= broadcastRadius {
				log.Printf("📱 Notifying worker %d (distance: %.2f km)", worker.ID, distance)
				
				// Send real-time WebSocket notification
				notifyWorkerViaWebSocket(worker, serviceRequest, distance)
			}
		}
	}
}

// notifyWorker sends notification to a specific worker
func notifyWorker(worker models.WorkerProfile, request models.CustomerServiceRequest, distance float64) {
	// TODO: Implement actual notification system
	// This could be:
	// 1. Push notification via Firebase/Expo
	// 2. WebSocket notification
	// 3. SMS notification
	// 4. In-app notification
	
	log.Printf("🔔 Worker %d (%s) notified about request %d (%.2f km away)", 
		worker.ID, worker.User.FullName, request.ID, distance)
	
	// TODO: Send push notification with sound
	// TODO: Update worker's dashboard in real-time
}

// Helper function to broadcast service request to nearby workers via WebSocket
func broadcastServiceRequestViaWebSocket(serviceRequest models.CustomerServiceRequest) {
	// Call the global broadcast function from main.go
	// Note: This requires importing the main package, which creates import cycles
	// For now, we'll use a different approach - direct WebSocket broadcasting
	log.Printf("📡 Service request %d would be broadcasted via WebSocket to all connected workers", serviceRequest.ID)
	log.Printf("📡 Service request details: Title='%s', Category=%d, Location='%s, %s'", 
		serviceRequest.Title, serviceRequest.CategoryID, serviceRequest.LocationCity, serviceRequest.LocationAddress)
	
	// TODO: Implement direct WebSocket broadcasting when the hub is properly integrated
	// This will send real-time notifications to workers like Deliveroo/Glovo
}

// Helper function to notify a worker via WebSocket
func notifyWorkerViaWebSocket(worker models.WorkerProfile, request models.CustomerServiceRequest, distance float64) {
	// This function will be implemented when the WebSocket hub is properly integrated
	// For now, it just logs the notification
	log.Printf("📱 Notifying worker %d (distance: %.2f km) via WebSocket", worker.ID, distance)
}

// Additional helper functions for request management
func updateServiceRequestStatus(c *gin.Context) {
	// Implementation for updating request status
	c.JSON(http.StatusOK, gin.H{"message": "Status updated"})
}

func cancelServiceRequest(c *gin.Context) {
	// Implementation for canceling requests
	c.JSON(http.StatusOK, gin.H{"message": "Request cancelled"})
}

func reviewService(c *gin.Context) {
	// Implementation for rating and reviewing services
	c.JSON(http.StatusOK, gin.H{"message": "Review submitted"})
}

func startServiceRequest(c *gin.Context) {
	requestID := c.Param("id")
	userID := c.GetUint("user_id")
	
	log.Printf("🔄 Worker %d attempting to start work on request %s", userID, requestID)
	
	// Get worker profile for this user
	var workerProfile models.WorkerProfile
	if err := database.DB.Where("user_id = ?", userID).First(&workerProfile).Error; err != nil {
		log.Printf("❌ Worker profile not found for user %d: %v", userID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Worker profile not found"})
		return
	}
	
	log.Printf("🔍 Worker profile found: ID=%d, UserID=%d", workerProfile.ID, workerProfile.UserID)
	
	// Get service request
	var serviceRequest models.CustomerServiceRequest
	if err := database.DB.Where("id = ?", requestID).First(&serviceRequest).Error; err != nil {
		log.Printf("❌ Service request %s not found: %v", requestID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Service request not found"})
		return
	}

	// Optionally capture agreed price from body (non-fatal if missing)
	var body struct {
		AgreedPrice *float64 `json:"agreed_price"`
	}
	_ = c.ShouldBindJSON(&body)
	
	log.Printf("🔍 Service request %s found: status=%s, assigned_worker_id=%v", 
		requestID, serviceRequest.Status, serviceRequest.AssignedWorkerID)
	
	// Check if request is assigned to this worker (compare with worker profile ID)
	if serviceRequest.AssignedWorkerID == nil {
		log.Printf("❌ Service request %s has no assigned worker", requestID)
		c.JSON(http.StatusForbidden, gin.H{"error": "Service request is not assigned to any worker"})
		return
	}
	
	if *serviceRequest.AssignedWorkerID != workerProfile.ID {
		log.Printf("❌ Worker profile %d not assigned to request %s (assigned to %d)", 
			workerProfile.ID, requestID, *serviceRequest.AssignedWorkerID)
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not assigned to this request"})
		return
	}
	
	// Check if request is in accepted status
	if serviceRequest.Status != models.RequestStatusAccepted {
		log.Printf("❌ Service request %s status is %s, expected %s", 
			requestID, serviceRequest.Status, models.RequestStatusAccepted)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Service request is not in accepted status"})
		return
	}
	
	// Update status to in progress
	now := time.Now()
	serviceRequest.Status = models.RequestStatusInProgress
	serviceRequest.StartedAt = &now
	if body.AgreedPrice != nil {
		serviceRequest.Budget = body.AgreedPrice
	}
	
	if err := database.DB.Save(&serviceRequest).Error; err != nil {
		log.Printf("❌ Failed to update service request %s: %v", requestID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start service request"})
		return
	}
	
	// Send notification to customer about work starting
	if err := SendServiceStatusNotification(serviceRequest.CustomerID, serviceRequest.ID, "in_progress"); err != nil {
		log.Printf("⚠️ Failed to send work started notification: %v", err)
	}
	
	log.Printf("✅ Worker %d (profile %d) started work on service request %s", userID, workerProfile.ID, requestID)
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Work started successfully",
		"request_status": serviceRequest.Status,
		"started_at": serviceRequest.StartedAt,
		"agreed_price": serviceRequest.Budget,
	})
}

func completeServiceRequest(c *gin.Context) {
	requestID := c.Param("id")
	userID := c.GetUint("user_id")
	
	// Get worker profile for this user
	var workerProfile models.WorkerProfile
	if err := database.DB.Where("user_id = ?", userID).First(&workerProfile).Error; err != nil {
		log.Printf("❌ Worker profile not found for user %d: %v", userID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Worker profile not found"})
		return
	}
	
	// Get service request
	var serviceRequest models.CustomerServiceRequest
	if err := database.DB.Where("id = ?", requestID).First(&serviceRequest).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service request not found"})
		return
	}
	
	// Check if request is assigned to this worker (compare with worker profile ID)
	if serviceRequest.AssignedWorkerID == nil || *serviceRequest.AssignedWorkerID != workerProfile.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not assigned to this request"})
		return
	}
	
	// Check if request is in progress
	if serviceRequest.Status != models.RequestStatusInProgress {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Service request is not in progress"})
		return
	}
	
	// Update status to completed
	now := time.Now()
	serviceRequest.Status = models.RequestStatusCompleted
	serviceRequest.CompletedAt = &now
	
	if err := database.DB.Save(&serviceRequest).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete service request"})
		return
	}
	
	// Automatically create service history entry
	historyData := models.ServiceHistoryCreate{
		ServiceRequestID: serviceRequest.ID,
		WorkerID:         workerProfile.ID,
		ActualDuration:   nil, // Worker can update this later
		AgreedPrice:      serviceRequest.Budget, // Use budget as agreed price
		FinalPrice:       serviceRequest.Budget, // Use budget as final price
		PaymentStatus:    "pending",
		WorkerNotes:      "",
		CustomerNotes:    "",
	}
	
	// Create service history
	history := models.ServiceHistory{
		ServiceRequestID:  historyData.ServiceRequestID,
		WorkerID:          historyData.WorkerID,
		CustomerID:        serviceRequest.CustomerID,
		CategoryID:        serviceRequest.CategoryID,
		ServiceOptionID:   serviceRequest.ServiceOptionID,
		Title:             serviceRequest.Title,
		Description:       serviceRequest.Description,
		Priority:          serviceRequest.Priority,
		Budget:            serviceRequest.Budget,
		EstimatedDuration: serviceRequest.EstimatedDuration,
		ActualDuration:    historyData.ActualDuration,
		LocationAddress:   serviceRequest.LocationAddress,
		LocationCity:      serviceRequest.LocationCity,
		LocationLat:       serviceRequest.LocationLat,
		LocationLng:       serviceRequest.LocationLng,
		RequestCreatedAt:  serviceRequest.CreatedAt,
		AssignedAt:        nil, // Will be set when worker accepts
		StartedAt:         serviceRequest.StartedAt,
		CompletedAt:       *serviceRequest.CompletedAt,
		AgreedPrice:       historyData.AgreedPrice,
		FinalPrice:        historyData.FinalPrice,
		PaymentStatus:     historyData.PaymentStatus,
		WorkerNotes:       historyData.WorkerNotes,
		CustomerNotes:     historyData.CustomerNotes,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	
	if err := database.DB.Create(&history).Error; err != nil {
		log.Printf("⚠️ Failed to create service history for request %d: %v", serviceRequest.ID, err)
		// Don't fail the completion, just log the error
	} else {
		log.Printf("✅ Service history created for completed request %d", serviceRequest.ID)
	}
	
	// Update worker profile statistics
	if err := updateWorkerServiceStats(workerProfile.ID); err != nil {
		log.Printf("⚠️ Failed to update worker stats for worker %d: %v", workerProfile.ID, err)
		// Don't fail the completion, just log the error
	}
	
	// Track analytics for worker performance
	analyticsService := services.NewWorkerAnalyticsService()
	
	// Handle budget conversion (it's a pointer)
	var earnings float64
	if serviceRequest.Budget != nil {
		earnings = *serviceRequest.Budget
	}
	
	// Handle duration conversion (it's a string, convert to float)
	var workHours float64
	if duration, err := strconv.ParseFloat(serviceRequest.EstimatedDuration, 64); err == nil {
		workHours = duration / 60.0 // Convert minutes to hours
	}
	
	if err := analyticsService.TrackJobCompletion(workerProfile.ID, serviceRequest.ID, earnings, workHours); err != nil {
		log.Printf("⚠️ Failed to track job completion analytics: %v", err)
		// Don't fail the completion, just log the error
	}
	
	// Send notification to customer about completion
	if err := SendServiceStatusNotification(serviceRequest.CustomerID, serviceRequest.ID, "completed"); err != nil {
		log.Printf("⚠️ Failed to send completion notification: %v", err)
	}

	// Send feedback request notification to customer after first completion
	var customerCompleted int64
	database.DB.Model(&models.ServiceHistory{}).Where("customer_id = ?", serviceRequest.CustomerID).Count(&customerCompleted)
	if customerCompleted == 1 {
		customerFeedbackData := map[string]interface{}{
			"action": "feedback_request",
			"role": "customer",
			"service_request_id": serviceRequest.ID,
		}
		if err := SendPushNotification(serviceRequest.CustomerID,
			"We value your feedback",
			"Your first service is complete! Please share your feedback to help us improve.",
			"feedback_request",
			customerFeedbackData); err != nil {
			log.Printf("⚠️ Failed to send customer feedback request notification: %v", err)
		} else {
			log.Printf("✅ Feedback request notification sent to customer %d", serviceRequest.CustomerID)
		}
	}

	// Send feedback request notification to worker after first completion
	var completedJobs int64
	database.DB.Model(&models.ServiceHistory{}).Where("worker_id = ?", workerProfile.ID).Count(&completedJobs)
	
	if completedJobs == 1 { // First job completion
		feedbackData := map[string]interface{}{
			"action": "feedback_request",
			"worker_id": workerProfile.ID,
			"service_request_id": serviceRequest.ID,
		}
		
		if err := SendPushNotification(userID, 
			"Help Us Improve Your Experience", 
			"Your first job is complete! Please share your feedback to help us enhance your experience.", 
			"feedback_request", 
			feedbackData); err != nil {
			log.Printf("⚠️ Failed to send feedback request notification: %v", err)
		} else {
			log.Printf("✅ Feedback request notification sent to worker %d", userID)
		}
	}
	
	log.Printf("✅ Worker %d (profile %d) completed service request %d", userID, workerProfile.ID, serviceRequest.ID)
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Work completed successfully. Service history created. Customer can now rate your service.",
		"request_status": serviceRequest.Status,
		"completed_at": serviceRequest.CompletedAt,
		"service_history_id": history.ID,
	})
}

// GetScheduledServiceRequests - Get scheduled service requests for workers
func GetScheduledServiceRequests(c *gin.Context) {
	userID := c.GetUint("user_id")
	
	// Get worker profile
	var workerProfile models.WorkerProfile
	if err := database.DB.Where("user_id = ?", userID).First(&workerProfile).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "Worker profile not found",
		})
		return
	}
	
	// Get scheduled requests for this worker's category
	var scheduledRequests []models.CustomerServiceRequest
	query := database.DB.Where("category_id = ? AND status = ? AND scheduled_for IS NOT NULL", 
		workerProfile.CategoryID, "scheduled").
		Where("scheduled_for > NOW()"). // Only future scheduled requests
		Order("scheduled_for ASC")
	
	if err := query.Find(&scheduledRequests).Error; err != nil {
		log.Printf("❌ Error fetching scheduled requests: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to fetch scheduled requests",
		})
		return
	}
	
	// Get customer names for each request
	var responseData []gin.H
	for _, request := range scheduledRequests {
		var customer models.User
		if err := database.DB.Where("id = ?", request.CustomerID).First(&customer).Error; err != nil {
			log.Printf("⚠️ Failed to fetch customer for request %d: %v", request.ID, err)
			continue
		}
		
		responseData = append(responseData, gin.H{
			"id": request.ID,
			"title": request.Title,
			"description": request.Description,
			"category_id": request.CategoryID,
			"location_address": request.LocationAddress,
			"location_city": request.LocationCity,
			"location_lat": request.LocationLat,
			"location_lng": request.LocationLng,
			"priority": request.Priority,
			"budget": request.Budget,
			"estimated_duration": request.EstimatedDuration,
			"customer_name": customer.FullName,
			"created_at": request.CreatedAt,
			"status": request.Status,
			"scheduled_for": request.ScheduledFor,
		})
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"scheduled_requests": responseData,
		"total_count": len(responseData),
		"message": "Scheduled requests fetched successfully",
	})
}


