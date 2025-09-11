package models

import (
	"time"
)

// WorkerStats tracks comprehensive worker performance metrics
type WorkerStats struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	WorkerID  uint      `json:"worker_id" gorm:"not null;index"`
	Worker    WorkerProfile `json:"worker" gorm:"foreignKey:WorkerID"`
	
	// Lifetime Statistics
	TotalJobsReceived     int     `json:"total_jobs_received" gorm:"default:0"`
	TotalJobsResponded    int     `json:"total_jobs_responded" gorm:"default:0"`
	TotalJobsCompleted    int     `json:"total_jobs_completed" gorm:"default:0"`
	TotalJobsDeclined     int     `json:"total_jobs_declined" gorm:"default:0"`
	TotalEarnings         float64 `json:"total_earnings" gorm:"default:0"`
	TotalWorkHours        float64 `json:"total_work_hours" gorm:"default:0"`
	
	// Monthly Statistics (current month)
	MonthlyJobsReceived   int     `json:"monthly_jobs_received" gorm:"default:0"`
	MonthlyJobsResponded  int     `json:"monthly_jobs_responded" gorm:"default:0"`
	MonthlyJobsCompleted  int     `json:"monthly_jobs_completed" gorm:"default:0"`
	MonthlyJobsDeclined   int     `json:"monthly_jobs_declined" gorm:"default:0"`
	MonthlyEarnings       float64 `json:"monthly_earnings" gorm:"default:0"`
	MonthlyWorkHours      float64 `json:"monthly_work_hours" gorm:"default:0"`
	
	// Daily Statistics (current day)
	DailyJobsReceived     int     `json:"daily_jobs_received" gorm:"default:0"`
	DailyJobsResponded    int     `json:"daily_jobs_responded" gorm:"default:0"`
	DailyJobsCompleted    int     `json:"daily_jobs_completed" gorm:"default:0"`
	DailyJobsDeclined     int     `json:"daily_jobs_declined" gorm:"default:0"`
	DailyEarnings         float64 `json:"daily_earnings" gorm:"default:0"`
	DailyWorkHours        float64 `json:"daily_work_hours" gorm:"default:0"`
	
	// Performance Metrics
	ResponseRate          float64 `json:"response_rate" gorm:"default:0"` // Percentage of jobs responded to
	CompletionRate        float64 `json:"completion_rate" gorm:"default:0"` // Percentage of responded jobs completed
	AverageResponseTime   float64 `json:"average_response_time" gorm:"default:0"` // Average time to respond in minutes
	AverageJobDuration    float64 `json:"average_job_duration" gorm:"default:0"` // Average job completion time in hours
	AverageEarningsPerJob float64 `json:"average_earnings_per_job" gorm:"default:0"`
	
	// Customer Satisfaction
	AverageRating         float64 `json:"average_rating" gorm:"default:0"`
	TotalRatings          int     `json:"total_ratings" gorm:"default:0"`
	
	// Last Updated
	LastJobReceived       *time.Time `json:"last_job_received"`
	LastJobResponded      *time.Time `json:"last_job_responded"`
	LastJobCompleted      *time.Time `json:"last_job_completed"`
	LastEarning           *time.Time `json:"last_earning"`
	
	// Timestamps
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
	DeletedAt             *time.Time `json:"deleted_at" gorm:"index"`
}

// WorkerDailyStats tracks daily performance for trend analysis
type WorkerDailyStats struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	WorkerID  uint      `json:"worker_id" gorm:"not null;index"`
	Date      time.Time `json:"date" gorm:"not null;index"`
	
	// Daily Metrics
	JobsReceived     int     `json:"jobs_received"`
	JobsResponded    int     `json:"jobs_responded"`
	JobsCompleted    int     `json:"jobs_completed"`
	JobsDeclined     int     `json:"jobs_declined"`
	Earnings         float64 `json:"earnings"`
	WorkHours        float64 `json:"work_hours"`
	AverageRating    float64 `json:"average_rating"`
	
	// Response Time Metrics
	TotalResponseTime float64 `json:"total_response_time"` // Total response time in minutes
	JobsWithResponse  int     `json:"jobs_with_response"`  // Jobs that had response time tracked
	
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at" gorm:"index"`
}

