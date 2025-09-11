package models

import (
	"time"

	"gorm.io/gorm"
)

// ChatRoom represents a chat conversation between a customer and worker
type ChatRoom struct {
	ID                uint      `json:"id" gorm:"primaryKey"`
	CustomerID        uint      `json:"customer_id" gorm:"not null"`
	WorkerID          uint      `json:"worker_id" gorm:"not null"`
	ServiceRequestID  uint      `json:"service_request_id" gorm:"not null"`
	Customer          User      `json:"customer" gorm:"foreignKey:CustomerID"`
	Worker            User      `json:"worker" gorm:"foreignKey:WorkerID"`
	ServiceRequest    CustomerServiceRequest `json:"service_request" gorm:"foreignKey:ServiceRequestID"`
	LastMessageAt     *time.Time `json:"last_message_at"`
	LastMessageText   string    `json:"last_message_text"`
	UnreadCount       int       `json:"unread_count" gorm:"default:0"`
	IsActive          bool      `json:"is_active" gorm:"default:true"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	DeletedAt         *time.Time `json:"deleted_at,omitempty" gorm:"index"`
}

// ChatMessage represents a single message in a chat room
type ChatMessage struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	ChatRoomID uint      `json:"chat_room_id" gorm:"not null"`
	SenderID   uint      `json:"sender_id" gorm:"not null"`
	SenderType string    `json:"sender_type" gorm:"not null"` // "customer" or "worker"
	Content    string    `json:"content" gorm:"type:text;not null"`
	MessageText string   `json:"message_text" gorm:"type:text;not null"` // Alias for content
	MessageType string   `json:"message_type" gorm:"default:text"` // "text", "image", "file", "voice"
	AudioURL   string    `json:"audio_url"` // URL for voice messages
	Duration   int       `json:"duration"` // Duration in seconds for voice messages
	IsRead     bool      `json:"is_read" gorm:"default:false"`
	ReadAt     *time.Time `json:"read_at"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty" gorm:"index"`
}

// ChatNotification represents push notifications for chat messages
type ChatNotification struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	UserID       uint      `json:"user_id" gorm:"not null"`
	ChatRoomID   uint      `json:"chat_room_id" gorm:"not null"`
	MessageID    uint      `json:"message_id" gorm:"not null"`
	Title        string    `json:"title" gorm:"not null"`
	Body         string    `json:"body" gorm:"not null"`
	Type         string    `json:"type" gorm:"default:chat"` // "chat", "service_update", etc.
	IsRead       bool      `json:"is_read" gorm:"default:false"`
	ReadAt       *time.Time `json:"read_at"`
	DeviceToken  string    `json:"device_token"` // Firebase/Expo device token
	Platform     string    `json:"platform"` // "android", "ios", "web"
	SentAt       *time.Time `json:"sent_at"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty" gorm:"index"`
}

// UserDeviceToken stores device tokens for push notifications
type UserDeviceToken struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	UserID     uint      `json:"user_id" gorm:"not null;uniqueIndex:idx_user_platform"`
	Platform   string    `json:"platform" gorm:"not null;uniqueIndex:idx_user_platform"` // "android", "ios", "web"
	DeviceToken string   `json:"device_token" gorm:"not null"`
	DeviceInfo string    `json:"device_info"` // Device model, OS version, etc.
	IsActive   bool      `json:"is_active" gorm:"default:true"`
	LastUsedAt time.Time `json:"last_used_at"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty" gorm:"index"`
}

// TableName specifies the table name for ChatRoom
func (ChatRoom) TableName() string {
	return "chat_rooms"
}

// TableName specifies the table name for ChatMessage
func (ChatMessage) TableName() string {
	return "chat_messages"
}

// TableName specifies the table name for ChatNotification
func (ChatNotification) TableName() string {
	return "chat_notifications"
}

// TableName specifies the table name for UserDeviceToken
func (UserDeviceToken) TableName() string {
	return "user_device_tokens"
}

// BeforeSave hook to sync Content and MessageText fields
func (m *ChatMessage) BeforeSave(tx *gorm.DB) error {
	if m.Content != "" && m.MessageText == "" {
		m.MessageText = m.Content
	} else if m.MessageText != "" && m.Content == "" {
		m.Content = m.MessageText
	}
	return nil
}

// AfterFind hook to sync Content and MessageText fields
func (m *ChatMessage) AfterFind(tx *gorm.DB) error {
	if m.Content != "" && m.MessageText == "" {
		m.MessageText = m.Content
	} else if m.MessageText != "" && m.Content == "" {
		m.Content = m.MessageText
	}
	return nil
}
