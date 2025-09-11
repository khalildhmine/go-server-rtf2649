package models

import (
	"time"

	"gorm.io/gorm"
)

// RefreshToken represents a refresh token for JWT authentication
type RefreshToken struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Token     string    `json:"token" gorm:"size:255;uniqueIndex;not null"`
	UserID    uint      `json:"user_id" gorm:"not null;index"`
	ExpiresAt time.Time `json:"expires_at" gorm:"not null;index"`
	IsRevoked bool      `json:"is_revoked" gorm:"default:false;index"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
	
	// Device information for security
	DeviceID   string `json:"device_id" gorm:"size:255"`
	UserAgent  string `json:"user_agent" gorm:"size:500"`
	IPAddress  string `json:"ip_address" gorm:"size:45"`
	
	// Relationships
	User User `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

// TableName specifies the table name for the RefreshToken model
func (RefreshToken) TableName() string {
	return "refresh_tokens"
}

// IsExpired checks if the refresh token is expired
func (rt *RefreshToken) IsExpired() bool {
	return time.Now().After(rt.ExpiresAt)
}

// IsValid checks if the refresh token is valid (not expired and not revoked)
func (rt *RefreshToken) IsValid() bool {
	return !rt.IsExpired() && !rt.IsRevoked
}

// Revoke marks the refresh token as revoked
func (rt *RefreshToken) Revoke() {
	rt.IsRevoked = true
	rt.UpdatedAt = time.Now()
}

// BeforeCreate is a GORM hook that runs before creating a refresh token
func (rt *RefreshToken) BeforeCreate(tx *gorm.DB) error {
	// Set default expiration to 30 days
	if rt.ExpiresAt.IsZero() {
		rt.ExpiresAt = time.Now().Add(30 * 24 * time.Hour)
	}
	return nil
}
