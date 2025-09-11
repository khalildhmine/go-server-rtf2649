package routes

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"repair-service-server/database"
	"repair-service-server/middleware"
	"repair-service-server/models"
)

// RegisterWorkerRoutes registers worker profile routes
func RegisterWorkerRoutes(router *gin.RouterGroup) {
	// Public routes
	router.GET("/workers/available", getAvailableWorkers)
	router.GET("/workers/:id", getWorkerProfile)
	
	// Protected routes
	protected := router.Group("/")
	protected.Use(middleware.AuthMiddleware())
	{
		// Worker profile routes
		protected.GET("/profile", getMyWorkerProfile)
		protected.PUT("/profile", updateWorkerProfile)
		protected.POST("/profile", createWorkerProfile)
	
		// Worker location tracking
		protected.GET("/:id/location", getWorkerLocation)
	}
}

// ===== WORKER HANDLERS =====

func getAvailableWorkers(c *gin.Context) {
	category := c.Query("category")
	city := c.Query("city")
	limit := 20

	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	// Start with a basic query without preload to avoid potential issues
	query := database.DB.Where("is_available = ?", true)

	if category != "" {
		query = query.Where("category = ?", category)
	}

	if city != "" {
		query = query.Where("city ILIKE ?", "%"+city+"%")
	}

	var workers []models.WorkerProfile
	if err := query.Limit(limit).Find(&workers).Error; err != nil {
		log.Printf("Error fetching workers: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to fetch workers",
		})
		return
	}

	// Load user data separately to avoid constraint issues
	for i := range workers {
		var user models.User
		if err := database.DB.First(&user, workers[i].UserID).Error; err == nil {
			workers[i].User = user
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"workers": workers,
	})
}

func getWorkerProfile(c *gin.Context) {
	workerID := c.Param("id")
	id, err := strconv.ParseUint(workerID, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid worker ID",
		})
		return
	}

	var worker models.WorkerProfile
	if err := database.DB.First(&worker, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": "Worker not found",
			})
			return
		}
		log.Printf("Error fetching worker profile: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to fetch worker profile",
		})
		return
	}

	// Load user data separately
	var user models.User
	if err := database.DB.First(&user, worker.UserID).Error; err == nil {
		worker.User = user
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"worker": worker,
	})
}

func getMyWorkerProfile(c *gin.Context) {
	userID := c.GetUint("user_id")

	var worker models.WorkerProfile
	if err := database.DB.Where("user_id = ?", userID).
		Preload("Category"). // Preload category information
		First(&worker).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": "Worker profile not found",
			})
			return
		}
		log.Printf("Error fetching my worker profile: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to fetch worker profile",
		})
		return
	}

	// Debug logging
	log.Printf("🔍 Worker profile loaded - ID: %d, CategoryID: %d, Category: %+v", 
		worker.ID, worker.CategoryID, worker.Category)

	// Load user data separately
	var user models.User
	if err := database.DB.First(&user, worker.UserID).Error; err == nil {
		worker.User = user
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"worker": worker,
	})
}

func createWorkerProfile(c *gin.Context) {
	userID := c.GetUint("user_id")

	// Check if user already has a worker profile
	var existingWorker models.WorkerProfile
	if err := database.DB.Where("user_id = ?", userID).First(&existingWorker).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"message": "Worker profile already exists",
		})
		return
	}

	var request models.WorkerProfileRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request data",
			"error":   err.Error(),
		})
		return
	}

	// Debug logging
	log.Printf("🔧 Creating worker profile - UserID: %d, CategoryID: %d", userID, request.CategoryID)

	worker := models.WorkerProfile{
		UserID:       userID,
		CategoryID:   request.CategoryID,
		PhoneNumber:  request.PhoneNumber,
		Country:      request.Country,
		State:        request.State,
		City:         request.City,
		PostalCode:   request.PostalCode,
		Address:      request.Address,
		Experience:   request.Experience,
		Skills:       request.Skills,
		HourlyRate:   request.HourlyRate,
		ProfilePhoto: request.ProfilePhoto,
		IDCardPhoto:  request.IDCardPhoto,
	}

	if err := database.DB.Create(&worker).Error; err != nil {
		log.Printf("❌ Database error creating worker profile: %v", err)
		log.Printf("❌ Worker data: %+v", worker)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to create worker profile",
			"error":   err.Error(),
		})
		return
	}

	log.Printf("✅ Worker profile created successfully - ID: %d, CategoryID: %d", worker.ID, worker.CategoryID)

	// Load the user data and category
	database.DB.Preload("User").Preload("Category").First(&worker, worker.ID)

	log.Printf("✅ Worker profile loaded with category: %+v", worker.Category)

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Worker profile created successfully",
		"worker":  worker,
	})
}

