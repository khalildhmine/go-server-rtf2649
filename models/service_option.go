package models

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// ServiceOption represents a specific service option within a category
type ServiceOption struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	CategoryID  uint           `json:"category_id" gorm:"not null"`
	Title       string         `json:"title" gorm:"not null"`
	Description string         `json:"description" gorm:"not null"`
	ImageURL    string         `json:"image_url"`
	Price       float64        `json:"price" gorm:"not null"`
	Duration    int            `json:"duration" gorm:"not null"` // in minutes
	Features    []string       `json:"features" gorm:"-"`        // Will be stored as JSON
	FeaturesJSON string        `json:"-" gorm:"column:features;type:json"`
	IsActive    bool           `json:"is_active" gorm:"default:true"`
	SortOrder   int            `json:"sort_order" gorm:"default:0"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`

	// Relationships
	Category ServiceCategory `json:"category" gorm:"foreignKey:CategoryID"`
}

// TableName specifies the table name for ServiceOption
func (ServiceOption) TableName() string {
	return "service_options"
}

// BeforeSave hook to convert features slice to JSON
func (so *ServiceOption) BeforeSave(tx *gorm.DB) error {
	if len(so.Features) > 0 {
		featuresJSON, err := json.Marshal(so.Features)
		if err != nil {
			return err
		}
		so.FeaturesJSON = string(featuresJSON)
	}
	return nil
}

// AfterFind hook to convert JSON back to features slice
func (so *ServiceOption) AfterFind(tx *gorm.DB) error {
	if so.FeaturesJSON != "" {
		return json.Unmarshal([]byte(so.FeaturesJSON), &so.Features)
	}
	return nil
}
