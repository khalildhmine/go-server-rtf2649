package models

import (
	"time"

	"gorm.io/gorm"
)

// CustomerServiceRequestStatus represents the current status of a customer service request
type CustomerServiceRequestStatus string

const (
	RequestStatusPending    CustomerServiceRequestStatus = "pending"
	RequestStatusBroadcast  CustomerServiceRequestStatus = "broadcast"
	RequestStatusAccepted   CustomerServiceRequestStatus = "accepted"
	RequestStatusInProgress CustomerServiceRequestStatus = "in_progress"
	RequestStatusCompleted  CustomerServiceRequestStatus = "completed"
	RequestStatusCancelled  CustomerServiceRequestStatus = "cancelled"
	RequestStatusExpired    CustomerServiceRequestStatus = "expired"
	RequestStatusScheduled  CustomerServiceRequestStatus = "scheduled"
)

// CustomerServiceRequest represents a service request from a customer
type CustomerServiceRequest struct {
	ID              uint           `json:"id" gorm:"primaryKey"`
	CustomerID      uint           `json:"customer_id" gorm:"not null"`
	Customer        User           `json:"customer" gorm:"foreignKey:CustomerID"`
	CategoryID      uint           `json:"category_id" gorm:"not null"`
	Category        ServiceCategory `json:"category" gorm:"foreignKey:CategoryID"`
	ServiceOptionID *uint          `json:"service_option_id"` // New: Selected service option
	ServiceOption   *ServiceOption `json:"service_option,omitempty" gorm:"foreignKey:ServiceOptionID"` // New: Service option details
	Title           string         `json:"title" gorm:"type:varchar(200);not null"`
	Description     string         `json:"description" gorm:"type:text"`
	Priority        string         `json:"priority" gorm:"type:varchar(20);not null"` // low, medium, high, urgent
	Budget          *float64       `json:"budget" gorm:"type:decimal(10,2)"`
	EstimatedDuration string       `json:"estimated_duration" gorm:"type:varchar(100)"`
	LocationAddress string         `json:"location_address" gorm:"type:text;not null"`
	LocationCity    string         `json:"location_city" gorm:"type:varchar(100);not null"`
	LocationLat     *float64       `json:"location_lat" gorm:"type:decimal(10,8)"`
	LocationLng     *float64       `json:"location_lng" gorm:"type:decimal(11,8)"`
	Status          CustomerServiceRequestStatus `json:"status" gorm:"type:varchar(20);not null;default:'broadcast'"` // broadcast, assigned, in_progress, completed, cancelled
	AssignedWorkerID *uint         `json:"assigned_worker_id"`
	AssignedWorker  *WorkerProfile `json:"assigned_worker,omitempty" gorm:"foreignKey:AssignedWorkerID"`
	StartedAt       *time.Time     `json:"started_at"`
	CompletedAt     *time.Time     `json:"completed_at"`
	ExpiresAt       *time.Time     `json:"expires_at"`
	ScheduledFor    *time.Time     `json:"scheduled_for"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

// CustomerServiceRequestCreate represents the request structure for creating a customer service request
type CustomerServiceRequestCreate struct {
	CategoryID       uint     `json:"category_id" binding:"required"`
	ServiceOptionID  *uint    `json:"service_option_id"` // New: Selected service option ID
	Title            string   `json:"title" binding:"required"`
	Description      string   `json:"description"`
	Priority         string   `json:"priority"`
	Budget           *float64 `json:"budget"`
	EstimatedDuration string  `json:"estimated_duration"`
	LocationLat      float64  `json:"location_lat" binding:"required"`
	LocationLng      float64  `json:"location_lng" binding:"required"`
	LocationAddress  string   `json:"location_address" binding:"required"`
	LocationCity     string   `json:"location_city" binding:"required"`
}

// CustomerServiceRequestResponse represents the response structure for customer service request data
type CustomerServiceRequestResponse struct {
	ID              uint                           `json:"id"`
	CustomerID      uint                           `json:"customer_id"`
	ServiceCategory WorkerCategory                 `json:"service_category"`
	Title           string                         `json:"title"`
	Description     string                         `json:"description"`
	Notes           string                         `json:"notes"`
	LocationLat     float64                        `json:"location_lat"`
	LocationLng     float64                        `json:"location_lng"`
	LocationAddress string                         `json:"location_address"`
	LocationCity    string                         `json:"location_city"`
	IsImmediate     bool                           `json:"is_immediate"`
	ScheduledDate   *time.Time                     `json:"scheduled_date"`
	ScheduledTime   *time.Time                     `json:"scheduled_time"`
	PreferredTime   string                         `json:"preferred_time"`
	Status          CustomerServiceRequestStatus   `json:"status"`
	Priority        string                         `json:"priority"`
	Budget          *float64                       `json:"budget"`
	EstimatedDuration string                       `json:"estimated_duration"`
	AssignedWorkerID *uint                         `json:"assigned_worker_id"`
	AcceptedAt       *time.Time                    `json:"accepted_at"`
	StartedAt        *time.Time                    `json:"started_at"`
	CompletedAt      *time.Time                    `json:"completed_at"`
	BroadcastRadius  float64                       `json:"broadcast_radius"`
	BroadcastedAt   *time.Time                     `json:"broadcasted_at"`
	ExpiresAt       *time.Time                     `json:"expires_at"`
	CustomerRating  *float64                       `json:"customer_rating"`
	CustomerReview  string                         `json:"customer_review"`
	CreatedAt       time.Time                      `json:"created_at"`
	UpdatedAt       time.Time                      `json:"updated_at"`
	Customer        User                           `json:"customer,omitempty"`
	AssignedWorker *WorkerProfile                 `json:"assigned_worker,omitempty"`
}

// WorkerResponse represents a worker's response to a customer service request
type WorkerResponse struct {
	ID              uint                     `json:"id" gorm:"primaryKey"`
	ServiceRequestID uint                    `json:"service_request_id" gorm:"not null"`
	WorkerID        uint                     `json:"worker_id" gorm:"not null"`
	Response        string                   `json:"response" gorm:"type:varchar(20);not null"` // "accept", "decline", "interested"
	Message         string                   `json:"message" gorm:"type:text"`
	ProposedPrice   *float64                 `json:"proposed_price" gorm:"type:decimal(10,2)"`
	ProposedTime    *time.Time               `json:"proposed_time"`
	Distance        float64                  `json:"distance" gorm:"type:decimal(5,2)"` // in kilometers
	ETA             *time.Time               `json:"eta"`
	RespondedAt     time.Time                `json:"responded_at"`
	
	// Relationships
	ServiceRequest  CustomerServiceRequest    `json:"service_request,omitempty" gorm:"foreignKey:ServiceRequestID"`
	Worker          WorkerProfile             `json:"worker,omitempty" gorm:"foreignKey:WorkerID"`
}

// WorkerResponseCreate represents the request structure for a worker's response
type WorkerResponseCreate struct {
	Response        string     `json:"response" binding:"required,oneof=accept decline interested"`
	Message         string     `json:"message"`
	ProposedPrice   *float64   `json:"proposed_price"`
	ProposedTime    *time.Time `json:"proposed_time"`
}