// WorkerMonthlyStats tracks monthly performance for trend analysis
type WorkerMonthlyStats struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	WorkerID  uint      `json:"worker_id" gorm:"not null;index"`
	Year      int       `json:"year" gorm:"not null;index"`
	Month     int       `json:"month" gorm:"not null;index"`
	
	// Monthly Metrics
	JobsReceived     int     `json:"jobs_received"`
	JobsResponded    int     `json:"jobs_responded"`
	JobsCompleted    int     `json:"jobs_completed"`
	JobsDeclined     int     `json:"jobs_declined"`
	Earnings         float64 `json:"earnings"`
	WorkHours        float64 `json:"work_hours"`
	AverageRating    float64 `json:"average_rating"`
	
	// Performance Metrics
	ResponseRate          float64 `json:"response_rate"`
	CompletionRate        float64 `json:"completion_rate"`
	AverageResponseTime   float64 `json:"average_response_time"`
	AverageJobDuration    float64 `json:"average_job_duration"`
	AverageEarningsPerJob float64 `json:"average_earnings_per_job"`
	
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at" gorm:"index"`
}

// WorkerPerformanceSummary provides a comprehensive overview
type WorkerPerformanceSummary struct {
	WorkerID              uint      `json:"worker_id"`
	WorkerName            string    `json:"worker_name"`
	CategoryName          string    `json:"category_name"`
	
	// Current Period Stats
	TodayStats            WorkerDailyStats  `json:"today_stats"`
	ThisMonthStats        WorkerMonthlyStats `json:"this_month_stats"`
	
	// Lifetime Stats
	LifetimeStats         WorkerStats        `json:"lifetime_stats"`
	
	// Recent Performance (Last 7 days)
	Last7DaysStats        []WorkerDailyStats `json:"last_7_days_stats"`
	
	// Monthly Trends (Last 6 months)
	Last6MonthsStats      []WorkerMonthlyStats `json:"last_6_months_stats"`
	
	// Performance Rankings
	ResponseRateRank      int     `json:"response_rate_rank"`      // Rank among workers in same category
	CompletionRateRank    int     `json:"completion_rate_rank"`    // Rank among workers in same category
	EarningsRank          int     `json:"earnings_rank"`           // Rank among workers in same category
	RatingRank            int     `json:"rating_rank"`             // Rank among workers in same category
	
	// Goals and Achievements
	MonthlyGoal           int     `json:"monthly_goal"`            // Target jobs for current month
	GoalProgress          float64 `json:"goal_progress"`           // Percentage of goal achieved
	StreakDays            int     `json:"streak_days"`             // Consecutive days with completed jobs
	BestDay               WorkerDailyStats `json:"best_day"`       // Day with highest earnings
	BestMonth             WorkerMonthlyStats `json:"best_month"`   // Month with highest earnings
}

// TableName specifies the table name for WorkerStats
func (WorkerStats) TableName() string {
	return "worker_stats"
}

// TableName specifies the table name for WorkerDailyStats
func (WorkerDailyStats) TableName() string {
	return "worker_daily_stats"
}

// TableName specifies the table name for WorkerMonthlyStats
func (WorkerMonthlyStats) TableName() string {
	return "worker_monthly_stats"
}

// WorkerJobTracking tracks which jobs have been processed for analytics to prevent duplicates
type WorkerJobTracking struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	WorkerID        uint      `json:"worker_id" gorm:"not null;index"`
	ServiceRequestID uint      `json:"service_request_id" gorm:"not null;index"`
	JobType         string    `json:"job_type" gorm:"not null"` // "completion", "response", "received", "declined"
	ProcessedAt     time.Time `json:"processed_at"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	DeletedAt       *time.Time `json:"deleted_at" gorm:"index"`
}

// TableName specifies the table name for WorkerJobTracking
func (WorkerJobTracking) TableName() string {
	return "worker_job_tracking"
}
