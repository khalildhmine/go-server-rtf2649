package routes

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"repair-service-server/database"
	"repair-service-server/models"
)

// GetAllWorkers returns all workers with pagination and filters
func GetAllWorkers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	verified := c.Query("verified")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}

	offset := (page - 1) * limit

	var workers []models.WorkerProfile
	var total int64

	query := database.DB.Model(&models.WorkerProfile{}).Preload("User").Preload("Category")
	
	// Apply verification filter
	if verified == "true" {
		query = query.Where("is_verified = ?", true)
	} else if verified == "false" {
		query = query.Where("is_verified = ?", false)
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		log.Printf("❌ Failed to count workers: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count workers"})
		return
	}

	// Get workers with pagination
	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&workers).Error; err != nil {
		log.Printf("❌ Failed to fetch workers: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch workers"})
		return
	}

	// Format response
	var workerList []gin.H
	for _, worker := range workers {
		workerList = append(workerList, gin.H{
			"id":                    worker.ID,
			"user_id":               worker.UserID,
			"category_id":           worker.CategoryID,
			"category": gin.H{
				"id":   worker.Category.ID,
				"name": worker.Category.Name,
			},
			"phone_number":          worker.PhoneNumber,
			"country":               worker.Country,
			"state":                 worker.State,
			"city":                  worker.City,
			"postal_code":           worker.PostalCode,
			"address":               worker.Address,
			"experience":            worker.Experience,
			"skills":                worker.Skills,
			"hourly_rate":           worker.HourlyRate,
			"profile_photo":         worker.ProfilePhoto,
			"id_card_photo":         worker.IDCardPhoto,
			"id_card_photo_back":    worker.IDCardBackPhoto,
			"is_available":          worker.IsAvailable,
			"current_lat":           worker.CurrentLat,
			"current_lng":           worker.CurrentLng,
			"last_location_update":  worker.LastLocationUpdate,
			"location_accuracy":     worker.LocationAccuracy,
			"active_requests":       worker.ActiveRequests,
			"completed_jobs":        worker.CompletedJobs,
			"rating":                worker.Rating,
			"total_reviews":         worker.TotalReviews,
			"is_verified":           worker.IsVerified,
			"created_at":            worker.CreatedAt,
			"updated_at":            worker.UpdatedAt,
			"user": gin.H{
				"id":                worker.User.ID,
				"full_name":         worker.User.FullName,
				"phone_number":      worker.User.PhoneNumber,
				"role":              worker.User.Role,
				"profile_picture_url": worker.User.ProfilePictureURL,
				"is_active":         worker.User.IsActive,
				"created_at":        worker.User.CreatedAt,
				"updated_at":        worker.User.UpdatedAt,
			},
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    workerList,
		"total":   total,
		"page":    page,
		"limit":   limit,
	})
}

// GetWorkerById returns worker by ID
func GetWorkerById(c *gin.Context) {
	workerID := c.Param("id")
	
	var worker models.WorkerProfile
	if err := database.DB.Preload("User").Preload("Category").First(&worker, workerID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Worker not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"id":                    worker.ID,
			"user_id":               worker.UserID,
			"category_id":           worker.CategoryID,
			"category": gin.H{
				"id":   worker.Category.ID,
				"name": worker.Category.Name,
			},
			"phone_number":          worker.PhoneNumber,
			"country":               worker.Country,
			"state":                 worker.State,
			"city":                  worker.City,
			"postal_code":           worker.PostalCode,
			"address":               worker.Address,
			"experience":            worker.Experience,
			"skills":                worker.Skills,
			"hourly_rate":           worker.HourlyRate,
			"profile_photo":         worker.ProfilePhoto,
			"id_card_photo":         worker.IDCardPhoto,
			"id_card_photo_back":    worker.IDCardBackPhoto,
			"is_available":          worker.IsAvailable,
			"current_lat":           worker.CurrentLat,
			"current_lng":           worker.CurrentLng,
			"last_location_update":  worker.LastLocationUpdate,
			"location_accuracy":     worker.LocationAccuracy,
			"active_requests":       worker.ActiveRequests,
			"completed_jobs":        worker.CompletedJobs,
			"rating":                worker.Rating,
			"total_reviews":         worker.TotalReviews,
			"is_verified":           worker.IsVerified,
			"created_at":            worker.CreatedAt,
			"updated_at":            worker.UpdatedAt,
			"user": gin.H{
				"id":                worker.User.ID,
				"full_name":         worker.User.FullName,
				"phone_number":      worker.User.PhoneNumber,
				"role":              worker.User.Role,
				"profile_picture_url": worker.User.ProfilePictureURL,
				"is_active":         worker.User.IsActive,
				"created_at":        worker.User.CreatedAt,
				"updated_at":        worker.User.UpdatedAt,
			},
		},
	})
}

