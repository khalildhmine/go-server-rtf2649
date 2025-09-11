package models

import (
	"time"

	"gorm.io/gorm"
)

// ServiceHistory represents a completed service with detailed tracking information
type ServiceHistory struct {
	ID              uint           `json:"id" gorm:"primaryKey"`
	ServiceRequestID uint           `json:"service_request_id" gorm:"not null;uniqueIndex"`
	ServiceRequest  CustomerServiceRequest `json:"service_request" gorm:"foreignKey:ServiceRequestID"`
	
	// Worker information
	WorkerID        uint           `json:"worker_id" gorm:"not null"`
	Worker          WorkerProfile  `json:"worker" gorm:"foreignKey:WorkerID"`
	
	// Customer information
	CustomerID      uint           `json:"customer_id" gorm:"not null"`
	Customer        User           `json:"customer" gorm:"foreignKey:CustomerID"`
	
	// Service details
	CategoryID      uint           `json:"category_id" gorm:"not null"`
	Category        ServiceCategory `json:"category" gorm:"foreignKey:CategoryID"`
	ServiceOptionID *uint          `json:"service_option_id"`
	ServiceOption   *ServiceOption `json:"service_option,omitempty" gorm:"foreignKey:ServiceOptionID"`
	
	// Service execution details
	Title           string         `json:"title" gorm:"type:varchar(200);not null"`
	Description     string         `json:"description" gorm:"type:text"`
	Priority        string         `json:"priority" gorm:"type:varchar(20);not null"`
	Budget          *float64       `json:"budget" gorm:"type:decimal(10,2)"`
	EstimatedDuration string       `json:"estimated_duration" gorm:"type:varchar(100)"`
	ActualDuration  *int           `json:"actual_duration" gorm:"type:int"` // in minutes
	
	// Location information
	LocationAddress string         `json:"location_address" gorm:"type:text;not null"`
	LocationCity    string         `json:"location_city" gorm:"type:varchar(100);not null"`
	LocationLat     *float64       `json:"location_lat" gorm:"type:decimal(10,8)"`
	LocationLng     *float64       `json:"location_lng" gorm:"type:decimal(11,8)"`
	
	// Timing information
	RequestCreatedAt time.Time     `json:"request_created_at"`
	AssignedAt       *time.Time    `json:"assigned_at"`
	StartedAt        *time.Time    `json:"started_at"`
	CompletedAt      time.Time     `json:"completed_at"`
	
	// Financial information
	AgreedPrice     *float64       `json:"agreed_price" gorm:"type:decimal(10,2)"`
	FinalPrice      *float64       `json:"final_price" gorm:"type:decimal(10,2)"`
	PaymentStatus   string         `json:"payment_status" gorm:"type:varchar(20);default:'pending'"`
	
	// Quality metrics
	CustomerSatisfaction *int      `json:"customer_satisfaction" gorm:"type:int;check:customer_satisfaction >= 1 AND customer_satisfaction <= 5"`
	WorkQuality          *int      `json:"work_quality" gorm:"type:int;check:work_quality >= 1 AND work_quality <= 5"`
	
	// Additional notes
	WorkerNotes     string         `json:"worker_notes" gorm:"type:text"`
	CustomerNotes   string         `json:"customer_notes" gorm:"type:text"`
	
	// Metadata
	IsDisputed      bool           `json:"is_disputed" gorm:"default:false"`
	DisputeReason   string         `json:"dispute_reason" gorm:"type:text"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

// ServiceHistoryCreate represents the request structure for creating a service history entry
type ServiceHistoryCreate struct {
	ServiceRequestID uint      `json:"service_request_id" binding:"required"`
	WorkerID         uint      `json:"worker_id" binding:"required"`
	ActualDuration   *int      `json:"actual_duration"`
	AgreedPrice      *float64  `json:"agreed_price"`
	FinalPrice       *float64  `json:"final_price"`
	PaymentStatus    string    `json:"payment_status"`
	WorkerNotes      string    `json:"worker_notes"`
	CustomerNotes    string    `json:"customer_notes"`
}

// ServiceHistoryResponse represents the response structure for service history data
type ServiceHistoryResponse struct {
	ID              uint           `json:"id"`
	ServiceRequestID uint           `json:"service_request_id"`
	WorkerID        uint           `json:"worker_id"`
	CustomerID      uint           `json:"customer_id"`
	CategoryID      uint           `json:"category_id"`
	ServiceOptionID *uint          `json:"service_option_id"`
	Title           string         `json:"title"`
	Description     string         `json:"description"`
	Priority        string         `json:"priority"`
	Budget          *float64       `json:"budget"`
	EstimatedDuration string       `json:"estimated_duration"`
	ActualDuration  *int           `json:"actual_duration"`
	LocationAddress string         `json:"location_address"`
	LocationCity    string         `json:"location_city"`
	LocationLat     *float64       `json:"location_lat"`
	LocationLng     *float64       `json:"location_lng"`
	RequestCreatedAt time.Time     `json:"request_created_at"`
	AssignedAt       *time.Time    `json:"assigned_at"`
	StartedAt        *time.Time    `json:"started_at"`
	CompletedAt      time.Time     `json:"completed_at"`
	AgreedPrice     *float64       `json:"agreed_price"`
	FinalPrice      *float64       `json:"final_price"`
	PaymentStatus   string         `json:"payment_status"`
	CustomerSatisfaction *int      `json:"customer_satisfaction"`
	WorkQuality          *int      `json:"work_quality"`
	WorkerNotes     string         `json:"worker_notes"`
	CustomerNotes   string         `json:"customer_notes"`
	IsDisputed      bool           `json:"is_disputed"`
	DisputeReason   string         `json:"dispute_reason"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	
	// Relationships
	Worker          WorkerProfile  `json:"worker,omitempty"`
	Customer        User           `json:"customer,omitempty"`
	Category        ServiceCategory `json:"category,omitempty"`
	ServiceOption   *ServiceOption `json:"service_option,omitempty"`
}

// WorkerServiceSummary represents a summary of services for a worker
type WorkerServiceSummary struct {
	WorkerID           uint    `json:"worker_id"`
	TotalServices      int     `json:"total_services"`
	TotalEarnings      float64 `json:"total_earnings"`
	AverageRating      float64 `json:"average_rating"`
	TotalRatings       int     `json:"total_ratings"`
	CompletedThisMonth int     `json:"completed_this_month"`
	CompletedThisYear  int     `json:"completed_this_year"`
	AverageDuration    float64 `json:"average_duration"`
	CustomerSatisfaction float64 `json:"customer_satisfaction"`
}

// TableName specifies the table name for the ServiceHistory model
func (ServiceHistory) TableName() string {
	return "service_histories"
}
