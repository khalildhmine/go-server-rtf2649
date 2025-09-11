package routes

import (
	"context"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"repair-service-server/database"
	"repair-service-server/middleware"
	"repair-service-server/models"
	ws "repair-service-server/websocket"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

var chatHub *ws.Hub

// InitChatHub initializes the WebSocket hub for chat
func InitChatHub() {
	// This function is now handled in main.go with globalChatHub
	log.Println("🚀 Chat WebSocket hub initialization delegated to main.go")
}

// GetChatHub returns the chat hub instance
func GetChatHub() *ws.Hub {
	// Import the global hub from main package
	// For now, return the local chatHub variable
	return chatHub
}

// ChatRoutes sets up chat-related routes
func ChatRoutes(router *gin.Engine, hub *ws.Hub) {
	// Set the local chatHub variable to use the passed hub
	chatHub = hub
	
	chat := router.Group("/api/v1/chat")
	{
		// WebSocket connection - use WebSocket-specific auth middleware
		chat.GET("/ws", middleware.WebSocketAuthMiddleware(), handleWebSocketConnection)
		
		// Chat room management
		chat.GET("/rooms", middleware.AuthMiddleware(), getChatRooms)
		chat.POST("/rooms", middleware.AuthMiddleware(), createChatRoom)
		chat.POST("/rooms/get-or-create", middleware.AuthMiddleware(), getOrCreateChatRoom)
		chat.GET("/rooms/:id", middleware.AuthMiddleware(), getChatRoom)
		
		// Message management
		chat.GET("/rooms/:id/messages", middleware.AuthMiddleware(), getChatMessages)
		chat.POST("/rooms/:id/messages", middleware.AuthMiddleware(), sendMessage)
		chat.POST("/rooms/:id/mark-read", middleware.AuthMiddleware(), markMessagesAsReadEndpoint)
		chat.PUT("/messages/:id/read", middleware.AuthMiddleware(), markMessageAsRead)
		
		// Voice message management
		chat.POST("/rooms/:id/voice-messages", middleware.AuthMiddleware(), uploadVoiceMessage)
		
		// Device token management for push notifications
		chat.POST("/device-token", middleware.AuthMiddleware(), registerDeviceToken)
		chat.DELETE("/device-token", middleware.AuthMiddleware(), unregisterDeviceToken)
	}
}

// handleWebSocketConnection handles WebSocket connection and adds user to their chat rooms
func handleWebSocketConnection(c *gin.Context) {
	userID := c.GetUint("user_id")
	userType := c.Query("user_type") // Get user_type from query parameters
	
	if userType == "" {
		// Determine user type based on whether they have a worker profile
		var workerProfile models.WorkerProfile
		if err := database.DB.Where("user_id = ?", userID).First(&workerProfile).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				userType = "customer"
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to determine user type"})
				return
			}
		} else {
			userType = "worker"
		}
	}
	
	log.Printf("🔌 WebSocket connection: UserID=%d, UserType=%s", userID, userType)
	
	// Add user to their existing chat rooms for real-time messaging
	var chatRooms []models.ChatRoom
	if err := database.DB.Where("customer_id = ? OR worker_id = ?", userID, userID).Find(&chatRooms).Error; err == nil {
		for _, room := range chatRooms {
			chatHub.AddUserToChatRoom(userID, room.ID)
			log.Printf("👥 User %d added to existing chat room %d", userID, room.ID)
		}
	}
	
	// Upgrade HTTP connection to WebSocket
	ws.ServeWebSocket(chatHub, c.Writer, c.Request, userID, userType)
}

// getChatRooms returns all chat rooms for the authenticated user
func getChatRooms(c *gin.Context) {
	userID := c.GetUint("user_id")
	
	var chatRooms []models.ChatRoom
	
	// Get chat rooms where user is either customer or worker
	if err := database.DB.
		Preload("Customer").
		Preload("Worker").
		Preload("ServiceRequest").
		Where("customer_id = ? OR worker_id = ?", userID, userID).
		Order("last_message_at DESC NULLS LAST, created_at DESC").
		Find(&chatRooms).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch chat rooms"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"chat_rooms": chatRooms,
	})
}

