package models

import (
	"time"
)

type BookingStatus string

const (
	BookingStatusPending   BookingStatus = "pending"
	BookingStatusAccepted  BookingStatus = "accepted"
	BookingStatusInProgress BookingStatus = "in_progress"
	BookingStatusCompleted BookingStatus = "completed"
	BookingStatusCancelled BookingStatus = "cancelled"
)

type Booking struct {
	ID          uint          `json:"id" gorm:"primaryKey"`
	UserID      uint          `json:"user_id" gorm:"not null"`
	ServiceID   uint          `json:"service_id" gorm:"not null"`
	WorkerID    *uint         `json:"worker_id"` // Can be null initially
	Status      BookingStatus `json:"status" gorm:"type:varchar(20);default:'pending';check:status IN ('pending','accepted','in_progress','completed','cancelled')"`
	Address     string        `json:"address" gorm:"size:500;not null"`
	Date        time.Time     `json:"date" gorm:"not null"`
	Time        string        `json:"time" gorm:"size:20;not null"`
	Notes       *string       `json:"notes" gorm:"size:1000"`
	TotalPrice  float64       `json:"total_price" gorm:"type:decimal(10,2);not null"`
	CreatedAt   time.Time     `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time     `json:"updated_at" gorm:"autoUpdateTime"`

	// Relationships
	User    User         `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Service Service      `json:"service,omitempty" gorm:"foreignKey:ServiceID"`
	Worker  *WorkerProfile `json:"worker,omitempty" gorm:"foreignKey:WorkerID"`
}

// TableName specifies the table name for the Booking model
func (Booking) TableName() string {
	return "bookings"
}