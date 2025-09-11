package routes

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"repair-service-server/database"
	"repair-service-server/models"
	"repair-service-server/services"
)

// RegisterWorkerAnalyticsRoutes registers all worker analytics routes
func RegisterWorkerAnalyticsRoutes(router *gin.RouterGroup) {
	analyticsRoutes := router.Group("/analytics")
	{
		// Get comprehensive worker performance summary
		analyticsRoutes.GET("/performance", getWorkerPerformanceSummary)
		
		// Get worker statistics breakdown
		analyticsRoutes.GET("/stats", getWorkerStats)
		
		// Get daily performance trends
		analyticsRoutes.GET("/trends/daily", getWorkerDailyTrends)
		
		// Get monthly performance trends
		analyticsRoutes.GET("/trends/monthly", getWorkerMonthlyTrends)
		
		// Get worker leaderboard in their category
		analyticsRoutes.GET("/leaderboard", getWorkerLeaderboard)
		
		// Get earnings breakdown
		analyticsRoutes.GET("/earnings", getWorkerEarningsBreakdown)
		
		// Get job performance metrics
		analyticsRoutes.GET("/jobs", getWorkerJobMetrics)
		
		// Get customer satisfaction metrics
		analyticsRoutes.GET("/satisfaction", getWorkerSatisfactionMetrics)
		
		// Get productivity insights
		analyticsRoutes.GET("/productivity", getWorkerProductivityInsights)
		
		// Backfill historical analytics data
		analyticsRoutes.POST("/backfill", backfillWorkerAnalytics)
	}
}

// getWorkerPerformanceSummary provides comprehensive worker performance overview
func getWorkerPerformanceSummary(c *gin.Context) {
	userID := c.GetUint("user_id")
	
	// Get worker profile first
	var workerProfile models.WorkerProfile
	if err := database.DB.Where("user_id = ?", userID).First(&workerProfile).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Worker profile not found"})
		return
	}
	
	analyticsService := services.NewWorkerAnalyticsService()
	summary, err := analyticsService.GetWorkerPerformanceSummary(workerProfile.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch performance summary"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"summary": summary,
		},
	})
}

// getWorkerStats provides detailed worker statistics
func getWorkerStats(c *gin.Context) {
	userID := c.GetUint("user_id")
	period := c.DefaultQuery("period", "lifetime") // lifetime, monthly, daily
	
	// Get worker profile first
	var workerProfile models.WorkerProfile
	if err := database.DB.Where("user_id = ?", userID).First(&workerProfile).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Worker profile not found"})
		return
	}
	
	analyticsService := services.NewWorkerAnalyticsService()
	
	var stats interface{}
	var err error
	
	switch period {
	case "daily":
		stats, err = analyticsService.GetWorkerTrends(workerProfile.ID, 30) // Last 30 days
	case "monthly":
		// For monthly, we'll get lifetime stats for now
		summary, err := analyticsService.GetWorkerPerformanceSummary(workerProfile.ID)
		if err == nil {
			stats = summary.LifetimeStats
		}
	default:
		summary, err := analyticsService.GetWorkerPerformanceSummary(workerProfile.ID)
		if err == nil {
			stats = summary.LifetimeStats
		}
	}
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch statistics"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"period":  period,
			"stats":   stats,
		},
	})
}

// getWorkerDailyTrends provides daily performance trends
func getWorkerDailyTrends(c *gin.Context) {
	userID := c.GetUint("user_id")
	daysStr := c.DefaultQuery("days", "30")
	
	days, err := strconv.Atoi(daysStr)
	if err != nil || days <= 0 || days > 365 {
		days = 30
	}
	
	// Get worker profile first
	var workerProfile models.WorkerProfile
	if err := database.DB.Where("user_id = ?", userID).First(&workerProfile).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Worker profile not found"})
		return
	}
	
	analyticsService := services.NewWorkerAnalyticsService()
	trends, err := analyticsService.GetWorkerTrends(workerProfile.ID, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch daily trends"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"days":    days,
		"trends":  trends,
	})
}

// getWorkerMonthlyTrends provides monthly performance trends
func getWorkerMonthlyTrends(c *gin.Context) {
	monthsStr := c.DefaultQuery("months", "12")
	
	months, err := strconv.Atoi(monthsStr)
	if err != nil || months <= 0 || months > 60 {
		months = 12
	}
	
	// This would need to be implemented in the analytics service
	// For now, we'll return a placeholder
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"months":  months,
		"message": "Monthly trends endpoint - implementation in progress",
	})
}

