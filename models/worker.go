package models

import (
	"time"

	"gorm.io/gorm"
)

// WorkerCategory represents the category of work a worker can perform
type WorkerCategory string

const (
	Plomberie     WorkerCategory = "Plomberie"
	Electricite   WorkerCategory = "Électricité"
	Peinture      WorkerCategory = "Peinture"
	Climatisation WorkerCategory = "Climatisation"
	Menuiserie    WorkerCategory = "Menuiserie"
	Maconnerie    WorkerCategory = "Maçonnerie"
	Nettoyage     WorkerCategory = "Nettoyage"
	Jardinage     WorkerCategory = "Jardinage"
	Serrurerie    WorkerCategory = "Serrurerie"
	Vitrerie      WorkerCategory = "Vitrerie"
)

// WorkerProfile represents a worker's professional profile
type WorkerProfile struct {
	ID              uint           `json:"id" gorm:"primaryKey"`
	UserID          uint           `json:"user_id" gorm:"uniqueIndex;not null"`
	CategoryID      uint           `json:"category_id" gorm:"not null"`
	Category        ServiceCategory `json:"category" gorm:"foreignKey:CategoryID"`
	PhoneNumber     string         `json:"phone_number" gorm:"type:varchar(20);not null"`
	Country         string         `json:"country" gorm:"type:varchar(100);not null"`
	State           string         `json:"state" gorm:"type:varchar(100);not null"`
	City            string         `json:"city" gorm:"type:varchar(100);not null"`
	PostalCode      string         `json:"postal_code" gorm:"type:varchar(20);not null"`
	Address         string         `json:"address" gorm:"type:text"`
	Experience      string         `json:"experience" gorm:"type:text"`
	Skills          string         `json:"skills" gorm:"type:text"`
	HourlyRate      float64        `json:"hourly_rate" gorm:"type:decimal(10,2);default:2500"`
	ProfilePhoto    *string        `json:"profile_photo" gorm:"type:varchar(500)"`
	IDCardPhoto     *string        `json:"id_card_photo" gorm:"type:varchar(500)"`
	IDCardBackPhoto *string        `json:"id_card_photo_back" gorm:"type:varchar(500)"`
	
	// Location and Availability Fields
	IsAvailable     bool           `json:"is_available" gorm:"default:false"`
	CurrentLat      *float64       `json:"current_lat" gorm:"type:decimal(10,8)"`
	CurrentLng      *float64       `json:"current_lng" gorm:"type:decimal(11,8)"`
	LastLocationUpdate *time.Time  `json:"last_location_update"`
	LocationAccuracy *float64      `json:"location_accuracy" gorm:"type:decimal(5,2)"`
	
	// Service Request Fields
	ActiveRequests  int            `json:"active_requests" gorm:"default:0"`
	CompletedJobs   int            `json:"completed_jobs" gorm:"default:0"`
	Rating          float64        `json:"rating" gorm:"type:decimal(3,2);default:0"`
	TotalReviews    int            `json:"total_reviews" gorm:"default:0"`
	IsVerified      bool           `json:"is_verified" gorm:"default:false"`
	
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
	
	// Relationships
	User            User           `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

// WorkerProfileRequest represents the request structure for creating/updating a worker profile
type WorkerProfileRequest struct {
	CategoryID      uint           `json:"category_id" binding:"required"`
	PhoneNumber     string         `json:"phone_number" binding:"required"`
	Country         string         `json:"country" binding:"required"`
	State           string         `json:"state" binding:"required"`
	City            string         `json:"city" binding:"required"`
	PostalCode      string         `json:"postal_code" binding:"required"`
	Address         string         `json:"address"`
	Experience      string         `json:"experience"`
	Skills          string         `json:"skills"`
	HourlyRate      float64        `json:"hourly_rate"`
	ProfilePhoto    *string        `json:"profile_photo"`
	IDCardPhoto     *string        `json:"id_card_photo"`
}

// WorkerProfileResponse represents the response structure for worker profile data
type WorkerProfileResponse struct {
	ID              uint           `json:"id"`
	UserID          uint           `json:"user_id"`
	Category        WorkerCategory `json:"category"`
	PhoneNumber     string         `json:"phone_number"`
	Country         string         `json:"country"`
	City            string         `json:"city"`
	Address         string         `json:"address"`
	Experience      string         `json:"experience"`
	Skills          string         `json:"skills"`
	HourlyRate      float64        `json:"hourly_rate"`
	ProfilePhoto    *string        `json:"profile_photo"`
	IDCardPhoto     *string        `json:"id_card_photo"`
	IsAvailable     bool           `json:"is_available"`
	CurrentLat      *float64       `json:"current_lat"`
	CurrentLng      *float64       `json:"current_lng"`
	LastLocationUpdate *time.Time  `json:"last_location_update"`
	LocationAccuracy *float64      `json:"location_accuracy"`
	ActiveRequests  int            `json:"active_requests"`
	CompletedJobs   int            `json:"completed_jobs"`
	Rating          float64        `json:"rating"`
	TotalReviews    int            `json:"total_reviews"`
	IsVerified      bool           `json:"is_verified"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	User            User           `json:"user,omitempty"`
}

// LocationUpdateRequest represents a worker's location update
type LocationUpdateRequest struct {
	Latitude        float64 `json:"latitude" binding:"required"`
	Longitude       float64 `json:"longitude" binding:"required"`
	Accuracy        float64 `json:"accuracy"`
	IsAvailable     bool    `json:"is_available"`
}

// GetWorkerCategories returns all available worker categories
func GetWorkerCategories() []WorkerCategory {
	return []WorkerCategory{
		Plomberie,
		Electricite,
		Peinture,
		Climatisation,
		Menuiserie,
		Maconnerie,
		Nettoyage,
		Jardinage,
		Serrurerie,
		Vitrerie,
	}
}