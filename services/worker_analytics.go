package services

import (
	"log"
	"time"

	"gorm.io/gorm"

	"repair-service-server/database"
	"repair-service-server/models"
)

// WorkerAnalyticsService handles all worker performance tracking and analytics
type WorkerAnalyticsService struct {
	db *gorm.DB
}

// NewWorkerAnalyticsService creates a new worker analytics service
func NewWorkerAnalyticsService() *WorkerAnalyticsService {
	return &WorkerAnalyticsService{
		db: database.DB,
	}
}

// TrackJobReceived records when a worker receives a new job opportunity
func (s *WorkerAnalyticsService) TrackJobReceived(workerID uint, serviceRequestID uint) error {
	// Check if this job received has already been tracked
	var existingTracking models.WorkerJobTracking
	err := s.db.Where("worker_id = ? AND service_request_id = ? AND job_type = ?", 
		workerID, serviceRequestID, "received").First(&existingTracking).Error
	
	if err == nil {
		// Job received already tracked, skip to prevent duplicates
		return nil
	}
	
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	
	// Update or create daily stats
	var dailyStats models.WorkerDailyStats
	err = s.db.Where("worker_id = ? AND date = ?", workerID, today).First(&dailyStats).Error
	if err == gorm.ErrRecordNotFound {
		// Create new daily stats
		dailyStats = models.WorkerDailyStats{
			WorkerID: workerID,
			Date:     today,
		}
	}
	
	dailyStats.JobsReceived++
	dailyStats.UpdatedAt = now
	
	if dailyStats.ID == 0 {
		dailyStats.CreatedAt = now
		err = s.db.Create(&dailyStats).Error
	} else {
		err = s.db.Save(&dailyStats).Error
	}
	if err != nil {
		return err
	}
	
	// Update or create monthly stats
	year, month, _ := now.Date()
	var monthlyStats models.WorkerMonthlyStats
	err = s.db.Where("worker_id = ? AND year = ? AND month = ?", workerID, year, month).First(&monthlyStats).Error
	if err == gorm.ErrRecordNotFound {
		// Create new monthly stats
		monthlyStats = models.WorkerMonthlyStats{
			WorkerID: workerID,
			Year:     year,
			Month:    int(month),
		}
	}
	
	monthlyStats.JobsReceived++
	monthlyStats.UpdatedAt = now
	
	if monthlyStats.ID == 0 {
		monthlyStats.CreatedAt = now
		err = s.db.Create(&monthlyStats).Error
	} else {
		err = s.db.Save(&monthlyStats).Error
	}
	if err != nil {
		return err
	}
	
	// Update or create lifetime stats
	var lifetimeStats models.WorkerStats
	err = s.db.Where("worker_id = ?", workerID).First(&lifetimeStats).Error
	if err == gorm.ErrRecordNotFound {
		// Create new lifetime stats
		lifetimeStats = models.WorkerStats{
			WorkerID: workerID,
		}
	}
	
	lifetimeStats.TotalJobsReceived++
	lifetimeStats.DailyJobsReceived = dailyStats.JobsReceived
	lifetimeStats.MonthlyJobsReceived = monthlyStats.JobsReceived
	lifetimeStats.LastJobReceived = &now
	lifetimeStats.UpdatedAt = now
	
	// Calculate response rate
	if lifetimeStats.TotalJobsReceived > 0 {
		lifetimeStats.ResponseRate = float64(lifetimeStats.TotalJobsResponded) / float64(lifetimeStats.TotalJobsReceived) * 100
	}
	
	if lifetimeStats.ID == 0 {
		lifetimeStats.CreatedAt = now
		err = s.db.Create(&lifetimeStats).Error
	} else {
		err = s.db.Save(&lifetimeStats).Error
	}
	
	if err != nil {
		return err
	}
	
	// Create tracking record to prevent duplicate processing
	tracking := models.WorkerJobTracking{
		WorkerID:        workerID,
		ServiceRequestID: serviceRequestID,
		JobType:         "received",
		ProcessedAt:     now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	
	return s.db.Create(&tracking).Error
}

// TrackJobResponse records when a worker responds to a job
func (s *WorkerAnalyticsService) TrackJobResponse(workerID uint, serviceRequestID uint, responseTimeMinutes float64) error {
	// Check if this job response has already been tracked
	var existingTracking models.WorkerJobTracking
	err := s.db.Where("worker_id = ? AND service_request_id = ? AND job_type = ?", 
		workerID, serviceRequestID, "response").First(&existingTracking).Error
	
	if err == nil {
		// Job response already tracked, skip to prevent duplicates
		return nil
	}
	
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	
	// Update or create daily stats
	var dailyStats models.WorkerDailyStats
	err = s.db.Where("worker_id = ? AND date = ?", workerID, today).First(&dailyStats).Error
	if err == gorm.ErrRecordNotFound {
		// Create new daily stats if they don't exist
		dailyStats = models.WorkerDailyStats{
			WorkerID: workerID,
			Date:     today,
		}
	}
	
	dailyStats.JobsResponded++
	dailyStats.TotalResponseTime += responseTimeMinutes
	dailyStats.JobsWithResponse++
	dailyStats.UpdatedAt = now
	
	if dailyStats.ID == 0 {
		dailyStats.CreatedAt = now
		err = s.db.Create(&dailyStats).Error
	} else {
		err = s.db.Save(&dailyStats).Error
	}
	if err != nil {
		return err
	}
	
	// Update or create monthly stats
	year, month, _ := now.Date()
	var monthlyStats models.WorkerMonthlyStats
	err = s.db.Where("worker_id = ? AND year = ? AND month = ?", workerID, year, month).First(&monthlyStats).Error
	if err == gorm.ErrRecordNotFound {
		// Create new monthly stats if they don't exist
		monthlyStats = models.WorkerMonthlyStats{
			WorkerID: workerID,
			Year:     year,
			Month:    int(month),
		}
	}
	
	monthlyStats.JobsResponded++
	monthlyStats.UpdatedAt = now
	
	if monthlyStats.ID == 0 {
		monthlyStats.CreatedAt = now
		err = s.db.Create(&monthlyStats).Error
	} else {
		err = s.db.Save(&monthlyStats).Error
	}
	if err != nil {
		return err
	}
	
	// Update or create lifetime stats
	var lifetimeStats models.WorkerStats
	err = s.db.Where("worker_id = ?", workerID).First(&lifetimeStats).Error
	if err == gorm.ErrRecordNotFound {
		// Create new lifetime stats if they don't exist
		lifetimeStats = models.WorkerStats{
			WorkerID: workerID,
		}
	}
	
	lifetimeStats.TotalJobsResponded++
	lifetimeStats.DailyJobsResponded = dailyStats.JobsResponded
	lifetimeStats.MonthlyJobsResponded = monthlyStats.JobsResponded
	lifetimeStats.LastJobResponded = &now
	lifetimeStats.UpdatedAt = now
	
	// Calculate response rate
	if lifetimeStats.TotalJobsReceived > 0 {
		lifetimeStats.ResponseRate = float64(lifetimeStats.TotalJobsResponded) / float64(lifetimeStats.TotalJobsReceived) * 100
	}
	
	// Calculate average response time
	if lifetimeStats.TotalJobsResponded > 0 {
		// This is a simplified calculation - in production you'd want to store individual response times
		lifetimeStats.AverageResponseTime = (lifetimeStats.AverageResponseTime*float64(lifetimeStats.TotalJobsResponded-1) + responseTimeMinutes) / float64(lifetimeStats.TotalJobsResponded)
	}
	
	if lifetimeStats.ID == 0 {
		lifetimeStats.CreatedAt = now
		err = s.db.Create(&lifetimeStats).Error
	} else {
		err = s.db.Save(&lifetimeStats).Error
	}
	
	if err != nil {
		return err
	}
	
	// Create tracking record to prevent duplicate processing
	tracking := models.WorkerJobTracking{
		WorkerID:        workerID,
		ServiceRequestID: serviceRequestID,
		JobType:         "response",
		ProcessedAt:     now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	
	return s.db.Create(&tracking).Error
}

// TrackJobCompletion records when a worker completes a job
func (s *WorkerAnalyticsService) TrackJobCompletion(workerID uint, serviceRequestID uint, earnings float64, workHours float64) error {
	// Check if this job completion has already been tracked
	var existingTracking models.WorkerJobTracking
	err := s.db.Where("worker_id = ? AND service_request_id = ? AND job_type = ?", 
		workerID, serviceRequestID, "completion").First(&existingTracking).Error
	
	if err == nil {
		// Job completion already tracked, skip to prevent duplicates
		return nil
	}
	
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	
	// Update or create daily stats
	var dailyStats models.WorkerDailyStats
	err = s.db.Where("worker_id = ? AND date = ?", workerID, today).First(&dailyStats).Error
	if err == gorm.ErrRecordNotFound {
		// Create new daily stats if they don't exist
		dailyStats = models.WorkerDailyStats{
			WorkerID: workerID,
			Date:     today,
		}
	}
	
	dailyStats.JobsCompleted++
	dailyStats.Earnings += earnings
	dailyStats.WorkHours += workHours
	dailyStats.UpdatedAt = now
	
	if dailyStats.ID == 0 {
		dailyStats.CreatedAt = now
		err = s.db.Create(&dailyStats).Error
	} else {
		err = s.db.Save(&dailyStats).Error
	}
	if err != nil {
		return err
	}
	
	// Update or create monthly stats
	year, month, _ := now.Date()
	var monthlyStats models.WorkerMonthlyStats
	err = s.db.Where("worker_id = ? AND year = ? AND month = ?", workerID, year, month).First(&monthlyStats).Error
	if err == gorm.ErrRecordNotFound {
		// Create new monthly stats if they don't exist
		monthlyStats = models.WorkerMonthlyStats{
			WorkerID: workerID,
			Year:     year,
			Month:    int(month),
		}
	}
	
	monthlyStats.JobsCompleted++
	monthlyStats.Earnings += earnings
	monthlyStats.WorkHours += workHours
	monthlyStats.UpdatedAt = now
	
	if monthlyStats.ID == 0 {
		monthlyStats.CreatedAt = now
		err = s.db.Create(&monthlyStats).Error
	} else {
		err = s.db.Save(&monthlyStats).Error
	}
	if err != nil {
		return err
	}
	
	// Update or create lifetime stats
	var lifetimeStats models.WorkerStats
	err = s.db.Where("worker_id = ?", workerID).First(&lifetimeStats).Error
	if err == gorm.ErrRecordNotFound {
		// Create new lifetime stats if they don't exist
		lifetimeStats = models.WorkerStats{
			WorkerID: workerID,
		}
	}
	
	lifetimeStats.TotalJobsCompleted++
	lifetimeStats.TotalEarnings += earnings
	lifetimeStats.TotalWorkHours += workHours
	lifetimeStats.DailyJobsCompleted = dailyStats.JobsCompleted
	lifetimeStats.MonthlyJobsCompleted = monthlyStats.JobsCompleted
	lifetimeStats.DailyEarnings = dailyStats.Earnings
	lifetimeStats.MonthlyEarnings = monthlyStats.Earnings
	lifetimeStats.DailyWorkHours = dailyStats.WorkHours
	lifetimeStats.MonthlyWorkHours = monthlyStats.WorkHours
	lifetimeStats.LastJobCompleted = &now
	lifetimeStats.LastEarning = &now
	lifetimeStats.UpdatedAt = now
	
	// Calculate completion rate
	if lifetimeStats.TotalJobsResponded > 0 {
		lifetimeStats.CompletionRate = float64(lifetimeStats.TotalJobsCompleted) / float64(lifetimeStats.TotalJobsResponded) * 100
	}
	
	// Calculate average earnings per job
	if lifetimeStats.TotalJobsCompleted > 0 {
		lifetimeStats.AverageEarningsPerJob = lifetimeStats.TotalEarnings / float64(lifetimeStats.TotalJobsCompleted)
	}
	
	// Calculate average job duration
	if lifetimeStats.TotalJobsCompleted > 0 {
		lifetimeStats.AverageJobDuration = lifetimeStats.TotalWorkHours / float64(lifetimeStats.TotalJobsCompleted)
	}
	
	if lifetimeStats.ID == 0 {
		lifetimeStats.CreatedAt = now
		err = s.db.Create(&lifetimeStats).Error
	} else {
		err = s.db.Save(&lifetimeStats).Error
	}
	
	if err != nil {
		return err
	}
	
	// Create tracking record to prevent duplicate processing
	tracking := models.WorkerJobTracking{
		WorkerID:        workerID,
		ServiceRequestID: serviceRequestID,
		JobType:         "completion",
		ProcessedAt:     now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	
	return s.db.Create(&tracking).Error
}

// TrackJobDecline records when a worker declines or ignores a job
func (s *WorkerAnalyticsService) TrackJobDecline(workerID uint, serviceRequestID uint) error {
	// Check if this job decline has already been tracked
	var existingTracking models.WorkerJobTracking
	err := s.db.Where("worker_id = ? AND service_request_id = ? AND job_type = ?", 
		workerID, serviceRequestID, "declined").First(&existingTracking).Error
	
	if err == nil {
		// Job decline already tracked, skip to prevent duplicates
		return nil
	}
	
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	
	// Update daily stats
	var dailyStats models.WorkerDailyStats
	err = s.db.Where("worker_id = ? AND date = ?", workerID, today).First(&dailyStats).Error
	if err == nil {
		dailyStats.JobsDeclined++
		dailyStats.UpdatedAt = now
		s.db.Save(&dailyStats)
	}
	
	// Update monthly stats
	year, month, _ := now.Date()
	var monthlyStats models.WorkerMonthlyStats
	err = s.db.Where("worker_id = ? AND year = ? AND month = ?", workerID, year, month).First(&monthlyStats).Error
	if err == nil {
		monthlyStats.JobsDeclined++
		monthlyStats.UpdatedAt = now
		s.db.Save(&monthlyStats)
	}
	
	// Update lifetime stats
	var lifetimeStats models.WorkerStats
	err = s.db.Where("worker_id = ?", workerID).First(&lifetimeStats).Error
	if err == nil {
		lifetimeStats.TotalJobsDeclined++
		lifetimeStats.DailyJobsDeclined = dailyStats.JobsDeclined
		lifetimeStats.MonthlyJobsDeclined = monthlyStats.JobsDeclined
		lifetimeStats.UpdatedAt = now
		s.db.Save(&lifetimeStats)
	}
	
	// Create tracking record to prevent duplicate processing
	tracking := models.WorkerJobTracking{
		WorkerID:        workerID,
		ServiceRequestID: serviceRequestID,
		JobType:         "declined",
		ProcessedAt:     now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	
	return s.db.Create(&tracking).Error
}

// UpdateWorkerRating updates worker rating statistics
func (s *WorkerAnalyticsService) UpdateWorkerRating(workerID uint, newRating float64) error {
	var lifetimeStats models.WorkerStats
	err := s.db.Where("worker_id = ?", workerID).First(&lifetimeStats).Error
	if err == nil {
		// Calculate new average rating
		if lifetimeStats.TotalRatings == 0 {
			lifetimeStats.AverageRating = newRating
		} else {
			lifetimeStats.AverageRating = (lifetimeStats.AverageRating*float64(lifetimeStats.TotalRatings) + newRating) / float64(lifetimeStats.TotalRatings+1)
		}
		lifetimeStats.TotalRatings++
		lifetimeStats.UpdatedAt = time.Now()
		s.db.Save(&lifetimeStats)
	}
	
	return nil
}

// GetWorkerPerformanceSummary provides comprehensive worker performance data
func (s *WorkerAnalyticsService) GetWorkerPerformanceSummary(workerID uint) (*models.WorkerPerformanceSummary, error) {
	summary := &models.WorkerPerformanceSummary{
		WorkerID: workerID,
	}
	
	// Get worker profile
	var workerProfile models.WorkerProfile
	err := s.db.Preload("User").Preload("Category").Where("id = ?", workerID).First(&workerProfile).Error
	if err != nil {
		return nil, err
	}
	
	summary.WorkerName = workerProfile.User.FullName
	summary.CategoryName = workerProfile.Category.Name
	
	// Get today's stats
	today := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Now().Location())
	err = s.db.Where("worker_id = ? AND date = ?", workerID, today).First(&summary.TodayStats).Error
	if err == gorm.ErrRecordNotFound {
		// Create empty today stats
		summary.TodayStats = models.WorkerDailyStats{
			WorkerID: workerID,
			Date:     today,
		}
	}
	
	// Get this month's stats
	year, month, _ := time.Now().Date()
	err = s.db.Where("worker_id = ? AND year = ? AND month = ?", workerID, year, month).First(&summary.ThisMonthStats).Error
	if err == gorm.ErrRecordNotFound {
		// Create empty this month stats
		summary.ThisMonthStats = models.WorkerMonthlyStats{
			WorkerID: workerID,
			Year:     year,
			Month:    int(month),
		}
	}
	
	// Get lifetime stats
	err = s.db.Where("worker_id = ?", workerID).First(&summary.LifetimeStats).Error
	if err == gorm.ErrRecordNotFound {
		// Create empty lifetime stats
		summary.LifetimeStats = models.WorkerStats{
			WorkerID: workerID,
		}
	}
	
	// Get last 7 days stats
	sevenDaysAgo := today.AddDate(0, 0, -7)
	err = s.db.Where("worker_id = ? AND date >= ?", workerID, sevenDaysAgo).
		Order("date DESC").
		Find(&summary.Last7DaysStats).Error
	if err != nil {
		log.Printf("Error fetching last 7 days stats: %v", err)
	}
	
	// Get last 6 months stats
	sixMonthsAgo := time.Date(year, month-6, 1, 0, 0, 0, 0, time.Now().Location())
	err = s.db.Where("worker_id = ? AND (year > ? OR (year = ? AND month >= ?))", 
		workerID, sixMonthsAgo.Year(), sixMonthsAgo.Year(), int(sixMonthsAgo.Month())).
		Order("year DESC, month DESC").
		Find(&summary.Last6MonthsStats).Error
	if err != nil {
		log.Printf("Error fetching last 6 months stats: %v", err)
	}
	
	// Calculate performance rankings among workers in same category
	summary.ResponseRateRank = s.calculateResponseRateRank(workerID, workerProfile.CategoryID)
	summary.CompletionRateRank = s.calculateCompletionRateRank(workerID, workerProfile.CategoryID)
	summary.EarningsRank = s.calculateEarningsRank(workerID, workerProfile.CategoryID)
	summary.RatingRank = s.calculateRatingRank(workerID, workerProfile.CategoryID)
	
	// Calculate goal progress (assuming monthly goal of 20 jobs)
	summary.MonthlyGoal = 20
	if summary.MonthlyGoal > 0 {
		summary.GoalProgress = float64(summary.ThisMonthStats.JobsCompleted) / float64(summary.MonthlyGoal) * 100
	}
	
	// Calculate streak days
	summary.StreakDays = s.calculateStreakDays(workerID)
	
	// Get best day and month
	summary.BestDay = s.getBestDay(workerID)
	summary.BestMonth = s.getBestMonth(workerID)
	
	return summary, nil
}

// calculateResponseRateRank calculates worker's rank based on response rate
func (s *WorkerAnalyticsService) calculateResponseRateRank(workerID uint, categoryID uint) int {
	var rank int
	s.db.Raw(`
		SELECT COUNT(*) + 1 as rank
		FROM worker_stats ws1
		JOIN worker_profiles wp1 ON ws1.worker_id = wp1.id
		WHERE wp1.category_id = ? 
		AND ws1.response_rate > (
			SELECT response_rate 
			FROM worker_stats ws2 
			WHERE ws2.worker_id = ?
		)
	`, categoryID, workerID).Scan(&rank)
	return rank
}

// calculateCompletionRateRank calculates worker's rank based on completion rate
func (s *WorkerAnalyticsService) calculateCompletionRateRank(workerID uint, categoryID uint) int {
	var rank int
	s.db.Raw(`
		SELECT COUNT(*) + 1 as rank
		FROM worker_stats ws1
		JOIN worker_profiles wp1 ON ws1.worker_id = wp1.id
		WHERE wp1.category_id = ? 
		AND ws1.completion_rate > (
			SELECT completion_rate 
			FROM worker_stats ws2 
			WHERE ws2.worker_id = ?
		)
	`, categoryID, workerID).Scan(&rank)
	return rank
}

// calculateEarningsRank calculates worker's rank based on total earnings
func (s *WorkerAnalyticsService) calculateEarningsRank(workerID uint, categoryID uint) int {
	var rank int
	s.db.Raw(`
		SELECT COUNT(*) + 1 as rank
		FROM worker_stats ws1
		JOIN worker_profiles wp1 ON ws1.worker_id = wp1.id
		WHERE wp1.category_id = ? 
		AND ws1.total_earnings > (
			SELECT total_earnings 
			FROM worker_stats ws2 
			WHERE ws2.worker_id = ?
		)
	`, categoryID, workerID).Scan(&rank)
	return rank
}

// calculateRatingRank calculates worker's rank based on average rating
func (s *WorkerAnalyticsService) calculateRatingRank(workerID uint, categoryID uint) int {
	var rank int
	s.db.Raw(`
		SELECT COUNT(*) + 1 as rank
		FROM worker_stats ws1
		JOIN worker_profiles wp1 ON ws1.worker_id = wp1.id
		WHERE wp1.category_id = ? 
		AND ws1.average_rating > (
			SELECT average_rating 
			FROM worker_stats ws2 
			WHERE ws2.worker_id = ?
		)
	`, categoryID, workerID).Scan(&rank)
	return rank
}

// calculateStreakDays calculates consecutive days with completed jobs
func (s *WorkerAnalyticsService) calculateStreakDays(workerID uint) int {
	var streak int
	s.db.Raw(`
		WITH RECURSIVE dates AS (
			SELECT DATE(completed_at) as work_date, 1 as streak
			FROM service_histories 
			WHERE worker_id = ? 
			AND DATE(completed_at) = CURDATE()
			
			UNION ALL
			
			SELECT DATE(d.work_date - INTERVAL 1 DAY), d.streak + 1
			FROM dates d
			WHERE EXISTS (
				SELECT 1 FROM service_histories 
				WHERE worker_id = ? 
				AND DATE(completed_at) = d.work_date - INTERVAL 1 DAY
			)
		)
		SELECT MAX(streak) FROM dates
	`, workerID, workerID).Scan(&streak)
	return streak
}

// getBestDay returns the day with highest earnings
func (s *WorkerAnalyticsService) getBestDay(workerID uint) models.WorkerDailyStats {
	var bestDay models.WorkerDailyStats
	s.db.Where("worker_id = ?", workerID).
		Order("earnings DESC").
		First(&bestDay)
	return bestDay
}

// getBestMonth returns the month with highest earnings
func (s *WorkerAnalyticsService) getBestMonth(workerID uint) models.WorkerMonthlyStats {
	var bestMonth models.WorkerMonthlyStats
	s.db.Where("worker_id = ?", workerID).
		Order("earnings DESC").
		First(&bestMonth)
	return bestMonth
}

// GetWorkerLeaderboard returns top workers in a category
func (s *WorkerAnalyticsService) GetWorkerLeaderboard(categoryID uint, limit int) ([]models.WorkerStats, error) {
	var leaderboard []models.WorkerStats
	
	err := s.db.Joins("JOIN worker_profiles wp ON worker_stats.worker_id = wp.id").
		Where("wp.category_id = ?", categoryID).
		Order("total_earnings DESC").
		Limit(limit).
		Preload("Worker.User").
		Preload("Worker.Category").
		Find(&leaderboard).Error
	
	return leaderboard, err
}

// GetWorkerTrends returns performance trends over time
func (s *WorkerAnalyticsService) GetWorkerTrends(workerID uint, days int) ([]models.WorkerDailyStats, error) {
	var trends []models.WorkerDailyStats
	
	startDate := time.Now().AddDate(0, 0, -days)
	err := s.db.Where("worker_id = ? AND date >= ?", workerID, startDate).
		Order("date ASC").
		Find(&trends).Error
	
	return trends, err
}
