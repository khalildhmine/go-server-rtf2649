package websocket

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Client represents a connected WebSocket client
type Client struct {
	Hub      *Hub
	ID       uint
	UserType string // "customer" or "worker"
	Conn     *websocket.Conn
	Send     chan []byte
	mu       sync.Mutex
}

// Hub manages all WebSocket connections
type Hub struct {
	// Registered clients
	Clients map[uint]*Client

	// Chat room members
	ChatRoomMembers map[uint]map[uint]bool

	// Broadcast channel for messages to all clients
	Broadcast chan *Message

	// Register requests from clients
	Register chan *Client

	// Unregister requests from clients
	Unregister chan *Client

	// Message handlers
	MessageHandlers map[string]MessageHandler

	mu sync.RWMutex
}

// Message represents a chat message
type Message struct {
	Type      string      `json:"type"`
	ChatRoomID uint       `json:"chat_room_id,omitempty"`
	SenderID  uint        `json:"sender_id,omitempty"`
	SenderType string     `json:"sender_type,omitempty"`
	Content   string      `json:"content,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data,omitempty"`
}

// MessageHandler handles different types of messages
type MessageHandler func(*Client, *Message) error

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	hub := &Hub{
		Clients:         make(map[uint]*Client),
		ChatRoomMembers: make(map[uint]map[uint]bool),
		Broadcast:       make(chan *Message),
		Register:        make(chan *Client),
		Unregister:      make(chan *Client),
		MessageHandlers: make(map[string]MessageHandler),
	}

	// Register default message handlers
	hub.registerDefaultHandlers()

	return hub
}

// registerDefaultHandlers registers default message handlers
func (h *Hub) registerDefaultHandlers() {
	h.MessageHandlers["chat"] = h.handleChatMessage
	h.MessageHandlers["typing"] = h.handleTypingIndicator
	h.MessageHandlers["read"] = h.handleReadReceipt
	h.MessageHandlers["ping"] = h.handlePing
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.Clients[client.ID] = client
			h.mu.Unlock()
			log.Printf("🔌 Client registered: ID=%d, Type=%s", client.ID, client.UserType)

		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.Clients[client.ID]; ok {
				// Remove user from all chat rooms
				for chatRoomID := range h.ChatRoomMembers {
					if h.ChatRoomMembers[chatRoomID][client.ID] {
						delete(h.ChatRoomMembers[chatRoomID], client.ID)
						log.Printf("👥 User %d removed from chat room %d on disconnect", client.ID, chatRoomID)
					}
				}
				
				delete(h.Clients, client.ID)
				close(client.Send)
			}
			h.mu.Unlock()
			log.Printf("🔌 Client unregistered: ID=%d, Type=%s", client.ID, client.UserType)

		case message := <-h.Broadcast:
			h.broadcastMessage(message)
		}
	}
}

// broadcastMessage sends a message to all connected clients
func (h *Hub) broadcastMessage(message *Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("❌ Error marshaling message: %v", err)
		return
	}

	for _, client := range h.Clients {
		select {
		case client.Send <- data:
		default:
			close(client.Send)
			delete(h.Clients, client.ID)
		}
	}
}

// SendToUser sends a message to a specific user
func (h *Hub) SendToUser(userID uint, message *Message) {
	h.mu.RLock()
	client, exists := h.Clients[userID]
	h.mu.RUnlock()

	if !exists {
		log.Printf("⚠️ User %d not connected, message will be sent via push notification", userID)
		return
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("❌ Error marshaling message: %v", err)
		return
	}

	select {
	case client.Send <- data:
		log.Printf("✅ Message sent to user %d", userID)
	default:
		log.Printf("⚠️ User %d's send buffer is full", userID)
	}
}

// AddUserToChatRoom adds a user to a specific chat room
func (h *Hub) AddUserToChatRoom(userID uint, chatRoomID uint) {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	if h.ChatRoomMembers[chatRoomID] == nil {
		h.ChatRoomMembers[chatRoomID] = make(map[uint]bool)
	}
	h.ChatRoomMembers[chatRoomID][userID] = true
	
	log.Printf("👥 User %d added to chat room %d", userID, chatRoomID)
}

