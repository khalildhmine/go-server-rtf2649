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

// RegisterServiceHistoryRoutes registers all service history-related routes
func RegisterServiceHistoryRoutes(router *gin.RouterGroup) {
	historyRoutes := router.Group("/service-history")
	{
		// Create service history entry when service is completed
		historyRoutes.POST("/", createServiceHistory)
		
		// Get service history for a specific worker
		historyRoutes.GET("/worker/:workerId", getWorkerServiceHistory)
		
		// Get service history for a specific customer
		historyRoutes.GET("/customer", getCustomerServiceHistory)
		
		// Get service history summary for a worker
		historyRoutes.GET("/worker/:workerId/summary", getWorkerServiceSummary)
		
		// Get a specific service history entry
		historyRoutes.GET("/:historyId", getServiceHistory)
		
		// Update service history (only by the worker who completed it)
		historyRoutes.PUT("/:historyId", updateServiceHistory)
		
		// Get all service history with filters
		historyRoutes.GET("/", getServiceHistoryList)
	}
}

// createServiceHistory creates a new service history entry when a service is completed
func createServiceHistory(c *gin.Context) {
	var historyData models.ServiceHistoryCreate
	if err := c.ShouldBindJSON(&historyData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid history data", "details": err.Error()})
		return
	}

	// Get current user ID from context
	workerID := c.GetUint("user_id")

	// Verify the worker is authorized to create history for this service
	if historyData.WorkerID != workerID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only create history for your own services"})
		return
	}

	// Verify the service request exists and is completed
	var serviceRequest models.CustomerServiceRequest
	if err := database.DB.
		Preload("Customer").
		Preload("Category").
		Preload("ServiceOption").
		First(&serviceRequest, historyData.ServiceRequestID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Service request not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch service request"})
		}
		return
	}

	// Verify the service request is assigned to this worker
	if serviceRequest.AssignedWorkerID == nil || *serviceRequest.AssignedWorkerID != workerID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Service request is not assigned to you"})
		return
	}

	// Verify the service request is completed
	if serviceRequest.Status != models.RequestStatusCompleted {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Can only create history for completed services"})
		return
	}

	// Check if history already exists for this service request
	var existingHistory models.ServiceHistory
	if err := database.DB.Where("service_request_id = ?", historyData.ServiceRequestID).First(&existingHistory).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Service history already exists for this service request"})
		return
	}

	// Create the service history
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
		CompletedAt:       *serviceRequest.CompletedAt, // Dereference the pointer
		AgreedPrice:       historyData.AgreedPrice,
		FinalPrice:        historyData.FinalPrice,
		PaymentStatus:     historyData.PaymentStatus,
		WorkerNotes:       historyData.WorkerNotes,
		CustomerNotes:     historyData.CustomerNotes,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	if err := database.DB.Create(&history).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create service history"})
		return
	}

	// Update worker profile statistics
	if err := updateWorkerServiceStats(workerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "History created but failed to update worker stats"})
		return
	}

	// Load the created history with relationships
	var createdHistory models.ServiceHistory
	if err := database.DB.
		Preload("Customer").
		Preload("Worker").
		Preload("Category").
		Preload("ServiceOption").
		First(&createdHistory, history.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "History created but failed to load details"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Service history created successfully",
		"history": createdHistory,
	})
}