// GetWorkerStatsForAdmin gets worker statistics for admin
func GetWorkerStatsForAdmin(c *gin.Context) {
	workerID := c.Param("id")
	
	var worker models.WorkerProfile
	if err := database.DB.First(&worker, workerID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Worker not found"})
		return
	}

	// Get worker stats
	var stats models.WorkerStats
	if err := database.DB.Where("worker_id = ?", worker.ID).First(&stats).Error; err != nil {
		// If no stats found, create default stats
		stats = models.WorkerStats{
			WorkerID: worker.ID,
		}
		database.DB.Create(&stats)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"id":                        stats.ID,
			"worker_id":                 stats.WorkerID,
			"total_jobs_received":       stats.TotalJobsReceived,
			"total_jobs_responded":      stats.TotalJobsResponded,
			"total_jobs_completed":      stats.TotalJobsCompleted,
			"total_jobs_declined":       stats.TotalJobsDeclined,
			"total_earnings":            stats.TotalEarnings,
			"total_work_hours":          stats.TotalWorkHours,
			"monthly_jobs_received":     stats.MonthlyJobsReceived,
			"monthly_jobs_responded":    stats.MonthlyJobsResponded,
			"monthly_jobs_completed":    stats.MonthlyJobsCompleted,
			"monthly_jobs_declined":     stats.MonthlyJobsDeclined,
			"monthly_earnings":          stats.MonthlyEarnings,
			"monthly_work_hours":        stats.MonthlyWorkHours,
			"daily_jobs_received":       stats.DailyJobsReceived,
			"daily_jobs_responded":      stats.DailyJobsResponded,
			"daily_jobs_completed":      stats.DailyJobsCompleted,
			"daily_jobs_declined":       stats.DailyJobsDeclined,
			"daily_earnings":            stats.DailyEarnings,
			"daily_work_hours":          stats.DailyWorkHours,
			"response_rate":             stats.ResponseRate,
			"completion_rate":           stats.CompletionRate,
			"average_response_time":     stats.AverageResponseTime,
			"average_job_duration":      stats.AverageJobDuration,
			// "success_rate":              stats.SuccessRate,
			"created_at":                stats.CreatedAt,
			"updated_at":                stats.UpdatedAt,
		},
	})
}

