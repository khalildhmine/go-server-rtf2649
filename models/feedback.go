package models

import (
	"time"

	"gorm.io/gorm"
)

// Feedback represents a worker experience feedback entry
type Feedback struct {
    ID              uint           `json:"id" gorm:"primaryKey"`
    UserID          uint           `json:"user_id" gorm:"not null"`
    WorkerID        *uint          `json:"worker_id"` // optional linkage to worker profile
    ServiceRequestID *uint         `json:"service_request_id"`
    Rating          int            `json:"rating" gorm:"type:int;check:rating >= 1 AND rating <= 5"`
    Comment         string         `json:"comment" gorm:"type:text"`
    AppVersion      string         `json:"app_version" gorm:"type:varchar(50)"`
    DeviceModel     string         `json:"device_model" gorm:"type:varchar(100)"`
    OS              string         `json:"os" gorm:"type:varchar(50)"`
    CreatedAt       time.Time      `json:"created_at"`
    UpdatedAt       time.Time      `json:"updated_at"`
    DeletedAt       gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

// TableName sets custom table name
func (Feedback) TableName() string { return "feedback" }


