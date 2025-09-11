package websocket

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512
)

// Error constants
var (
	ErrClientBufferFull = errors.New("client send buffer is full")
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for now, can be restricted later
	},
}

// ServeWebSocket handles the WebSocket connection upgrade and client management
func ServeWebSocket(hub *Hub, w http.ResponseWriter, r *http.Request, userID uint, userType string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("❌ WebSocket upgrade failed: %v", err)
		return
	}

	client := &Client{
		Hub:      hub,
		ID:       userID,
		UserType: userType,
		Conn:     conn,
		Send:     make(chan []byte, 256),
	}

	client.Hub.Register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()
}

// readPump pumps messages from the WebSocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, messageBytes, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("❌ WebSocket read error: %v", err)
			}
			break
		}

		// Parse the incoming message
		var message Message
		if err := json.Unmarshal(messageBytes, &message); err != nil {
			log.Printf("❌ Error unmarshaling message: %v", err)
			continue
		}

		// Set message metadata
		message.SenderID = c.ID
		message.SenderType = c.UserType
		message.Timestamp = time.Now()

		// Handle the message based on its type
		if handler, exists := c.Hub.MessageHandlers[message.Type]; exists {
			if err := handler(c, &message); err != nil {
				log.Printf("❌ Error handling message: %v", err)
			}
		} else {
			log.Printf("⚠️ Unknown message type: %s", message.Type)
		}
	}
}

// writePump pumps messages from the hub to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}

			w.Write(message)

			// Add queued chat messages to the current WebSocket message
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// SendMessage sends a message to this specific client
func (c *Client) SendMessage(message *Message) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	select {
	case c.Send <- data:
		return nil
	default:
		return ErrClientBufferFull
	}
}

// SendTypingIndicator sends a typing indicator to the client
func (c *Client) SendTypingIndicator(chatRoomID uint, isTyping bool) error {
	message := &Message{
		Type:       "typing",
		ChatRoomID: chatRoomID,
		Data: map[string]interface{}{
			"is_typing": isTyping,
		},
		Timestamp: time.Now(),
	}

	return c.SendMessage(message)
}

// SendReadReceipt sends a read receipt to the client
func (c *Client) SendReadReceipt(chatRoomID uint, messageID uint) error {
	message := &Message{
		Type:       "read_receipt",
		ChatRoomID: chatRoomID,
		Data: map[string]interface{}{
			"message_id": messageID,
			"read_at":    time.Now(),
		},
		Timestamp: time.Now(),
	}

	return c.SendMessage(message)
}

// SendChatMessage sends a chat message to the client
func (c *Client) SendChatMessage(chatRoomID uint, senderID uint, senderType string, content string) error {
	message := &Message{
		Type:        "chat",
		ChatRoomID:  chatRoomID,
		SenderID:    senderID,
		SenderType:  senderType,
		Content:     content,
		Timestamp:   time.Now(),
	}

	return c.SendMessage(message)
}

// SendSystemMessage sends a system message to the client
func (c *Client) SendSystemMessage(chatRoomID uint, content string, data interface{}) error {
	message := &Message{
		Type:        "system",
		ChatRoomID:  chatRoomID,
		Content:     content,
		Data:        data,
		Timestamp:   time.Now(),
	}

	return c.SendMessage(message)
}

// SendError sends an error message to the client
func (c *Client) SendError(errorType string, message string) error {
	errorMessage := &Message{
		Type: "error",
		Data: map[string]interface{}{
			"error_type": errorType,
			"message":    message,
		},
		Timestamp: time.Now(),
	}

	return c.SendMessage(errorMessage)
}

// Close closes the client connection
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.Conn != nil {
		c.Conn.Close()
	}
	
	close(c.Send)
}

// IsConnected checks if the client is still connected
func (c *Client) IsConnected() bool {
	return c.Conn != nil
}

// GetConnectionInfo returns connection information
func (c *Client) GetConnectionInfo() map[string]interface{} {
	return map[string]interface{}{
		"user_id":   c.ID,
		"user_type": c.UserType,
		"connected": c.IsConnected(),
	}
}