// createChatRoom creates a new chat room between customer and worker
func createChatRoom(c *gin.Context) {
	userID := c.GetUint("user_id")
	
	var request struct {
		WorkerID         uint `json:"worker_id" binding:"required"`
		ServiceRequestID uint `json:"service_request_id" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}
	
	// Verify the service request exists and belongs to the customer
	var serviceRequest models.CustomerServiceRequest
	if err := database.DB.Where("id = ? AND customer_id = ?", request.ServiceRequestID, userID).First(&serviceRequest).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service request not found"})
		return
	}
	
	// Check if chat room already exists
	var existingRoom models.ChatRoom
	if err := database.DB.Where("customer_id = ? AND worker_id = ? AND service_request_id = ?", 
		userID, request.WorkerID, request.ServiceRequestID).First(&existingRoom).Error; err == nil {
		// Room already exists, return it
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"chat_room": existingRoom,
			"message": "Chat room already exists",
		})
		return
	}
	
	// Create new chat room
	chatRoom := models.ChatRoom{
		CustomerID:       userID,
		WorkerID:         request.WorkerID,
		ServiceRequestID: request.ServiceRequestID,
		IsActive:         true,
	}
	
	if err := database.DB.Create(&chatRoom).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create chat room"})
		return
	}
	
	// Load the created room with relationships
	database.DB.
		Preload("Customer", "id, full_name, profile_picture_url").
		Preload("Worker", "id, full_name, profile_picture_url").
		Preload("ServiceRequest", "id, title, status").
		First(&chatRoom, chatRoom.ID)
	
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"chat_room": chatRoom,
	})
}

// getChatRoom returns a specific chat room with messages
func getChatRoom(c *gin.Context) {
	userID := c.GetUint("user_id")
	chatRoomID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chat room ID"})
		return
	}
	
	var chatRoom models.ChatRoom
	if err := database.DB.
		Preload("Customer").
		Preload("Worker").
		Preload("ServiceRequest").
		Where("id = ? AND (customer_id = ? OR worker_id = ?)", chatRoomID, userID, userID).
		First(&chatRoom).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chat room not found"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"chat_room": chatRoom,
	})
}

// getChatMessages returns messages for a specific chat room
func getChatMessages(c *gin.Context) {
	userID := c.GetUint("user_id")
	chatRoomID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chat room ID"})
		return
	}
	
	// Verify user has access to this chat room
	var chatRoom models.ChatRoom
	if err := database.DB.Where("id = ? AND (customer_id = ? OR worker_id = ?)", 
		chatRoomID, userID, userID).First(&chatRoom).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chat room not found"})
		return
	}
	
	// Get pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset := (page - 1) * limit
	
	var messages []models.ChatMessage
	var total int64
	
	// Get total count
	database.DB.Model(&models.ChatMessage{}).Where("chat_room_id = ?", chatRoomID).Count(&total)
	
	// Get messages with pagination
	if err := database.DB.
		Where("chat_room_id = ?", chatRoomID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&messages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch messages"})
		return
	}
	
	// Mark messages as read for the other user
	go markMessagesAsRead(uint(chatRoomID), userID)
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"messages": messages,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
		},
	})
}

// sendMessage sends a new message in a chat room
func sendMessage(c *gin.Context) {
	userID := c.GetUint("user_id")
	chatRoomID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chat room ID"})
		return
	}
	
	var request struct {
		MessageText string `json:"message_text" binding:"required"`
		MessageType string `json:"message_type" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}
	
	// Verify user has access to this chat room
	var chatRoom models.ChatRoom
	if err := database.DB.Where("id = ? AND (customer_id = ? OR worker_id = ?)", 
		chatRoomID, userID, userID).First(&chatRoom).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chat room not found"})
		return
	}
	
	// Determine sender type
	var senderType string
	if chatRoom.CustomerID == userID {
		senderType = "customer"
	} else {
		senderType = "worker"
	}
	
	// Create the message
	message := models.ChatMessage{
		ChatRoomID:  uint(chatRoomID),
		SenderID:    userID,
		SenderType:  senderType,
		Content:     request.MessageText,
		MessageText: request.MessageText, // Also set MessageText to match Content
		MessageType: request.MessageType,
		IsRead:      false,
	}
	
	if err := database.DB.Create(&message).Error; err != nil {
		log.Printf("❌ Database error creating chat message: %v", err)
		log.Printf("🔍 Message data: ChatRoomID=%d, SenderID=%d, SenderType=%s, Content='%s', MessageText='%s'", 
			message.ChatRoomID, message.SenderID, message.SenderType, message.Content, message.MessageText)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send message"})
		return
	}
	
	// Update chat room last message info
	now := time.Now()
	database.DB.Model(&chatRoom).Updates(map[string]interface{}{
		"last_message_at":   &now,
		"last_message_text": request.MessageText,
		"unread_count":      gorm.Expr("unread_count + 1"),
	})
	
	// Send real-time message via WebSocket
	websocketMessage := &ws.Message{
		Type:        "chat",
		ChatRoomID:  uint(chatRoomID),
		SenderID:    userID,
		SenderType:  senderType,
		Content:     request.MessageText,
		Timestamp:   now,
	}
	
	// Ensure sender is in the chat room for WebSocket
	chatHub.AddUserToChatRoom(userID, uint(chatRoomID))
	
	// Send to all users in the chat room (excluding sender)
	chatHub.SendToChatRoom(uint(chatRoomID), websocketMessage, userID)
	
	// Send push notifications to offline users
	go sendPushNotifications(uint(chatRoomID), userID, request.MessageText)
	
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": message,
	})
}

