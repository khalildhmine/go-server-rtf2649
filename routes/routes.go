// package routes

// import (
// 	"net/http"
// 	"strconv"

// 	"github.com/gin-gonic/gin"
// 	"gorm.io/gorm"

// 	"repair-service-server/database"
// 	"repair-service-server/middleware"
// 	"repair-service-server/models"
// )

// // RegisterRoutes registers all API routes
// func RegisterRoutes(router *gin.Engine) {
// 	// CORS middleware
// 	router.Use(middleware.CORSMiddleware())

// 	// Health check
// 	router.GET("/health", func(c *gin.Context) {
// 		c.JSON(200, gin.H{"status": "ok"})
// 	})

// 	// API v1 routes
// 	apiV1 := router.Group("/api/v1")
// 	{
// 		// Auth routes
// 		RegisterAuthRoutes(apiV1)

// 		// Address routes
// 		RegisterAddressRoutes(apiV1)

// 		// Service routes
// 		RegisterServiceRoutes(apiV1)

// 		// Worker routes
// 		RegisterWorkerRoutes(apiV1)

// 		// Location routes (NEW)
// 		RegisterLocationRoutes(apiV1)

// 		// Service Request routes (NEW)
// 		RegisterServiceRequestRoutes(apiV1)
// 	}
// }

// // RegisterAuthRoutes registers authentication routes
// func RegisterAuthRoutes(router *gin.RouterGroup) {
// 	auth := router.Group("/auth")
// 	{
// 		auth.POST("/signup", signUp)
// 		auth.POST("/signin", signIn)
// 		auth.POST("/refresh", refreshToken)
// 		auth.GET("/me", middleware.AuthMiddleware(), getCurrentUser)
// 		auth.PUT("/profile", middleware.AuthMiddleware(), updateUserProfile)
// 	}
// }

// // RegisterAddressRoutes registers address management routes
// func RegisterAddressRoutes(router *gin.RouterGroup) {
// 	addresses := router.Group("/addresses")
// 	addresses.Use(middleware.AuthMiddleware())
// 	{
// 		addresses.GET("/", getAddresses)
// 		addresses.POST("/", createAddress)
// 		addresses.GET("/:id", getAddress)
// 		addresses.PUT("/:id", updateAddress)
// 		addresses.DELETE("/:id", deleteAddress)
// 		addresses.POST("/:id/set-default", setDefaultAddress)
// 	}
// }

// // RegisterServiceRoutes registers service management routes
// func RegisterServiceRoutes(router *gin.RouterGroup) {
// 	services := router.Group("/services")
// 	{
// 		services.GET("/", getAllServices)
// 		services.GET("/:id", getService)
// 		services.POST("/", middleware.AuthMiddleware(), createService)
// 		services.PUT("/:id", middleware.AuthMiddleware(), updateService)
// 		services.DELETE("/:id", middleware.AuthMiddleware(), deleteService)
// 	}
// }

// // RegisterWorkerRoutes registers worker profile routes
// func RegisterWorkerRoutes(router *gin.RouterGroup) {
// 	workers := router.Group("/workers")
// 	{
// 		// Public routes
// 		workers.GET("/categories", getWorkerCategories)
// 		workers.GET("/available", getAvailableWorkers)
// 		workers.GET("/:id", getWorkerProfile)

// 		// Protected routes
// 		protected := workers.Group("/")
// 		protected.Use(middleware.AuthMiddleware())
// 		{
// 			protected.GET("/profile", getMyWorkerProfile)
// 			protected.POST("/profile", createWorkerProfile)
// 			protected.PUT("/profile", updateWorkerProfile)
// 			protected.POST("/availability", updateAvailability)
// 			protected.POST("/photos", uploadWorkerPhotos)
// 		}
// 	}
// }

// // Placeholder handlers - these will be implemented in separate files
// func getUserProfile(c *gin.Context) {
// 	// Get user ID from JWT token (set by AuthMiddleware)
// 	userID := c.GetUint("user_id")

// 	// Get user from database
// 	var user models.User
// 	if err := database.DB.First(&user, userID).Error; err != nil {
// 		c.JSON(http.StatusNotFound, gin.H{
// 			"error":   "User not found",
// 			"message": "The requested user does not exist",
// 		})
// 		return
// 	}

// 	// Return user profile (excluding sensitive fields like password_hash)
// 	c.JSON(http.StatusOK, gin.H{
// 		"message": "User profile retrieved successfully",
// 		"data": gin.H{
// 			"id":               user.ID,
// 			"full_name":        user.FullName,
// 			"phone_number":     user.PhoneNumber,
// 			"role":             user.Role,
// 			"profile_picture_url": user.ProfilePictureURL,
// 			"is_active":        user.IsActive,
// 			"created_at":       user.CreatedAt,
// 		},
// 	})
// }