func updateWorkerProfile(c *gin.Context) {
	userID := c.GetUint("user_id")

	var request models.WorkerProfileRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request data",
			"error":   err.Error(),
		})
		return
	}

	var worker models.WorkerProfile
	if err := database.DB.Where("user_id = ?", userID).First(&worker).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "Worker profile not found",
		})
		return
	}

	// Update fields
	worker.CategoryID = request.CategoryID
	worker.PhoneNumber = request.PhoneNumber
	worker.Country = request.Country
	worker.State = request.State
	worker.PostalCode = request.PostalCode
	worker.City = request.City
	worker.Address = request.Address
	worker.Experience = request.Experience
	worker.Skills = request.Skills
	worker.HourlyRate = request.HourlyRate
	worker.ProfilePhoto = request.ProfilePhoto
	worker.IDCardPhoto = request.IDCardPhoto

	if err := database.DB.Save(&worker).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to update worker profile",
		})
		return
	}

	// Load the user data and category
	database.DB.Preload("User").Preload("Category").First(&worker, worker.ID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Worker profile updated successfully",
		"worker":  worker,
	})
}

func updateAvailability(c *gin.Context) {
	userID := c.GetUint("user_id")

	var request struct {
		IsAvailable bool `json:"is_available" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request data",
		})
		return
	}

	if err := database.DB.Model(&models.WorkerProfile{}).Where("user_id = ?", userID).Update("is_available", request.IsAvailable).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to update availability",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Availability updated successfully",
		"is_available": request.IsAvailable,
	})
}

func uploadWorkerPhotos(c *gin.Context) {
	userID := c.GetUint("user_id")

	var request struct {
		IDCardPhoto  string `json:"id_card_photo"`
		ProfilePhoto string `json:"profile_photo"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request data",
		})
		return
	}

	updates := make(map[string]interface{})
	if request.IDCardPhoto != "" {
		updates["id_card_photo"] = request.IDCardPhoto
	}
	if request.ProfilePhoto != "" {
		updates["profile_photo"] = request.ProfilePhoto
	}

	if err := database.DB.Model(&models.WorkerProfile{}).Where("user_id = ?", userID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to update photos",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Photos updated successfully",
	})
}

// Get worker location for tracking
func getWorkerLocation(c *gin.Context) {
	workerID := c.Param("id")
	userID := c.GetUint("user_id")

	log.Printf("🔍 Getting location for worker %s by user %d", workerID, userID)

	// Parse worker ID
	workerIDUint, err := strconv.ParseUint(workerID, 10, 32)
	if err != nil {
		log.Printf("❌ Invalid worker ID: %s", workerID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid worker ID"})
		return
	}

	// Get worker profile
	var workerProfile models.WorkerProfile
	if err := database.DB.Where("id = ?", workerIDUint).First(&workerProfile).Error; err != nil {
		log.Printf("❌ Worker profile not found: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Worker not found"})
		return
	}

	// Check if worker has location data
	if workerProfile.CurrentLat == nil || workerProfile.CurrentLng == nil {
		log.Printf("❌ Worker %d has no location data", workerIDUint)
		c.JSON(http.StatusNotFound, gin.H{"error": "Worker location not available"})
		return
	}

	// Return worker location
	location := gin.H{
		"lat":       *workerProfile.CurrentLat,
		"lng":       *workerProfile.CurrentLng,
		"accuracy":  workerProfile.LocationAccuracy,
		"timestamp": workerProfile.LastLocationUpdate,
		"status":    "active",
	}

	if workerProfile.LastLocationUpdate != nil {
		// Check if location is recent (within last 5 minutes)
		timeSinceUpdate := time.Since(*workerProfile.LastLocationUpdate)
		if timeSinceUpdate > 5*time.Minute {
			location["status"] = "stale"
		}
	}

	log.Printf("✅ Worker location retrieved: lat=%v, lng=%v, accuracy=%v", 
		*workerProfile.CurrentLat, *workerProfile.CurrentLng, workerProfile.LocationAccuracy)

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"location": location,
	})
}