// getWorkerServiceHistory retrieves service history for a specific worker
func getWorkerServiceHistory(c *gin.Context) {
	workerIDStr := c.Param("workerId")
	workerID, err := strconv.ParseUint(workerIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid worker ID"})
		return
	}

	// Get query parameters for pagination and filtering
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	year, _ := strconv.Atoi(c.Query("year"))
	month, _ := strconv.Atoi(c.Query("month"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 10
	}

	offset := (page - 1) * limit

	// Build query
	query := database.DB.Where("worker_id = ?", workerID)
	
	// Filter by year and month if provided
	if year > 0 {
		query = query.Where("YEAR(completed_at) = ?", year)
		if month > 0 && month <= 12 {
			query = query.Where("MONTH(completed_at) = ?", month)
		}
	}

	// Get total count
	var total int64
	query.Model(&models.ServiceHistory{}).Count(&total)

	// Get history with pagination
	var history []models.ServiceHistory
	if err := query.
		Preload("Customer").
		Preload("Category").
		Preload("ServiceOption").
		Order("completed_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&history).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch service history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"history": history,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
			"pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// getCustomerServiceHistory retrieves service history for the current customer
func getCustomerServiceHistory(c *gin.Context) {
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
	database.DB.Model(&models.ServiceHistory{}).Where("customer_id = ?", customerID).Count(&total)

	// Get history with pagination
	var history []models.ServiceHistory
	if err := database.DB.
		Where("customer_id = ?", customerID).
		Preload("Worker").
		Preload("Category").
		Preload("ServiceOption").
		Order("completed_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&history).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch service history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"history": history,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
			"pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// getWorkerServiceSummary retrieves a summary of services for a specific worker
func getWorkerServiceSummary(c *gin.Context) {
	workerIDStr := c.Param("workerId")
	workerID, err := strconv.ParseUint(workerIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid worker ID"})
		return
	}

	// Get current month and year
	now := time.Now()
	currentMonth := now.Month()
	currentYear := now.Year()

	// Get summary statistics
	var summary models.WorkerServiceSummary
	if err := database.DB.Raw(`
		SELECT 
			worker_id,
			COUNT(*) as total_services,
			COALESCE(SUM(final_price), 0) as total_earnings,
			AVG(CAST(actual_duration AS DECIMAL(10,2))) as average_duration,
			AVG(CAST(customer_satisfaction AS DECIMAL(3,2))) as customer_satisfaction
		FROM service_histories 
		WHERE worker_id = ? AND deleted_at IS NULL
	`, workerID).Scan(&summary).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch service summary"})
		return
	}

	// Get monthly and yearly counts
	var monthlyCount int64
	var yearlyCount int64

	database.DB.Model(&models.ServiceHistory{}).
		Where("worker_id = ? AND MONTH(completed_at) = ? AND YEAR(completed_at) = ?", 
			workerID, currentMonth, currentYear).
		Count(&monthlyCount)

	database.DB.Model(&models.ServiceHistory{}).
		Where("worker_id = ? AND YEAR(completed_at) = ?", 
			workerID, currentYear).
		Count(&yearlyCount)

	summary.WorkerID = uint(workerID)
	summary.CompletedThisMonth = int(monthlyCount)
	summary.CompletedThisYear = int(yearlyCount)

	// Get average rating from worker profile
	var workerProfile models.WorkerProfile
	if err := database.DB.Select("rating, total_reviews").First(&workerProfile, workerID).Error; err == nil {
		summary.AverageRating = workerProfile.Rating
		summary.TotalRatings = workerProfile.TotalReviews
	}

	c.JSON(http.StatusOK, summary)
}

// getServiceHistory retrieves a specific service history entry
func getServiceHistory(c *gin.Context) {
	historyIDStr := c.Param("historyId")
	historyID, err := strconv.ParseUint(historyIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid history ID"})
		return
	}

	var history models.ServiceHistory
	if err := database.DB.
		Preload("Customer").
		Preload("Worker").
		Preload("Category").
		Preload("ServiceOption").
		First(&history, historyID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Service history not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch service history"})
		}
		return
	}

	c.JSON(http.StatusOK, history)
}

// updateServiceHistory updates an existing service history entry
func updateServiceHistory(c *gin.Context) {
	historyIDStr := c.Param("historyId")
	historyID, err := strconv.ParseUint(historyIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid history ID"})
		return
	}

	// Get current user ID
	workerID := c.GetUint("user_id")

	// Check if history exists and belongs to current worker
	var existingHistory models.ServiceHistory
	if err := database.DB.First(&existingHistory, historyID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Service history not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch service history"})
		}
		return
	}

	if existingHistory.WorkerID != workerID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only update your own service history"})
		return
	}

	// Parse update data
	var updateData models.ServiceHistoryCreate
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid update data"})
		return
	}

	// Update the history
	updates := map[string]interface{}{
		"actual_duration": updateData.ActualDuration,
		"agreed_price":    updateData.AgreedPrice,
		"final_price":     updateData.FinalPrice,
		"payment_status":  updateData.PaymentStatus,
		"worker_notes":    updateData.WorkerNotes,
		"customer_notes":  updateData.CustomerNotes,
		"updated_at":      time.Now(),
	}

	if err := database.DB.Model(&existingHistory).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update service history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Service history updated successfully"})
}

// getServiceHistoryList retrieves all service history with filters
func getServiceHistoryList(c *gin.Context) {
	// Get query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	workerID, _ := strconv.ParseUint(c.Query("worker_id"), 10, 32)
	customerID, _ := strconv.ParseUint(c.Query("customer_id"), 10, 32)
	categoryID, _ := strconv.ParseUint(c.Query("category_id"), 10, 32)
	year, _ := strconv.Atoi(c.Query("year"))
	month, _ := strconv.Atoi(c.Query("month"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	offset := (page - 1) * limit

	// Build query
	query := database.DB.Model(&models.ServiceHistory{})
	
	if workerID > 0 {
		query = query.Where("worker_id = ?", workerID)
	}
	if customerID > 0 {
		query = query.Where("customer_id = ?", customerID)
	}
	if categoryID > 0 {
		query = query.Where("category_id = ?", categoryID)
	}
	if year > 0 {
		query = query.Where("YEAR(completed_at) = ?", year)
		if month > 0 && month <= 12 {
			query = query.Where("MONTH(completed_at) = ?", month)
		}
	}

	// Get total count
	var total int64
	query.Count(&total)

	// Get history with pagination
	var history []models.ServiceHistory
	if err := query.
		Preload("Customer").
		Preload("Worker").
		Preload("Category").
		Preload("ServiceOption").
		Order("completed_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&history).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch service history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"history": history,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
			"pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// updateWorkerServiceStats updates the service statistics for a worker
func updateWorkerServiceStats(workerID uint) error {
	// Count total completed services
	var totalServices int64
	if err := database.DB.Model(&models.ServiceHistory{}).Where("worker_id = ?", workerID).Count(&totalServices).Error; err != nil {
		return err
	}

	// Update worker profile
	updates := map[string]interface{}{
		"completed_jobs": totalServices,
		"updated_at":     time.Now(),
	}

	return database.DB.Model(&models.WorkerProfile{}).Where("id = ?", workerID).Updates(updates).Error
}
