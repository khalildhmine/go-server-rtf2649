package models

import (
	"time"

	"gorm.io/gorm"
)

// WorkerRating represents a rating given by a customer to a worker after service completion
type WorkerRating struct {
	ID              uint           `json:"id" gorm:"primaryKey"`
	CustomerID      uint           `json:"customer_id" gorm:"not null"`
	Customer        User           `json:"customer" gorm:"foreignKey:CustomerID"`
	WorkerID        uint           `json:"worker_id" gorm:"not null"`
	Worker          WorkerProfile  `json:"worker" gorm:"foreignKey:WorkerID"`
	ServiceRequestID uint           `json:"service_request_id" gorm:"not null"`
	ServiceRequest  CustomerServiceRequest `json:"service_request" gorm:"foreignKey:ServiceRequestID"`
	
	// Rating details
	Stars           int            `json:"stars" gorm:"type:int;not null;check:stars >= 1 AND stars <= 5"`
	Comment         string         `json:"comment" gorm:"type:text"`
	ServiceQuality  int            `json:"service_quality" gorm:"type:int;check:service_quality >= 1 AND service_quality <= 5"`
	Professionalism int            `json:"professionalism" gorm:"type:int;check:professionalism >= 1 AND professionalism <= 5"`
	Punctuality     int            `json:"punctuality" gorm:"type:int;check:punctuality >= 1 AND punctuality <= 5"`
	Communication   int            `json:"communication" gorm:"type:int;check:communication >= 1 AND communication <= 5"`
	
	// Metadata
	IsAnonymous     bool           `json:"is_anonymous" gorm:"default:false"`
	IsVerified      bool           `json:"is_verified" gorm:"default:false"` // Service was actually completed
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

// WorkerRatingCreate represents the request structure for creating a worker rating
type WorkerRatingCreate struct {
	ServiceRequestID uint   `json:"service_request_id" binding:"required"`
	Stars            int    `json:"stars" binding:"required,min=1,max=5"`
	Comment          string `json:"comment"`
	ServiceQuality   int    `json:"service_quality" binding:"required,min=1,max=5"`
	Professionalism  int    `json:"professionalism" binding:"required,min=1,max=5"`
	Punctuality      int    `json:"punctuality" binding:"required,min=1,max=5"`
	Communication    int    `json:"communication" binding:"required,min=1,max=5"`
	IsAnonymous      bool   `json:"is_anonymous"`
}

// WorkerRatingResponse represents the response structure for worker rating data
type WorkerRatingResponse struct {
	ID              uint           `json:"id"`
	CustomerID      uint           `json:"customer_id"`
	WorkerID        uint           `json:"worker_id"`
	ServiceRequestID uint           `json:"service_request_id"`
	Stars           int            `json:"stars"`
	Comment         string         `json:"comment"`
	ServiceQuality  int            `json:"service_quality"`
	Professionalism int            `json:"professionalism"`
	Punctuality     int            `json:"punctuality"`
	Communication   int            `json:"communication"`
	IsAnonymous     bool           `json:"is_anonymous"`
	IsVerified      bool           `json:"is_verified"`
	CreatedAt       time.Time      `json:"created_at"`
	Customer        User           `json:"customer,omitempty"`
	Worker          WorkerProfile  `json:"worker,omitempty"`
	ServiceRequest  CustomerServiceRequest `json:"service_request,omitempty"`
}

// WorkerRatingSummary represents a summary of ratings for a worker
type WorkerRatingSummary struct {
	WorkerID        uint    `json:"worker_id"`
	AverageStars    float64 `json:"average_stars"`
	TotalRatings    int     `json:"total_ratings"`
	FiveStarCount   int     `json:"five_star_count"`
	FourStarCount   int     `json:"four_star_count"`
	ThreeStarCount  int     `json:"three_star_count"`
	TwoStarCount    int     `json:"two_star_count"`
	OneStarCount    int     `json:"one_star_count"`
	AverageServiceQuality  float64 `json:"average_service_quality"`
	AverageProfessionalism float64 `json:"average_professionalism"`
	AveragePunctuality     float64 `json:"average_punctuality"`
	AverageCommunication   float64 `json:"average_communication"`
}

// TableName specifies the table name for the WorkerRating model
func (WorkerRating) TableName() string {
	return "worker_ratings"
}