// getWorkerLeaderboard provides worker ranking in their category
func getWorkerLeaderboard(c *gin.Context) {
	userID := c.GetUint("user_id")
	limitStr := c.DefaultQuery("limit", "10")
	
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 50 {
		limit = 10
	}
	
	// Get worker's category first
	var categoryID uint
	err = database.DB.Model(&models.WorkerProfile{}).
		Where("user_id = ?", userID).
		Select("category_id").
		Scan(&categoryID).Error
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get worker category"})
		return
	}
	
	analyticsService := services.NewWorkerAnalyticsService()
	leaderboard, err := analyticsService.GetWorkerLeaderboard(categoryID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch leaderboard"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"category_id": categoryID,
		"limit":       limit,
		"leaderboard": leaderboard,
	})
}

// getWorkerEarningsBreakdown provides detailed earnings analysis
func getWorkerEarningsBreakdown(c *gin.Context) {
	userID := c.GetUint("user_id")
	period := c.DefaultQuery("period", "monthly") // daily, weekly, monthly, yearly
	
	// Get worker profile first
	var workerProfile models.WorkerProfile
	if err := database.DB.Where("user_id = ?", userID).First(&workerProfile).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Worker profile not found"})
		return
	}
	
	// Get earnings data from service history
	var earnings []struct {
		Date   time.Time `json:"date"`
		Amount float64   `json:"amount"`
		Jobs   int       `json:"jobs"`
	}
	
	query := database.DB.Table("service_histories").
		Select("DATE(completed_at) as date, SUM(final_price) as amount, COUNT(*) as jobs").
		Where("worker_id = ?", workerProfile.ID).
		Group("DATE(completed_at)").
		Order("date DESC")
	
	switch period {
	case "daily":
		query = query.Limit(30) // Last 30 days
	case "weekly":
		query = query.Where("completed_at >= ?", time.Now().AddDate(0, 0, -7))
	case "monthly":
		query = query.Where("completed_at >= ?", time.Now().AddDate(0, -1, 0))
	case "yearly":
		query = query.Where("completed_at >= ?", time.Now().AddDate(-1, 0, 0))
	}
	
	err := query.Find(&earnings).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch earnings data"})
		return
	}
	
	// Calculate totals
	var totalEarnings float64
	var totalJobs int
	for _, e := range earnings {
		totalEarnings += e.Amount
		totalJobs += e.Jobs
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"period":         period,
			"total_earnings": totalEarnings,
			"total_jobs":     totalJobs,
			"average_per_job": func() float64 {
				if totalJobs > 0 {
					return totalEarnings / float64(totalJobs)
				}
				return 0
			}(),
			"breakdown": earnings,
		},
	})
}

// getWorkerJobMetrics provides detailed job performance metrics
func getWorkerJobMetrics(c *gin.Context) {
	userID := c.GetUint("user_id")
	
	// Get worker profile first
	var workerProfile models.WorkerProfile
	if err := database.DB.Where("user_id = ?", userID).First(&workerProfile).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Worker profile not found"})
		return
	}
	
	// Get comprehensive job metrics
	var metrics struct {
		TotalReceived     int64   `json:"total_received"`
		TotalResponded    int64   `json:"total_responded"`
		TotalCompleted    int64   `json:"total_completed"`
		TotalDeclined     int64   `json:"total_declined"`
		ResponseRate      float64 `json:"response_rate"`
		CompletionRate    float64 `json:"completion_rate"`
		AverageResponseTime float64 `json:"average_response_time"`
		AverageJobDuration float64 `json:"average_job_duration"`
	}
	
	// Get counts from different sources
	database.DB.Model(&models.CustomerServiceRequest{}).
		Where("category_id = ?", workerProfile.CategoryID).
		Count(&metrics.TotalReceived)
	
	database.DB.Model(&models.CustomerServiceRequest{}).
		Where("assigned_worker_id = ?", workerProfile.ID).
		Count(&metrics.TotalResponded)
	
	database.DB.Model(&models.CustomerServiceRequest{}).
		Where("assigned_worker_id = ? AND status = ?", workerProfile.ID, "completed").
		Count(&metrics.TotalCompleted)
	
	// Calculate rates
	if metrics.TotalReceived > 0 {
		metrics.ResponseRate = float64(metrics.TotalResponded) / float64(metrics.TotalReceived) * 100
	}
	if metrics.TotalResponded > 0 {
		metrics.CompletionRate = float64(metrics.TotalCompleted) / float64(metrics.TotalResponded) * 100
	}
	
	// Get average response time (PostgreSQL compatible) - using worker responses
	var avgResponseTime float64
	database.DB.Raw(`
		SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (wr.responded_at - csr.created_at)) / 60), 0) as avg_response_time
		FROM customer_service_requests csr
		JOIN worker_responses wr ON csr.id = wr.service_request_id
		WHERE wr.worker_id = ? AND wr.response = 'accept'
	`, workerProfile.ID).Scan(&avgResponseTime)
	metrics.AverageResponseTime = avgResponseTime
	
	// Get average job duration (PostgreSQL compatible)
	var avgJobDuration float64
	database.DB.Raw(`
		SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (completed_at - started_at)) / 3600), 0) as avg_job_duration
		FROM customer_service_requests 
		WHERE assigned_worker_id = ? AND started_at IS NOT NULL AND completed_at IS NOT NULL
	`, workerProfile.ID).Scan(&avgJobDuration)
	metrics.AverageJobDuration = avgJobDuration
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"metrics": metrics,
		},
	})
}