// VerifyWorker verifies/unverifies worker
func VerifyWorker(c *gin.Context) {
	workerID := c.Param("id")
	adminID := c.GetUint("user_id")
	
	var req struct {
		IsVerified bool `json:"is_verified" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	var worker models.WorkerProfile
	if err := database.DB.Preload("User").Preload("Category").First(&worker, workerID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Worker not found"})
		return
	}

	worker.IsVerified = req.IsVerified
	if err := database.DB.Save(&worker).Error; err != nil {
		log.Printf("❌ Failed to update worker verification: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update worker verification"})
		return
	}

	log.Printf("✅ Worker %d verification updated to %v by admin %d", worker.ID, req.IsVerified, adminID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Worker verification updated successfully",
		"data": gin.H{
			"id":                    worker.ID,
			"user_id":               worker.UserID,
			"category_id":           worker.CategoryID,
			"category": gin.H{
				"id":   worker.Category.ID,
				"name": worker.Category.Name,
			},
			"phone_number":          worker.PhoneNumber,
			"country":               worker.Country,
			"state":                 worker.State,
			"city":                  worker.City,
			"postal_code":           worker.PostalCode,
			"address":               worker.Address,
			"experience":            worker.Experience,
			"skills":                worker.Skills,
			"hourly_rate":           worker.HourlyRate,
			"profile_photo":         worker.ProfilePhoto,
			"id_card_photo":         worker.IDCardPhoto,
			"id_card_photo_back":    worker.IDCardBackPhoto,
			"is_available":          worker.IsAvailable,
			"current_lat":           worker.CurrentLat,
			"current_lng":           worker.CurrentLng,
			"last_location_update":  worker.LastLocationUpdate,
			"location_accuracy":     worker.LocationAccuracy,
			"active_requests":       worker.ActiveRequests,
			"completed_jobs":        worker.CompletedJobs,
			"rating":                worker.Rating,
			"total_reviews":         worker.TotalReviews,
			"is_verified":           worker.IsVerified,
			"created_at":            worker.CreatedAt,
			"updated_at":            worker.UpdatedAt,
			"user": gin.H{
				"id":                worker.User.ID,
				"full_name":         worker.User.FullName,
				"phone_number":      worker.User.PhoneNumber,
				"role":              worker.User.Role,
				"profile_picture_url": worker.User.ProfilePictureURL,
				"is_active":         worker.User.IsActive,
				"created_at":        worker.User.CreatedAt,
				"updated_at":        worker.User.UpdatedAt,
			},
		},
	})
}

// UpdateWorkerAvailability updates worker availability
func UpdateWorkerAvailability(c *gin.Context) {
	workerID := c.Param("id")
	adminID := c.GetUint("user_id")
	
	var req struct {
		IsAvailable bool `json:"is_available" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	var worker models.WorkerProfile
	if err := database.DB.Preload("User").Preload("Category").First(&worker, workerID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Worker not found"})
		return
	}

	worker.IsAvailable = req.IsAvailable
	if err := database.DB.Save(&worker).Error; err != nil {
		log.Printf("❌ Failed to update worker availability: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update worker availability"})
		return
	}

	log.Printf("✅ Worker %d availability updated to %v by admin %d", worker.ID, req.IsAvailable, adminID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Worker availability updated successfully",
		"data": gin.H{
			"id":                    worker.ID,
			"user_id":               worker.UserID,
			"category_id":           worker.CategoryID,
			"category": gin.H{
				"id":   worker.Category.ID,
				"name": worker.Category.Name,
			},
			"phone_number":          worker.PhoneNumber,
			"country":               worker.Country,
			"state":                 worker.State,
			"city":                  worker.City,
			"postal_code":           worker.PostalCode,
			"address":               worker.Address,
			"experience":            worker.Experience,
			"skills":                worker.Skills,
			"hourly_rate":           worker.HourlyRate,
			"profile_photo":         worker.ProfilePhoto,
			"id_card_photo":         worker.IDCardPhoto,
			"id_card_photo_back":    worker.IDCardBackPhoto,
			"is_available":          worker.IsAvailable,
			"current_lat":           worker.CurrentLat,
			"current_lng":           worker.CurrentLng,
			"last_location_update":  worker.LastLocationUpdate,
			"location_accuracy":     worker.LocationAccuracy,
			"active_requests":       worker.ActiveRequests,
			"completed_jobs":        worker.CompletedJobs,
			"rating":                worker.Rating,
			"total_reviews":         worker.TotalReviews,
			"is_verified":           worker.IsVerified,
			"created_at":            worker.CreatedAt,
			"updated_at":            worker.UpdatedAt,
			"user": gin.H{
				"id":                worker.User.ID,
				"full_name":         worker.User.FullName,
				"phone_number":      worker.User.PhoneNumber,
				"role":              worker.User.Role,
				"profile_picture_url": worker.User.ProfilePictureURL,
				"is_active":         worker.User.IsActive,
				"created_at":        worker.User.CreatedAt,
				"updated_at":        worker.User.UpdatedAt,
			},
		},
	})
}