// func updateUserProfile(c *gin.Context) {
// 	c.JSON(200, gin.H{"message": "Update user profile"})
// }

// func createBooking(c *gin.Context) {
// 	c.JSON(200, gin.H{"message": "Create booking"})
// }

// func getUserBookings(c *gin.Context) {
// 	c.JSON(200, gin.H{"message": "Get user bookings"})
// }

// func getBooking(c *gin.Context) {
// 	c.JSON(200, gin.H{"message": "Get booking"})
// }

// func cancelBooking(c *gin.Context) {
// 	c.JSON(200, gin.H{"message": "Cancel booking"})
// }

// func getWorkers(c *gin.Context) {
// 	c.JSON(200, gin.H{"message": "Get workers"})
// }

// func getWorker(c *gin.Context) {
// 	c.JSON(200, gin.H{"message": "Get worker"})
// }

// func getAvailableWorkers(c *gin.Context) {
// 	category := c.Query("category")
// 	city := c.Query("city")
// 	limit := 20

// 	if limitStr := c.Query("limit"); limitStr != "" {
// 		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
// 			limit = l
// 		}
// 	}

// 	query := database.DB.Preload("User").Where("is_available = ?", true)

// 	if category != "" {
// 		query = query.Where("category = ?", category)
// 	}

// 	if city != "" {
// 		query = query.Where("city ILIKE ?", "%"+city+"%")
// 	}

// 	var workers []models.WorkerProfile
// 	if err := query.Limit(limit).Find(&workers).Error; err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"success": false,
// 			"message": "Failed to fetch workers",
// 		})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"success": true,
// 		"workers": workers,
// 	})
// }

// func getWorkerCategories(c *gin.Context) {
// 	categories := models.GetWorkerCategories()
// 	c.JSON(http.StatusOK, gin.H{
// 		"success": true,
// 		"categories": categories,
// 	})
// }

// func getWorkerProfile(c *gin.Context) {
// 	workerID := c.Param("id")
// 	id, err := strconv.ParseUint(workerID, 10, 32)
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"success": false,
// 			"message": "Invalid worker ID",
// 		})
// 		return
// 	}

// 	var worker models.WorkerProfile
// 	if err := database.DB.Preload("User").First(&worker, id).Error; err != nil {
// 		if err == gorm.ErrRecordNotFound {
// 			c.JSON(http.StatusNotFound, gin.H{
// 				"success": false,
// 				"message": "Worker not found",
// 			})
// 			return
// 		}
// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"success": false,
// 			"message": "Failed to fetch worker profile",
// 		})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"success": true,
// 		"worker": worker,
// 	})
// }

// func getMyWorkerProfile(c *gin.Context) {
// 	userID := c.GetUint("user_id")

// 	var worker models.WorkerProfile
// 	if err := database.DB.Preload("User").Where("user_id = ?", userID).First(&worker).Error; err != nil {
// 		if err == gorm.ErrRecordNotFound {
// 			c.JSON(http.StatusNotFound, gin.H{
// 				"success": false,
// 				"message": "Worker profile not found",
// 			})
// 			return
// 		}
// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"success": false,
// 			"message": "Failed to fetch worker profile",
// 		})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"success": true,
// 		"worker": worker,
// 	})
// }

// func createWorkerProfile(c *gin.Context) {
// 	userID := c.GetUint("user_id")

// 	// Check if user already has a worker profile
// 	var existingWorker models.WorkerProfile
// 	if err := database.DB.Where("user_id = ?", userID).First(&existingWorker).Error; err == nil {
// 		c.JSON(http.StatusConflict, gin.H{
// 			"success": false,
// 			"message": "Worker profile already exists",
// 		})
// 		return
// 	}

// 	var request models.WorkerProfileRequest
// 	if err := c.ShouldBindJSON(&request); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"success": false,
// 			"message": "Invalid request data",
// 			"error":   err.Error(),
// 		})
// 		return
// 	}

// 	worker := models.WorkerProfile{
// 		UserID:            userID,
// 		Category:          request.Category,
// 		PhoneNumber:       request.PhoneNumber,
// 		Address:           request.Address,
// 		City:              request.City,
// 		State:             request.State,
// 		PostalCode:        request.PostalCode,
// 		Country:           request.Country,
// 		Latitude:          request.Latitude,
// 		Longitude:         request.Longitude,
// 		Experience:        request.Experience,
// 		Skills:            request.Skills,
// 		Bio:               request.Bio,
// 		HourlyRate:        request.HourlyRate,
// 		EmergencyContact:  request.EmergencyContact,
// 		EmergencyName:     request.EmergencyName,
// 		EmergencyRelation: request.EmergencyRelation,
// 		BankAccount:       request.BankAccount,
// 		BankName:          request.BankName,
// 		TaxID:             request.TaxID,
// 		InsuranceInfo:     request.InsuranceInfo,
// 		LicenseNumber:     request.LicenseNumber,
// 		LicenseExpiry:     request.LicenseExpiry,
// 	}