// getWorkerSatisfactionMetrics provides customer satisfaction insights
func getWorkerSatisfactionMetrics(c *gin.Context) {
	userID := c.GetUint("user_id")
	
	// Get worker profile first
	var workerProfile models.WorkerProfile
	if err := database.DB.Where("user_id = ?", userID).First(&workerProfile).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Worker profile not found"})
		return
	}
	
	// Get rating statistics
	var ratingStats struct {
		AverageRating     float64 `json:"average_rating"`
		TotalRatings      int64   `json:"total_ratings"`
		FiveStarCount     int64   `json:"five_star_count"`
		FourStarCount     int64   `json:"four_star_count"`
		ThreeStarCount    int64   `json:"three_star_count"`
		TwoStarCount      int64   `json:"two_star_count"`
		OneStarCount      int64   `json:"one_star_count"`
		ServiceQuality    float64 `json:"average_service_quality"`
		Professionalism   float64 `json:"average_professionalism"`
		Punctuality       float64 `json:"average_punctuality"`
		Communication     float64 `json:"average_communication"`
	}
	
	// Get overall rating stats
	database.DB.Raw(`
		SELECT 
			COALESCE(AVG(stars), 0) as average_rating,
			COUNT(*) as total_ratings,
			COALESCE(SUM(CASE WHEN stars = 5 THEN 1 ELSE 0 END), 0) as five_star_count,
			COALESCE(SUM(CASE WHEN stars = 4 THEN 1 ELSE 0 END), 0) as four_star_count,
			COALESCE(SUM(CASE WHEN stars = 3 THEN 1 ELSE 0 END), 0) as three_star_count,
			COALESCE(SUM(CASE WHEN stars = 2 THEN 1 ELSE 0 END), 0) as two_star_count,
			COALESCE(SUM(CASE WHEN stars = 1 THEN 1 ELSE 0 END), 0) as one_star_count,
			COALESCE(AVG(service_quality), 0) as average_service_quality,
			COALESCE(AVG(professionalism), 0) as average_professionalism,
			COALESCE(AVG(punctuality), 0) as average_punctuality,
			COALESCE(AVG(communication), 0) as average_communication
		FROM worker_ratings 
		WHERE worker_id = ?
	`, workerProfile.ID).Scan(&ratingStats)
	
	// Get recent ratings for trend analysis
	var recentRatings []models.WorkerRating
	database.DB.Where("worker_id = ?", workerProfile.ID).
		Order("created_at DESC").
		Limit(10).
		Preload("Customer").
		Preload("ServiceRequest").
		Find(&recentRatings)
	
	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"data": gin.H{
			"rating_stats":  ratingStats,
			"recent_ratings": recentRatings,
		},
	})
}

