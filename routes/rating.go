package routes

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"repair-service-server/database"
	"repair-service-server/models"
)

// RegisterRatingRoutes registers all rating-related routes
func RegisterRatingRoutes(router *gin.RouterGroup) {
	ratingRoutes := router.Group("/ratings")
	{
		// Create a new rating for a worker
		ratingRoutes.POST("/", createWorkerRating)
		
		// Get ratings for a specific worker
		ratingRoutes.GET("/worker/:workerId", getWorkerRatings)
		
		// Get rating summary for a worker
		ratingRoutes.GET("/worker/:workerId/summary", getWorkerRatingSummary)
		
		// Get a specific rating
		ratingRoutes.GET("/:ratingId", getRating)
		
		// Update a rating (only by the customer who created it)
		ratingRoutes.PUT("/:ratingId", updateRating)
		
		// Delete a rating (only by the customer who created it)
		ratingRoutes.DELETE("/:ratingId", deleteRating)
		
		// Get all ratings for a customer
		ratingRoutes.GET("/customer", getCustomerRatings)
	}
}

// createWorkerRating creates a new rating for a worker after service completion
func createWorkerRating(c *gin.Context) {
	var ratingData models.WorkerRatingCreate
	if err := c.ShouldBindJSON(&ratingData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid rating data", "details": err.Error()})
		return
	}

	// Get current user ID from context
	customerID := c.GetUint("user_id")

	// Verify the service request exists and belongs to this customer
	var serviceRequest models.CustomerServiceRequest
	if err := database.DB.
		Preload("AssignedWorker").
		First(&serviceRequest, ratingData.ServiceRequestID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Service request not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch service request"})
		}
		return
	}

	// Verify the service request belongs to the current customer
	if serviceRequest.CustomerID != customerID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only rate services you requested"})
		return
	}

	// Verify the service request is completed
	if serviceRequest.Status != models.RequestStatusCompleted {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Can only rate completed services"})
		return
	}

	// Verify the service request has an assigned worker
	if serviceRequest.AssignedWorkerID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Service request has no assigned worker"})
		return
	}

	// Check if rating already exists for this service request
	var existingRating models.WorkerRating
	if err := database.DB.Where("service_request_id = ?", ratingData.ServiceRequestID).First(&existingRating).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Rating already exists for this service request"})
		return
	}

	// Create the rating
	rating := models.WorkerRating{
		CustomerID:      customerID,
		WorkerID:        *serviceRequest.AssignedWorkerID,
		ServiceRequestID: ratingData.ServiceRequestID,
		Stars:           ratingData.Stars,
		Comment:         ratingData.Comment,
		ServiceQuality:  ratingData.ServiceQuality,
		Professionalism: ratingData.Professionalism,
		Punctuality:     ratingData.Punctuality,
		Communication:   ratingData.Communication,
		IsAnonymous:     ratingData.IsAnonymous,
		IsVerified:      true, // Service was completed
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := database.DB.Create(&rating).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create rating"})
		return
	}

	// Update worker profile statistics
	if err := updateWorkerRatingStats(*serviceRequest.AssignedWorkerID); err != nil {
		// Log error but don't fail the rating creation
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Rating created but failed to update worker stats"})
		return
	}

	// Load the created rating with relationships
	var createdRating models.WorkerRating
	if err := database.DB.
		Preload("Customer").
		Preload("Worker").
		Preload("ServiceRequest").
		First(&createdRating, rating.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Rating created but failed to load details"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Rating created successfully",
		"rating":  createdRating,
	})
}

// getWorkerRatings retrieves all ratings for a specific worker
func getWorkerRatings(c *gin.Context) {
	workerIDStr := c.Param("workerId")
	workerID, err := strconv.ParseUint(workerIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid worker ID"})
		return
	}

	// Get query parameters for pagination and filtering
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	stars, _ := strconv.Atoi(c.Query("stars")) // Filter by star rating

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 10
	}

	offset := (page - 1) * limit

	// Build query
	query := database.DB.Where("worker_id = ?", workerID)
	if stars > 0 && stars <= 5 {
		query = query.Where("stars = ?", stars)
	}

	// Get total count
	var total int64
	query.Model(&models.WorkerRating{}).Count(&total)

	// Get ratings with pagination
	var ratings []models.WorkerRating
	if err := query.
		Preload("Customer").
		Preload("ServiceRequest").
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&ratings).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch ratings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ratings": ratings,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
			"pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// getWorkerRatingSummary retrieves a summary of ratings for a specific worker
