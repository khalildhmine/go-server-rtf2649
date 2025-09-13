package routes

import (
	"net/http"
	"strconv"
	"time"

	"repair-service-server/database"
	"repair-service-server/models"

	"github.com/gin-gonic/gin"
)

// GetAllFeedback returns all feedback with pagination and filtering
func GetAllFeedback(c *gin.Context) {
	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	rating, _ := strconv.Atoi(c.Query("rating"))
	search := c.Query("search")

	// Calculate offset
	offset := (page - 1) * limit

	// Build query
	query := database.DB.Model(&models.Feedback{})

	// Apply filters
	if rating > 0 {
		query = query.Where("rating = ?", rating)
	}
	if search != "" {
		query = query.Where("comment ILIKE ?", "%"+search+"%")
	}

	// Get total count
	var total int64
	query.Count(&total)

	// Get feedback with pagination
	var feedback []models.Feedback
	if err := query.
		Preload("User").
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&feedback).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to fetch feedback",
		})
		return
	}

	// Calculate statistics
	var avgRating float64
	var ratingCounts [6]int // 0-5 stars
	database.DB.Model(&models.Feedback{}).
		Select("COALESCE(AVG(rating), 0)").
		Scan(&avgRating)

	for i := 1; i <= 5; i++ {
		var count int64
		database.DB.Model(&models.Feedback{}).Where("rating = ?", i).Count(&count)
		ratingCounts[i] = int(count)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"feedback": feedback,
			"pagination": gin.H{
				"page":       page,
				"limit":      limit,
				"total":      total,
				"total_pages": (total + int64(limit) - 1) / int64(limit),
			},
			"statistics": gin.H{
				"average_rating": avgRating,
				"rating_counts":  ratingCounts[1:], // 1-5 stars
				"total_feedback": total,
			},
		},
	})
}

// GetFeedbackById returns a specific feedback entry
func GetFeedbackById(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid feedback ID",
		})
		return
	}

	var feedback models.Feedback
	if err := database.DB.
		Preload("User").
		First(&feedback, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "Feedback not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    feedback,
	})
}

// DeleteFeedback deletes a feedback entry
func DeleteFeedback(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid feedback ID",
		})
		return
	}

	if err := database.DB.Delete(&models.Feedback{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to delete feedback",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Feedback deleted successfully",
	})
}

// GetFeedbackStats returns feedback statistics for admin dashboard
func GetFeedbackStats(c *gin.Context) {
	var stats struct {
		TotalFeedback   int64   `json:"total_feedback"`
		AverageRating   float64 `json:"average_rating"`
		RecentFeedback  int64   `json:"recent_feedback"` // Last 7 days
		RatingDistribution [5]int `json:"rating_distribution"`
	}

	// Total feedback count
	database.DB.Model(&models.Feedback{}).Count(&stats.TotalFeedback)

	// Average rating
	database.DB.Model(&models.Feedback{}).
		Select("COALESCE(AVG(rating), 0)").
		Scan(&stats.AverageRating)

	// Recent feedback (last 7 days)
	weekAgo := time.Now().AddDate(0, 0, -7)
	database.DB.Model(&models.Feedback{}).
		Where("created_at >= ?", weekAgo).
		Count(&stats.RecentFeedback)

	// Rating distribution (1-5 stars)
	for i := 1; i <= 5; i++ {
		var count int64
		database.DB.Model(&models.Feedback{}).Where("rating = ?", i).Count(&count)
		stats.RatingDistribution[i-1] = int(count)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}