// getWorkerProductivityInsights provides productivity and efficiency insights
func getWorkerProductivityInsights(c *gin.Context) {
	userID := c.GetUint("user_id")
	
	// Get worker profile first
	var workerProfile models.WorkerProfile
	if err := database.DB.Where("user_id = ?", userID).First(&workerProfile).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Worker profile not found"})
		return
	}
	
	// Get productivity metrics
	var productivity struct {
		JobsPerDay        float64 `json:"jobs_per_day"`
		JobsPerWeek       float64 `json:"jobs_per_week"`
		JobsPerMonth      float64 `json:"jobs_per_month"`
		EarningsPerHour   float64 `json:"earnings_per_hour"`
		EfficiencyScore   float64 `json:"efficiency_score"`
		PeakHours         []int   `json:"peak_hours"`
		BestDays          []string `json:"best_days"`
		ProductivityTrend string  `json:"productivity_trend"`
	}
	
	// Calculate jobs per time period
	var weeklyJobs, monthlyJobs int64
	today := time.Now()
	
	// Jobs per day (last 7 days average)
	database.DB.Model(&models.CustomerServiceRequest{}).
		Where("assigned_worker_id = ? AND completed_at >= ?", workerProfile.ID, today.AddDate(0, 0, -7)).
		Count(&weeklyJobs)
	productivity.JobsPerWeek = float64(weeklyJobs)
	productivity.JobsPerDay = float64(weeklyJobs) / 7.0
	
	// Jobs per month (last 30 days)
	database.DB.Model(&models.CustomerServiceRequest{}).
		Where("assigned_worker_id = ? AND completed_at >= ?", workerProfile.ID, today.AddDate(0, 0, -30)).
		Count(&monthlyJobs)
	productivity.JobsPerMonth = float64(monthlyJobs)
	
	// Earnings per hour
	var totalEarnings, totalHours float64
	database.DB.Raw(`
		SELECT COALESCE(SUM(final_price), 0) as total_earnings
		FROM service_histories 
		WHERE worker_id = ? AND completed_at >= ?
	`, workerProfile.ID, today.AddDate(0, 0, -30)).Scan(&totalEarnings)
	database.DB.Raw(`
		SELECT COALESCE(SUM(EXTRACT(EPOCH FROM (completed_at - started_at)) / 3600), 0) as total_hours
		FROM service_histories 
		WHERE worker_id = ? AND completed_at >= ?
	`, workerProfile.ID, today.AddDate(0, 0, -30)).Scan(&totalHours)
	
	if totalHours > 0 {
		productivity.EarningsPerHour = totalEarnings / totalHours
	}
	
	// Calculate efficiency score (combination of response rate, completion rate, and rating)
	var responseRate, completionRate, avgRating float64
	database.DB.Raw(`
		SELECT 
			(COUNT(CASE WHEN assigned_worker_id IS NOT NULL THEN 1 END) * 100.0 / COUNT(*)) as response_rate
		FROM customer_service_requests 
		WHERE category_id = ?
	`, workerProfile.CategoryID).Scan(&responseRate)
	database.DB.Raw(`
		SELECT 
			(COUNT(CASE WHEN status = 'completed' THEN 1 END) * 100.0 / COUNT(CASE WHEN assigned_worker_id IS NOT NULL THEN 1 END)) as completion_rate
		FROM customer_service_requests 
		WHERE category_id = ?
	`, workerProfile.CategoryID).Scan(&completionRate)
	
	database.DB.Raw(`
		SELECT COALESCE(AVG(stars), 0) as avg_rating
		FROM worker_ratings 
		WHERE worker_id = ?
	`, workerProfile.ID).Scan(&avgRating)
	
	// Efficiency score: 40% response rate + 40% completion rate + 20% rating
	productivity.EfficiencyScore = (responseRate * 0.4) + (completionRate * 0.4) + (avgRating * 20)
	
	// Determine productivity trend
	if productivity.JobsPerMonth > productivity.JobsPerWeek*4.3 { // More than weekly average
		productivity.ProductivityTrend = "increasing"
	} else if productivity.JobsPerMonth < productivity.JobsPerWeek*4.3 {
		productivity.ProductivityTrend = "decreasing"
	} else {
		productivity.ProductivityTrend = "stable"
	}
	
	// Get peak hours (simplified - in production you'd analyze actual job timing data)
	productivity.PeakHours = []int{9, 10, 11, 14, 15, 16} // Typical business hours
	productivity.BestDays = []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday"}
	
	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"data": gin.H{
			"productivity": productivity,
		},
	})
}