// markMessageAsRead marks a specific message as read
func markMessageAsRead(c *gin.Context) {
	userID := c.GetUint("user_id")
	messageID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message ID"})
		return
	}
	
	var message models.ChatMessage
	if err := database.DB.Where("id = ?", messageID).First(&message).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Message not found"})
		return
	}
	
	// Verify user has access to this message's chat room
	var chatRoom models.ChatRoom
	if err := database.DB.Where("id = ? AND (customer_id = ? OR worker_id = ?)", 
		message.ChatRoomID, userID, userID).First(&chatRoom).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}
	
	// Mark message as read
	now := time.Now()
	database.DB.Model(&message).Updates(map[string]interface{}{
		"is_read": &now,
		"read_at": &now,
	})
	
	// Send read receipt via WebSocket
	readReceipt := &ws.Message{
		Type:       "read_receipt",
		ChatRoomID: message.ChatRoomID,
		Data: map[string]interface{}{
			"message_id": messageID,
			"read_at":    now,
		},
		Timestamp: now,
	}
	
	chatHub.SendToChatRoom(message.ChatRoomID, readReceipt, userID)
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Message marked as read",
	})
}

// registerDeviceToken registers a device token for push notifications
func registerDeviceToken(c *gin.Context) {
	userID := c.GetUint("user_id")
	
	var request struct {
		DeviceToken string `json:"device_token" binding:"required"`
		Platform   string `json:"platform" binding:"required"` // "android", "ios", "web"
		DeviceInfo string `json:"device_info"`
	}
	
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}
	
	// Validate platform
	if request.Platform != "android" && request.Platform != "ios" && request.Platform != "web" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid platform"})
		return
	}
	
	// Upsert device token
	var deviceToken models.UserDeviceToken
	result := database.DB.Where("user_id = ? AND platform = ?", userID, request.Platform).
		FirstOrCreate(&deviceToken, models.UserDeviceToken{
			UserID:      userID,
			Platform:    request.Platform,
			DeviceToken: request.DeviceToken,
			DeviceInfo:  request.DeviceInfo,
			IsActive:    true,
			LastUsedAt:  time.Now(),
		})
	
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register device token"})
		return
	}
	
	// Update existing token if found
	if result.RowsAffected == 0 {
		database.DB.Model(&deviceToken).Updates(map[string]interface{}{
			"device_token": request.DeviceToken,
			"device_info":  request.DeviceInfo,
			"is_active":    true,
			"last_used_at": time.Now(),
		})
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Device token registered successfully",
	})
}

