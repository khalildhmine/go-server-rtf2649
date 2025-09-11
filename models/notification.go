package models

import (
	"time"

	"gorm.io/gorm"
)

type Notification struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	UserID    uint           `json:"user_id" gorm:"not null"`
	Title     string         `json:"title" gorm:"not null"`
	Body      string         `json:"body" gorm:"not null"`
	Type      string         `json:"type" gorm:"not null"` // booking_created, booking_accepted, booking_in_progress, booking_completed, booking_cancelled, worker_assigned, payment_received, promotion, system
	Data      string         `json:"data" gorm:"type:text"` // JSON data
	Read      bool           `json:"read" gorm:"default:false"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	// Relations
	User User `json:"user" gorm:"foreignKey:UserID"`
}

type PushToken struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	UserID    uint           `json:"user_id" gorm:"not null"`
	Token     string         `json:"token" gorm:"not null;unique"`
	Platform  string         `json:"platform" gorm:"not null"` // ios, android
	DeviceID  string         `json:"device_id"`
	Active    bool           `json:"active" gorm:"default:true"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	// Relations
	User User `json:"user" gorm:"foreignKey:UserID"`
}