// backfillWorkerAnalytics populates analytics tables with historical data
func backfillWorkerAnalytics(c *gin.Context) {
	userID := c.GetUint("user_id")
	
	// Get worker profile first
	var workerProfile models.WorkerProfile
	if err := database.DB.Where("user_id = ?", userID).First(&workerProfile).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Worker profile not found"})
		return
	}

	fmt.Printf("🔄 Starting backfill for worker %d (profile ID: %d)\n", userID, workerProfile.ID)

	// Clear existing analytics data for this worker to prevent accumulation
	fmt.Printf("🧹 Clearing existing analytics data for worker %d\n", workerProfile.ID)
	
	// Delete existing tracking records
	if err := database.DB.Where("worker_id = ?", workerProfile.ID).Delete(&models.WorkerJobTracking{}).Error; err != nil {
		fmt.Printf("⚠️ Failed to clear job tracking records: %v\n", err)
	}
	
	// Reset daily stats
	if err := database.DB.Where("worker_id = ?", workerProfile.ID).Delete(&models.WorkerDailyStats{}).Error; err != nil {
		fmt.Printf("⚠️ Failed to clear daily stats: %v\n", err)
	}
	
	// Reset monthly stats
	if err := database.DB.Where("worker_id = ?", workerProfile.ID).Delete(&models.WorkerMonthlyStats{}).Error; err != nil {
		fmt.Printf("⚠️ Failed to clear monthly stats: %v\n", err)
	}
	
	// Reset lifetime stats
	if err := database.DB.Where("worker_id = ?", workerProfile.ID).Delete(&models.WorkerStats{}).Error; err != nil {
		fmt.Printf("⚠️ Failed to clear lifetime stats: %v\n", err)
	}

	// Get all completed service histories for this worker
	var serviceHistories []models.ServiceHistory
	if err := database.DB.Where("worker_id = ?", workerProfile.ID).Find(&serviceHistories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch service histories"})
		return
	}

	fmt.Printf("📊 Found %d completed services to backfill\n", len(serviceHistories))

	// Initialize analytics service
	analyticsService := services.NewWorkerAnalyticsService()

	// Process each service
	for _, service := range serviceHistories {
		// Calculate work hours (use estimated duration if actual not available)
		var workHours float64
		if service.ActualDuration != nil {
			workHours = float64(*service.ActualDuration) / 60.0 // Convert minutes to hours
		} else if service.EstimatedDuration != "" {
			// Parse estimated duration string to float
			if duration, err := strconv.ParseFloat(service.EstimatedDuration, 64); err == nil {
				workHours = duration / 60.0
			} else {
				workHours = 1.0 // Default to 1 hour
			}
		} else {
			workHours = 1.0 // Default to 1 hour
		}

		// Get earnings from final price
		earnings := 0.0
		if service.FinalPrice != nil {
			earnings = *service.FinalPrice
		}

		// Track job completion
		if err := analyticsService.TrackJobCompletion(workerProfile.ID, service.ServiceRequestID, earnings, workHours); err != nil {
			fmt.Printf("⚠️ Failed to track completion for service %d: %v\n", service.ID, err)
		}

		// Track job response (assume worker responded when they accepted)
		// Calculate response time as time between service creation and completion
		responseTime := service.CompletedAt.Sub(service.RequestCreatedAt).Minutes()
		if responseTime > 0 {
			if err := analyticsService.TrackJobResponse(workerProfile.ID, service.ServiceRequestID, responseTime); err != nil {
				fmt.Printf("⚠️ Failed to track response for service %d: %v\n", service.ID, err)
			}
		}

		// Track job received (assume worker received it when service was created)
		if err := analyticsService.TrackJobReceived(workerProfile.ID, service.ServiceRequestID); err != nil {
			fmt.Printf("⚠️ Failed to track job received for service %d: %v\n", service.ID, err)
		}
	}

	// Also backfill ratings data
	var ratings []models.WorkerRating
	if err := database.DB.Where("worker_id = ?", workerProfile.ID).Find(&ratings).Error; err == nil {
		fmt.Printf("📝 Found %d ratings to backfill\n", len(ratings))
		
		for _, rating := range ratings {
			if err := analyticsService.UpdateWorkerRating(workerProfile.ID, float64(rating.Stars)); err != nil {
				fmt.Printf("⚠️ Failed to update rating: %v\n", err)
			}
		}
	}

	fmt.Printf("✅ Backfill completed for worker %d\n", workerProfile.ID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"message": "Analytics backfill completed successfully",
			"services_processed": len(serviceHistories),
			"ratings_processed": len(ratings),
		},
	})
}