// unregisterDeviceToken removes a device token
func unregisterDeviceToken(c *gin.Context) {
	userID := c.GetUint("user_id")
	
	var request struct {
		Platform string `json:"platform" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}
	
	// Soft delete device token
	if err := database.DB.Where("user_id = ? AND platform = ?", userID, request.Platform).
		Delete(&models.UserDeviceToken{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unregister device token"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Device token unregistered successfully",
	})
}

// getOrCreateChatRoom gets an existing chat room or creates a new one between customer and worker
func getOrCreateChatRoom(c *gin.Context) {
	userID := c.GetUint("user_id")
	
	// Accept both numeric and string IDs for robustness
	var raw map[string]interface{}
	if err := c.ShouldBindJSON(&raw); err != nil {
		log.Printf("🔍 Invalid request data (bind): %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}
	
	parseUint := func(v interface{}) (uint, bool) {
		switch t := v.(type) {
		case float64:
			if t < 0 {
				return 0, false
			}
			return uint(t), true
		case string:
			if t == "" {
				return 0, false
			}
			if n, err := strconv.ParseUint(t, 10, 64); err == nil {
				return uint(n), true
			}
			return 0, false
		default:
			return 0, false
		}
	}
	
	customerID, ok1 := parseUint(raw["customer_id"])
	workerID, ok2 := parseUint(raw["worker_id"])
	serviceRequestID, ok3 := parseUint(raw["service_request_id"])
	if !ok1 || !ok2 || !ok3 || customerID == 0 || workerID == 0 || serviceRequestID == 0 {
		log.Printf("🔍 Invalid request values: raw=%v", raw)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}
	
	log.Printf("🔍 getOrCreateChatRoom request: userID=%d, customerID=%d, workerID=%d, serviceRequestID=%d", userID, customerID, workerID, serviceRequestID)
	
	// Verify the user is either the customer or worker
	if userID != customerID && userID != workerID {
		log.Printf("🔍 Access denied: userID=%d, customerID=%d, workerID=%d", userID, customerID, workerID)
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}
	
	// Check if chat room already exists
	var existingRoom models.ChatRoom
	if err := database.DB.
		Preload("Customer").
		Preload("Worker").
		Preload("ServiceRequest").
		Where("customer_id = ? AND worker_id = ? AND service_request_id = ?", customerID, workerID, serviceRequestID).
		First(&existingRoom).Error; err == nil {
		// Room already exists, return it
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"chat_room": existingRoom,
		})
		return
	}
	
	// Verify the service request exists
	var serviceRequest models.CustomerServiceRequest
	if err := database.DB.Where("id = ?", serviceRequestID).First(&serviceRequest).Error; err != nil {
		log.Printf("🔍 Service request not found: ID=%d, error=%v", serviceRequestID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Service request not found"})
		return
	}
	
	log.Printf("🔍 Service request found: ID=%d, status=%s", serviceRequest.ID, serviceRequest.Status)
	
	// Verify customer and worker exist
	var customer models.User
	if err := database.DB.Where("id = ?", customerID).First(&customer).Error; err != nil {
		log.Printf("🔍 Customer not found: ID=%d, error=%v", customerID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
		return
	}
	
	log.Printf("🔍 Customer found: ID=%d, name=%s", customer.ID, customer.FullName)
	
	var worker models.User
	if err := database.DB.Where("id = ?", workerID).First(&worker).Error; err != nil {
		log.Printf("🔍 Worker not found: ID=%d, error=%v", workerID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Worker not found"})
		return
	}
	
	log.Printf("🔍 Worker found: ID=%d, name=%s", worker.ID, worker.FullName)
	
	// Create new chat room
	chatRoom := models.ChatRoom{
		CustomerID:        customerID,
		WorkerID:          workerID,
		ServiceRequestID:  serviceRequestID,
		IsActive:          true,
		UnreadCount:       0,
	}
	
	if err := database.DB.Create(&chatRoom).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create chat room"})
		return
	}
	
	// Load the created chat room with relationships
	if err := database.DB.
		Preload("Customer").
		Preload("Worker").
		Preload("ServiceRequest").
		Where("id = ?", chatRoom.ID).
		First(&chatRoom).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load created chat room"})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"chat_room": chatRoom,
	})
}

// markMessagesAsReadEndpoint marks all messages in a chat room as read for the authenticated user
func markMessagesAsReadEndpoint(c *gin.Context) {
	userID := c.GetUint("user_id")
	chatRoomID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chat room ID"})
		return
	}
	
	// Verify user has access to this chat room
	var chatRoom models.ChatRoom
	if err := database.DB.Where("id = ? AND (customer_id = ? OR worker_id = ?)", 
		chatRoomID, userID, userID).First(&chatRoom).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chat room not found"})
		return
	}
	
	// Mark messages as read
	markMessagesAsRead(uint(chatRoomID), userID)
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Messages marked as read",
	})
}

// Helper functions

// markMessagesAsRead marks all unread messages in a chat room as read for a specific user
func markMessagesAsRead(chatRoomID uint, userID uint) {
	// Get the other user's ID from the chat room
	var chatRoom models.ChatRoom
	if err := database.DB.Where("id = ?", chatRoomID).First(&chatRoom).Error; err != nil {
		return
	}
	
	var otherUserID uint
	if chatRoom.CustomerID == userID {
		otherUserID = chatRoom.WorkerID
	} else {
		otherUserID = chatRoom.CustomerID
	}
	
	// Mark messages from the other user as read
	now := time.Now()
	database.DB.Model(&models.ChatMessage{}).
		Where("chat_room_id = ? AND sender_id = ? AND is_read = ?", 
			chatRoomID, otherUserID, false).
		Updates(map[string]interface{}{
			"is_read": &now,
			"read_at": &now,
		})
	
	// Reset unread count
	database.DB.Model(&chatRoom).Update("unread_count", 0)
}

// sendPushNotifications sends push notifications to offline users
func sendPushNotifications(chatRoomID uint, senderID uint, messageContent string) {
	// This will be implemented with Firebase/Expo push notification services
	// For now, just log the action
	log.Printf("📱 Push notification would be sent for chat room %d, message: %s", chatRoomID, messageContent)
}

// uploadVoiceMessage handles voice message uploads
func uploadVoiceMessage(c *gin.Context) {
	userID := c.GetUint("user_id")
	chatRoomID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chat room ID"})
		return
	}

	// Verify user has access to this chat room
	var chatRoom models.ChatRoom
	if err := database.DB.Where("id = ? AND (customer_id = ? OR worker_id = ?)", 
		chatRoomID, userID, userID).First(&chatRoom).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chat room not found"})
		return
	}

	// Parse multipart form
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil { // 32MB max
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse form"})
		return
	}

	// Get audio file
	file, header, err := c.Request.FormFile("audio")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No audio file provided"})
		return
	}
	defer file.Close()

	// Validate file type
	if !strings.HasSuffix(header.Filename, ".m4a") && !strings.HasSuffix(header.Filename, ".mp3") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only .m4a and .mp3 files are supported"})
		return
	}

	// Validate file size (max 10MB)
	if header.Size > 10<<20 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File size too large. Maximum 10MB allowed"})
		return
	}

	// Get duration from form
	durationStr := c.Request.FormValue("duration")
	duration, err := strconv.Atoi(durationStr)
	if err != nil || duration <= 0 || duration > 600 { // Max 10 minutes
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid duration. Must be between 1-600 seconds"})
		return
	}

	// Upload to Cloudinary
	audioURL, err := uploadToCloudinary(file, header.Filename)
	if err != nil {
		log.Printf("❌ Cloudinary upload failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload audio file"})
		return
	}

	// Determine sender type
	var senderType string
	if chatRoom.CustomerID == userID {
		senderType = "customer"
	} else {
		senderType = "worker"
	}

	// Create the voice message
	message := models.ChatMessage{
		ChatRoomID:  uint(chatRoomID),
		SenderID:    userID,
		SenderType:  senderType,
		Content:     "🎤 Voice message",
		MessageText: "🎤 Voice message",
		MessageType: "voice",
		AudioURL:    audioURL,
		Duration:    duration,
		IsRead:      false,
	}

	if err := database.DB.Create(&message).Error; err != nil {
		log.Printf("❌ Database error creating voice message: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save voice message"})
		return
	}

	// Update chat room last message info
	now := time.Now()
	database.DB.Model(&chatRoom).Updates(map[string]interface{}{
		"last_message_at":   &now,
		"last_message_text": "🎤 Voice message",
		"unread_count":      gorm.Expr("unread_count + ?", 1),
	})

	// Broadcast to WebSocket
	websocketMessage := &ws.Message{
		Type:        "voice_message",
		ChatRoomID:  uint(chatRoomID),
		SenderID:    userID,
		SenderType:  senderType,
		Content:     "🎤 Voice message",
		Timestamp:   now,
		Data: gin.H{
			"message": message,
			"chat_room_id": chatRoomID,
		},
	}
	
	// Add user to chat room and send message
	chatHub.AddUserToChatRoom(userID, uint(chatRoomID))
	chatHub.SendToChatRoom(uint(chatRoomID), websocketMessage, userID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Voice message sent successfully",
		"data": gin.H{
			"message": message,
		},
	})
}

// uploadToCloudinary uploads audio file to Cloudinary
func uploadToCloudinary(file multipart.File, filename string) (string, error) {
	// Configure Cloudinary
	cld, err := cloudinary.New()
	if err != nil {
		return "", err
	}

	// Upload file with basic parameters
	result, err := cld.Upload.Upload(context.Background(), file, uploader.UploadParams{
		ResourceType: "video", // Use video for audio files
		PublicID:     fmt.Sprintf("voice_messages/%s_%d", filename, time.Now().Unix()),
		Format:       "mp3", // Convert to MP3 for better compatibility
		Transformation: "f_mp3", // Force MP3 format
	})
	if err != nil {
		return "", err
	}

	return result.SecureURL, nil
}