// 	if err := database.DB.Create(&worker).Error; err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"success": false,
// 			"message": "Failed to create worker profile",
// 		})
// 		return
// 	}

// 	// Load the user data
// 	database.DB.Preload("User").First(&worker, worker.ID)

// 	c.JSON(http.StatusCreated, gin.H{
// 		"success": true,
// 		"message": "Worker profile created successfully",
// 		"worker":  worker,
// 	})
// }

// func updateWorkerProfile(c *gin.Context) {
// 	userID := c.GetUint("user_id")

// 	var request models.WorkerProfileRequest
// 	if err := c.ShouldBindJSON(&request); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"success": false,
// 			"message": "Invalid request data",
// 			"error":   err.Error(),
// 		})
// 		return
// 	}

// 	var worker models.WorkerProfile
// 	if err := database.DB.Where("user_id = ?", userID).First(&worker).Error; err != nil {
// 		c.JSON(http.StatusNotFound, gin.H{
// 			"success": false,
// 			"message": "Worker profile not found",
// 		})
// 		return
// 	}

// 	// Update fields
// 	worker.Category = request.Category
// 	worker.PhoneNumber = request.PhoneNumber
// 	worker.Address = request.Address
// 	worker.City = request.City
// 	worker.State = request.State
// 	worker.PostalCode = request.PostalCode
// 	worker.Country = request.Country
// 	worker.Latitude = request.Latitude
// 	worker.Longitude = request.Longitude
// 	worker.Experience = request.Experience
// 	worker.Skills = request.Skills
// 	worker.Bio = request.Bio
// 	worker.HourlyRate = request.HourlyRate
// 	worker.EmergencyContact = request.EmergencyContact
// 	worker.EmergencyName = request.EmergencyName
// 	worker.EmergencyRelation = request.EmergencyRelation
// 	worker.BankAccount = request.BankAccount
// 	worker.BankName = request.BankName
// 	worker.TaxID = request.TaxID
// 	worker.InsuranceInfo = request.InsuranceInfo
// 	worker.LicenseNumber = request.LicenseNumber
// 	worker.LicenseExpiry = request.LicenseExpiry

// 	if err := database.DB.Save(&worker).Error; err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"success": false,
// 			"message": "Failed to update worker profile",
// 		})
// 		return
// 	}

// 	// Load the user data
// 	database.DB.Preload("User").First(&worker, worker.ID)

// 	c.JSON(http.StatusOK, gin.H{
// 		"success": true,
// 		"message": "Worker profile updated successfully",
// 		"worker":  worker,
// 	})
// }

// func updateAvailability(c *gin.Context) {
// 	userID := c.GetUint("user_id")

// 	var request struct {
// 		IsAvailable bool `json:"is_available" binding:"required"`
// 	}

// 	if err := c.ShouldBindJSON(&request); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"success": false,
// 			"message": "Invalid request data",
// 		})
// 		return
// 	}

// 	if err := database.DB.Model(&models.WorkerProfile{}).Where("user_id = ?", userID).Update("is_available", request.IsAvailable).Error; err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"success": false,
// 			"message": "Failed to update availability",
// 		})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"success": true,
// 		"message": "Availability updated successfully",
// 		"is_available": request.IsAvailable,
// 	})
// }

// func uploadWorkerPhotos(c *gin.Context) {
// 	userID := c.GetUint("user_id")

// 	var request struct {
// 		IDCardPhoto  string `json:"id_card_photo"`
// 		ProfilePhoto string `json:"profile_photo"`
// 	}

// 	if err := c.ShouldBindJSON(&request); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"success": false,
// 			"message": "Invalid request data",
// 		})
// 		return
// 	}

// 	updates := make(map[string]interface{})
// 	if request.IDCardPhoto != "" {
// 		updates["id_card_photo"] = request.IDCardPhoto
// 	}
// 	if request.ProfilePhoto != "" {
// 		updates["profile_photo"] = request.ProfilePhoto
// 	}

// 	if err := database.DB.Model(&models.WorkerProfile{}).Where("user_id = ?", userID).Updates(updates).Error; err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"success": false,
// 			"message": "Failed to update photos",
// 		})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"success": true,
// 		"message": "Photos updated successfully",
// 	})
// }

package routes