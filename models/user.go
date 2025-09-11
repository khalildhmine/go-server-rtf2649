package models

import (
	"time"

	"gorm.io/gorm"
)

type UserRole string

const (
	RoleCustomer UserRole = "customer"
	RoleWorker   UserRole = "worker"
	RoleAdmin    UserRole = "admin"
)

type User struct {
	ID               uint      `json:"id" gorm:"primaryKey"`
	FullName         string    `json:"full_name" gorm:"size:255;not null"`
	PhoneNumber      string    `json:"phone_number" gorm:"size:20;uniqueIndex;not null"`
	PasswordHash     string    `json:"-" gorm:"size:255;not null"` // Hidden from JSON
	Role             UserRole  `json:"role" gorm:"type:varchar(20);not null;default:'customer';check:role IN ('customer','worker','admin')"`
	ProfilePictureURL *string  `json:"profile_picture_url" gorm:"size:255"`
	IsActive         bool      `json:"is_active" gorm:"default:true"`
	CreatedAt        time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt        time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	// Relationships
	Bookings []Booking `json:"bookings,omitempty" gorm:"foreignKey:UserID"`
	Addresses []Address `json:"addresses,omitempty" gorm:"foreignKey:UserID"`
}

// TableName specifies the table name for the User model
func (User) TableName() string {
	return "users"
}

// BeforeCreate is a GORM hook that runs before creating a user
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.Role == "" {
		u.Role = RoleCustomer
	}
	return nil
}

// IsValidRole checks if the user role is valid
func (u *User) IsValidRole() bool {
	switch u.Role {
	case RoleCustomer, RoleWorker, RoleAdmin:
		return true
	default:
		return false
	}
}

// IsWorker checks if the user is a worker
func (u *User) IsWorker() bool {
	return u.Role == RoleWorker
}

// IsAdmin checks if the user is an admin
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// IsCustomer checks if the user is a customer
func (u *User) IsCustomer() bool {
	return u.Role == RoleCustomer
}