package routes

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"repair-service-server/database"
	"repair-service-server/models"
	"repair-service-server/utils"
	"time"

	"github.com/gin-gonic/gin"
)

// RegisterLocationRoutes registers all location-related routes
func RegisterLocationRoutes(router *gin.RouterGroup) {
	// Update worker location and availability
	router.POST("/update", updateWorkerLocation)
	
	// Toggle worker availability
	router.POST("/availability", toggleWorkerAvailability)
	
	// Get nearby workers for a service category
	router.GET("/nearby-workers", getNearbyWorkers)
	
	// Get worker's current location (for debugging)
	router.GET("/current", getCurrentLocation)
}

// updateWorkerLocation handles worker location updates
func updateWorkerLocation(c *gin.Context) {
	userID := c.GetUint("user_id")
	
	var req models.LocationUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Validate location coordinates
	if !utils.IsLocationValid(req.Latitude, req.Longitude) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid location coordinates"})
		return
	}
	
	// Get worker profile
	var workerProfile models.WorkerProfile
	if err := database.DB.Where("user_id = ?", userID).First(&workerProfile).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Worker profile not found"})
		return
	}
	
	// Update location and availability
	now := time.Now()
	workerProfile.CurrentLat = &req.Latitude
	workerProfile.CurrentLng = &req.Longitude
	workerProfile.LastLocationUpdate = &now
	workerProfile.LocationAccuracy = &req.Accuracy
	workerProfile.IsAvailable = req.IsAvailable
	
	if err := database.DB.Save(&workerProfile).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update location"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Location updated successfully",
		"worker_profile": workerProfile,
	})
}

// toggleWorkerAvailability handles worker availability toggling
func toggleWorkerAvailability(c *gin.Context) {
	userID := c.GetUint("user_id")
	
	var req struct {
		IsAvailable bool `json:"is_available"`
	}
	
	// Log the raw request body for debugging
	body, _ := c.GetRawData()
	log.Printf("🔍 Availability toggle request body: %s", string(body))
	
	// Re-create the request body since GetRawData() consumes it
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
	
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("❌ JSON binding error: %v", err)
		log.Printf("🔍 Request body: %s", string(body))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
			"message": err.Error(),
			"expected": "JSON with 'is_available' boolean field",
		})
		return
	}
	
	log.Printf("✅ Parsed request: is_available = %v", req.IsAvailable)
	

	
	// Get worker profile
	var workerProfile models.WorkerProfile
	if err := database.DB.Where("user_id = ?", userID).First(&workerProfile).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Worker profile not found"})
		return
	}
	
	// Update availability
	workerProfile.IsAvailable = req.IsAvailable
	
	if err := database.DB.Save(&workerProfile).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update availability"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Availability updated successfully",
		"is_available": workerProfile.IsAvailable,
	})
}

// getNearbyWorkers returns workers within a specified radius for a service category
func getNearbyWorkers(c *gin.Context) {
	// Parse query parameters
	latStr := c.Query("lat")
	lngStr := c.Query("lng")
	categoryStr := c.Query("category")
	radiusStr := c.Query("radius")
	
	if latStr == "" || lngStr == "" || categoryStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required parameters: lat, lng, category"})
		return
	}
	
	// Parse coordinates
	lat, lng, err := parseCoordinates(latStr, lngStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid coordinates"})
		return
	}
	
	// Parse category
	category := models.WorkerCategory(categoryStr)
	
	// Parse radius (default to 10km if not specified)
	radius := utils.GetDefaultBroadcastRadius()
	if radiusStr != "" {
		if parsedRadius, err := parseFloat(radiusStr); err == nil {
			radius = parsedRadius
		}
	}
	
	// Validate radius
	if !utils.ValidateBroadcastRadius(radius) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid broadcast radius"})
		return
	}
	
	// Find nearby workers
	location := utils.Location{Latitude: lat, Longitude: lng}
	workers, err := utils.FindNearbyWorkers(database.DB, location, radius, category)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find nearby workers"})
		return
	}
	
	// Calculate distances and ETAs for each worker
	var workerResponses []gin.H
	for _, worker := range workers {
		if worker.CurrentLat != nil && worker.CurrentLng != nil {
			distance := utils.HaversineDistance(lat, lng, *worker.CurrentLat, *worker.CurrentLng)
			eta := utils.CalculateETA(
				utils.Location{Latitude: *worker.CurrentLat, Longitude: *worker.CurrentLng},
				location,
				30.0, // Assume average speed of 30 km/h
			)
			
			workerResponses = append(workerResponses, gin.H{
				"id": worker.ID,
				"name": worker.User.FullName,
				"category": worker.Category,
				"rating": worker.Rating,
				"completed_jobs": worker.CompletedJobs,
				"distance": distance,
				"eta_minutes": int(eta.Minutes()),
				"is_available": worker.IsAvailable,
				"last_location_update": worker.LastLocationUpdate,
			})
		}
	}
	
	c.JSON(http.StatusOK, gin.H{
		"workers": workerResponses,
		"total_count": len(workerResponses),
		"search_radius": radius,
		"search_location": gin.H{
			"lat": lat,
			"lng": lng,
		},
	})
}

// getCurrentLocation returns the current user's location (for debugging)
func getCurrentLocation(c *gin.Context) {
	userID := c.GetUint("user_id")
	
	var workerProfile models.WorkerProfile
	if err := database.DB.Where("user_id = ?", userID).First(&workerProfile).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Worker profile not found"})
		return
	}
	
	if workerProfile.CurrentLat == nil || workerProfile.CurrentLng == nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "No location data available",
			"has_location": false,
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"has_location": true,
		"location": gin.H{
			"lat": workerProfile.CurrentLat,
			"lng": workerProfile.CurrentLng,
			"accuracy": workerProfile.LocationAccuracy,
			"last_update": workerProfile.LastLocationUpdate,
		},
		"is_available": workerProfile.IsAvailable,
	})
}

// Helper functions for parsing query parameters
func parseCoordinates(latStr, lngStr string) (float64, float64, error) {
	lat, err := parseFloat(latStr)
	if err != nil {
		return 0, 0, err
	}
	
	lng, err := parseFloat(lngStr)
	if err != nil {
		return 0, 0, err
	}
	
	return lat, lng, nil
}

func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}