func getWorkerRatingSummary(c *gin.Context) {
	workerIDStr := c.Param("workerId")
	workerID, err := strconv.ParseUint(workerIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid worker ID"})
		return
	}

	// Get rating summary
	var summary models.WorkerRatingSummary
	if err := database.DB.Raw(`
		SELECT 
			worker_id,
			AVG(CAST(stars AS DECIMAL(3,2))) as average_stars,
			COUNT(*) as total_ratings,
			SUM(CASE WHEN stars = 5 THEN 1 ELSE 0 END) as five_star_count,
			SUM(CASE WHEN stars = 4 THEN 1 ELSE 0 END) as four_star_count,
			SUM(CASE WHEN stars = 3 THEN 1 ELSE 0 END) as three_star_count,
			SUM(CASE WHEN stars = 2 THEN 1 ELSE 0 END) as two_star_count,
			SUM(CASE WHEN stars = 1 THEN 1 ELSE 0 END) as one_star_count,
			AVG(CAST(service_quality AS DECIMAL(3,2))) as average_service_quality,
			AVG(CAST(professionalism AS DECIMAL(3,2))) as average_professionalism,
			AVG(CAST(punctuality AS DECIMAL(3,2))) as average_punctuality,
			AVG(CAST(communication AS DECIMAL(3,2))) as average_communication
		FROM worker_ratings 
		WHERE worker_id = ? AND deleted_at IS NULL
		GROUP BY worker_id
	`, workerID).Scan(&summary).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch rating summary"})
		return
	}

	// If no ratings found, return default values
	if summary.TotalRatings == 0 {
		summary.WorkerID = uint(workerID)
		summary.AverageStars = 0
		summary.TotalRatings = 0
	}

	c.JSON(http.StatusOK, summary)
}

// getRating retrieves a specific rating by ID
func getRating(c *gin.Context) {
	ratingIDStr := c.Param("ratingId")
	ratingID, err := strconv.ParseUint(ratingIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid rating ID"})
		return
	}

	var rating models.WorkerRating
	if err := database.DB.
		Preload("Customer").
		Preload("Worker").
		Preload("ServiceRequest").
		First(&rating, ratingID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Rating not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch rating"})
		}
		return
	}

	c.JSON(http.StatusOK, rating)
}

// updateRating updates an existing rating
func updateRating(c *gin.Context) {
	ratingIDStr := c.Param("ratingId")
	ratingID, err := strconv.ParseUint(ratingIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid rating ID"})
		return
	}

	// Get current user ID
	customerID := c.GetUint("user_id")

	// Check if rating exists and belongs to current user
	var existingRating models.WorkerRating
	if err := database.DB.First(&existingRating, ratingID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Rating not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch rating"})
		}
		return
	}

	if existingRating.CustomerID != customerID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only update your own ratings"})
		return
	}

	// Parse update data
	var updateData models.WorkerRatingCreate
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid update data"})
		return
	}

	// Update the rating
	updates := map[string]interface{}{
		"stars":            updateData.Stars,
		"comment":          updateData.Comment,
		"service_quality":  updateData.ServiceQuality,
		"professionalism":  updateData.Professionalism,
		"punctuality":      updateData.Punctuality,
		"communication":    updateData.Communication,
		"is_anonymous":     updateData.IsAnonymous,
		"updated_at":       time.Now(),
	}

	if err := database.DB.Model(&existingRating).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update rating"})
		return
	}

	// Update worker rating stats
	if err := updateWorkerRatingStats(existingRating.WorkerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Rating updated but failed to update worker stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Rating updated successfully"})
}

// deleteRating deletes a rating
func deleteRating(c *gin.Context) {
	ratingIDStr := c.Param("ratingId")
	ratingID, err := strconv.ParseUint(ratingIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid rating ID"})
		return
	}

	// Get current user ID
	customerID := c.GetUint("user_id")

	// Check if rating exists and belongs to current user
	var existingRating models.WorkerRating
	if err := database.DB.First(&existingRating, ratingID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Rating not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch rating"})
		}
		return
	}

	if existingRating.CustomerID != customerID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only delete your own ratings"})
		return
	}

	// Delete the rating
	if err := database.DB.Delete(&existingRating).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete rating"})
		return
	}

	// Update worker rating stats
	if err := updateWorkerRatingStats(existingRating.WorkerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Rating deleted but failed to update worker stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Rating deleted successfully"})
}

// getCustomerRatings retrieves all ratings given by the current customer
func getCustomerRatings(c *gin.Context) {
	// Get current user ID from context
	customerID := c.GetUint("user_id")

	// Get pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 10
	}

	offset := (page - 1) * limit

	// Get total count
	var total int64
	database.DB.Model(&models.WorkerRating{}).Where("customer_id = ?", customerID).Count(&total)

	// Get ratings with pagination
	var ratings []models.WorkerRating
	if err := database.DB.
		Where("customer_id = ?", customerID).
		Preload("Worker").
		Preload("ServiceRequest").
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&ratings).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch ratings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ratings": ratings,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
			"pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// updateWorkerRatingStats updates the rating statistics for a worker
func updateWorkerRatingStats(workerID uint) error {
	var summary models.WorkerRatingSummary
	if err := database.DB.Raw(`
		SELECT 
			AVG(CAST(stars AS DECIMAL(3,2))) as average_stars,
			COUNT(*) as total_ratings
		FROM worker_ratings 
		WHERE worker_id = ? AND deleted_at IS NULL
	`, workerID).Scan(&summary).Error; err != nil {
		return err
	}

	// Update worker profile
	updates := map[string]interface{}{
		"rating":        summary.AverageStars,
		"total_reviews": summary.TotalRatings,
		"updated_at":    time.Now(),
	}

	return database.DB.Model(&models.WorkerProfile{}).Where("id = ?", workerID).Updates(updates).Error
}
