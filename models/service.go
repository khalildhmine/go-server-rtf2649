package models

import (
	"time"

	"gorm.io/gorm"
)

// ServiceCategory represents a service category
type ServiceCategory struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name" gorm:"type:varchar(100);not null;unique"`
	Description string         `json:"description" gorm:"type:text"`
	Icon        string         `json:"icon" gorm:"type:varchar(255)"`
	Color       string         `json:"color" gorm:"type:varchar(20)"`
	IsActive    bool           `json:"is_active" gorm:"default:true"`
	IsNew       bool           `json:"is_new" gorm:"default:false"`
	SortOrder   int            `json:"sort_order" gorm:"default:0"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

// Service represents a service offered by workers
type Service struct {
	ID            uint           `json:"id" gorm:"primaryKey"`
	CategoryID    uint           `json:"category_id" gorm:"not null"`
	Category      ServiceCategory `json:"category" gorm:"foreignKey:CategoryID"`
	Name          string         `json:"name" gorm:"type:varchar(200);not null"`
	Description   string         `json:"description" gorm:"type:text"`
	Price         float64        `json:"price" gorm:"type:decimal(10,2)"`
	ImageURL      string         `json:"image_url" gorm:"type:varchar(255);not null"`
	IsActive      bool           `json:"is_active" gorm:"default:true"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	NameAr        string         `json:"name_ar" gorm:"type:varchar(200);not null"`
	DescriptionAr string         `json:"description_ar" gorm:"type:varchar(500);not null"`
	BasePrice     float64        `json:"base_price" gorm:"type:decimal(10,2)"`
	PriceUnit     string         `json:"price_unit" gorm:"type:varchar(50)"`
	Guarantee     string         `json:"guarantee" gorm:"type:varchar(100)"`
	Policies      string         `json:"policies" gorm:"type:varchar(500)"`
	DeletedAt     gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
	Duration      int            `json:"duration" gorm:"type:int"` // in minutes
}

// ServiceRequest represents the request structure for creating/updating services
type ServiceRequest struct {
	CategoryID  uint    `json:"category_id" binding:"required"`
	Name        string  `json:"name" binding:"required"`
	Description string  `json:"description" binding:"required"`
	Price       float64 `json:"price" binding:"required"`
	Duration    int     `json:"duration" binding:"required"`
}

// ServiceResponse represents the response structure for services
type ServiceResponse struct {
	ID            uint           `json:"id"`
	CategoryID    uint           `json:"category_id"`
	Category      ServiceCategory `json:"category"`
	Name          string         `json:"name"`
	Description   string         `json:"description"`
	Price         float64        `json:"price"`
	ImageURL      string         `json:"image_url"`
	Duration      int            `json:"duration"`
	IsActive      bool           `json:"is_active"`
	CreatedAt     time.Time      `json:"created_at"`
	NameAr        string         `json:"name_ar"`
	DescriptionAr string         `json:"description_ar"`
	BasePrice     float64        `json:"base_price"`
	PriceUnit     string         `json:"price_unit"`
	Guarantee     string         `json:"guarantee"`
	Policies      string         `json:"policies"`
}

// TableName specifies the table name for the Service model
func (Service) TableName() string {
	return "services"
}