// RemoveUserFromChatRoom removes a user from a specific chat room
func (h *Hub) RemoveUserFromChatRoom(userID uint, chatRoomID uint) {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	if h.ChatRoomMembers[chatRoomID] != nil {
		delete(h.ChatRoomMembers[chatRoomID], userID)
		log.Printf("👥 User %d removed from chat room %d", userID, chatRoomID)
	}
}

// SendToChatRoom sends a message to all users in a specific chat room
func (h *Hub) SendToChatRoom(chatRoomID uint, message *Message, excludeUserID uint) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("❌ Error marshaling message: %v", err)
		return
	}

	// Get users in this chat room
	roomMembers := h.ChatRoomMembers[chatRoomID]
	if roomMembers == nil {
		log.Printf("⚠️ No users found in chat room %d", chatRoomID)
		return
	}

	// Send message only to users in this chat room
	for userID := range roomMembers {
		if userID == excludeUserID {
			continue // Skip the sender
		}

		client, exists := h.Clients[userID]
		if !exists {
			log.Printf("⚠️ User %d not connected (was in chat room %d)", userID, chatRoomID)
			continue
		}

		select {
		case client.Send <- data:
			log.Printf("✅ Message sent to user %d in chat room %d", userID, chatRoomID)
		default:
			log.Printf("⚠️ User %d's send buffer is full", userID)
		}
	}
}

// GetConnectedUsers returns a list of currently connected user IDs
func (h *Hub) GetConnectedUsers() []uint {
	h.mu.RLock()
	defer h.mu.RUnlock()

	users := make([]uint, 0, len(h.Clients))
	for userID := range h.Clients {
		users = append(users, userID)
	}
	return users
}

// IsUserConnected checks if a user is currently connected
func (h *Hub) IsUserConnected(userID uint) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, exists := h.Clients[userID]
	return exists
}

// handleChatMessage handles incoming chat messages
func (h *Hub) handleChatMessage(client *Client, message *Message) error {
	log.Printf("💬 Chat message from user %d: %s", client.ID, message.Content)
	
	// Broadcast to chat room (excluding sender)
	h.SendToChatRoom(message.ChatRoomID, message, client.ID)
	
	return nil
}

// handleTypingIndicator handles typing indicators
func (h *Hub) handleTypingIndicator(client *Client, message *Message) error {
	log.Printf("⌨️ Typing indicator from user %d in chat room %d", client.ID, message.ChatRoomID)
	
	// Broadcast typing indicator to chat room (excluding sender)
	h.SendToChatRoom(message.ChatRoomID, message, client.ID)
	
	return nil
}

// handleReadReceipt handles read receipts
func (h *Hub) handleReadReceipt(client *Client, message *Message) error {
	log.Printf("👁️ Read receipt from user %d in chat room %d", client.ID, message.ChatRoomID)
	
	// Broadcast read receipt to chat room (excluding sender)
	h.SendToChatRoom(message.ChatRoomID, message, client.ID)
	
	return nil
}

// handlePing handles ping messages for connection health
func (h *Hub) handlePing(client *Client, message *Message) error {
	// Send pong response
	pongMessage := &Message{
		Type: "pong",
		Timestamp: time.Now(),
	}
	
	data, err := json.Marshal(pongMessage)
	if err != nil {
		return err
	}
	
	select {
	case client.Send <- data:
	default:
		log.Printf("⚠️ Could not send pong to user %d", client.ID)
	}
	
	return nil
}

// handleServiceRequest handles new service request notifications
func (h *Hub) handleServiceRequest(client *Client, message *Message) error {
	log.Printf("🔧 Service request notification: %v", message.Data)
	
	// Broadcast to all available workers in the same category
	if requestData, ok := message.Data.(map[string]interface{}); ok {
		if categoryID, exists := requestData["category_id"]; exists {
			h.broadcastToWorkersInCategory(uint(categoryID.(float64)), message)
		}
	}
	
	return nil
}

// handleWorkerAvailability handles worker availability updates
func (h *Hub) handleWorkerAvailability(client *Client, message *Message) error {
	log.Printf("👷 Worker availability update from user %d: %v", client.ID, message.Data)
	
	// Update worker's availability status
	if availabilityData, ok := message.Data.(map[string]interface{}); ok {
		if isAvailable, exists := availabilityData["is_available"]; exists {
			log.Printf("👷 Worker %d availability: %v", client.ID, isAvailable)
		}
	}
	
	return nil
}

// handleRequestAccepted handles service request acceptance
func (h *Hub) handleRequestAccepted(client *Client, message *Message) error {
	log.Printf("✅ Request accepted by worker %d", client.ID)
	
	// Notify the customer that their request was accepted
	if requestData, ok := message.Data.(map[string]interface{}); ok {
		if customerID, exists := requestData["customer_id"]; exists {
			h.SendToUser(uint(customerID.(float64)), message)
		}
	}
	
	return nil
}

// handleRequestDeclined handles service request decline
func (h *Hub) handleRequestDeclined(client *Client, message *Message) error {
	log.Printf("❌ Request declined by worker %d", client.ID)
	
	// Notify the customer that their request was declined
	if requestData, ok := message.Data.(map[string]interface{}); ok {
		if customerID, exists := requestData["customer_id"]; exists {
			h.SendToUser(uint(customerID.(float64)), message)
		}
	}
	
	return nil
}

// broadcastToWorkersInCategory sends a message to all available workers in a specific category
func (h *Hub) broadcastToWorkersInCategory(categoryID uint, message *Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("❌ Error marshaling message: %v", err)
		return
	}

	// This would need to be enhanced to check worker categories
	// For now, broadcast to all workers
	for userID, client := range h.Clients {
		if client.UserType == "worker" {
			select {
			case client.Send <- data:
				log.Printf("✅ Service request sent to worker %d", userID)
			default:
				log.Printf("⚠️ Worker %d's send buffer is full", userID)
			}
		}
	}
}

// SendServiceRequestNotification sends a new service request notification to available workers
func (h *Hub) SendServiceRequestNotification(request interface{}) {
	message := &Message{
		Type: "service_request",
		Data: request,
		Timestamp: time.Now(),
	}
	
	h.Broadcast <- message
}

// SendWorkerAvailabilityUpdate sends worker availability updates
func (h *Hub) SendWorkerAvailabilityUpdate(workerID uint, isAvailable bool) {
	message := &Message{
		Type: "worker_availability",
		SenderID: workerID,
		SenderType: "worker",
		Data: map[string]interface{}{
			"is_available": isAvailable,
			"worker_id": workerID,
		},
		Timestamp: time.Now(),
	}
	
	h.Broadcast <- message
}

// SendRequestAcceptedNotification sends notification when a request is accepted
func (h *Hub) SendRequestAcceptedNotification(requestID uint, workerID uint, customerID uint) {
	message := &Message{
		Type: "request_accepted",
		SenderID: workerID,
		SenderType: "worker",
		Data: map[string]interface{}{
			"request_id": requestID,
			"worker_id": workerID,
			"customer_id": customerID,
		},
		Timestamp: time.Now(),
	}
	
	h.SendToUser(customerID, message)
}

// SendRequestDeclinedNotification sends notification when a request is declined
func (h *Hub) SendRequestDeclinedNotification(requestID uint, workerID uint, customerID uint) {
	message := &Message{
		Type: "request_declined",
		SenderID: workerID,
		SenderType: "worker",
		Data: map[string]interface{}{
			"request_id": requestID,
			"worker_id": workerID,
			"customer_id": customerID,
		},
		Timestamp: time.Now(),
	}
	
	h.SendToUser(customerID, message)
}
