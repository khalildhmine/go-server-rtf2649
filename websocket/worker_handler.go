package websocket

import (
	"log"
	"net/http"
	"time"

	"repair-service-server/database"
	"repair-service-server/models"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var workerUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

type WorkerHandler struct {
	clients map[*websocket.Conn]bool
}

func NewWorkerHandler() *WorkerHandler {
	return &WorkerHandler{
		clients: make(map[*websocket.Conn]bool),
	}
}

func (h *WorkerHandler) HandleWorker(c *gin.Context) {
	// Authenticate the user
	userID, exists := c.Get("user_id")
	if !exists {
		log.Printf("❌ No user ID found for worker WebSocket")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Check if user is a worker
	var workerProfile models.WorkerProfile
	if err := database.DB.Where("user_id = ?", userID).First(&workerProfile).Error; err != nil {
		log.Printf("❌ Worker profile not found for user %v", userID)
		c.JSON(http.StatusForbidden, gin.H{"error": "Worker profile required"})
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := workerUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("❌ Worker WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	// Add client to the map
	h.clients[conn] = true
	log.Printf("👷 Worker connected: %v", userID)

	// Send welcome message
	welcomeMsg := map[string]interface{}{
		"type":      "connected",
		"message":   "Worker WebSocket connected successfully",
		"timestamp": time.Now().UTC(),
	}
	conn.WriteJSON(welcomeMsg)

	// Handle incoming messages
	for {
		var msg map[string]interface{}
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("❌ Worker WebSocket read error: %v", err)
			break
		}

		log.Printf("📱 Worker WebSocket message: %v", msg)
		
		// Handle different message types
		if msgType, ok := msg["type"].(string); ok {
			switch msgType {
			case "ping":
				// Respond to ping with pong
				pongMsg := map[string]interface{}{
					"type":      "pong",
					"timestamp": time.Now().UTC(),
				}
				conn.WriteJSON(pongMsg)
			default:
				log.Printf("📱 Unknown worker message type: %s", msgType)
			}
		}
	}

	// Remove client from the map
	delete(h.clients, conn)
	log.Printf("👷 Worker disconnected: %v", userID)
}

// BroadcastToWorkers sends a message to all connected workers
func (h *WorkerHandler) BroadcastToWorkers(messageType string, data interface{}) {
	message := map[string]interface{}{
		"type":      messageType,
		"data":      data,
		"timestamp": time.Now().UTC(),
	}

	for conn := range h.clients {
		if err := conn.WriteJSON(message); err != nil {
			log.Printf("❌ Failed to send message to worker: %v", err)
			delete(h.clients, conn)
		}
	}
}

// BroadcastNewServiceRequest notifies workers about new service requests
func (h *WorkerHandler) BroadcastNewServiceRequest(serviceRequest models.CustomerServiceRequest) {
	h.BroadcastToWorkers("new_service_request", serviceRequest)
}

// BroadcastServiceRequestUpdate notifies workers about service request updates
func (h *WorkerHandler) BroadcastServiceRequestUpdate(serviceRequest models.CustomerServiceRequest) {
	h.BroadcastToWorkers("service_request_updated", serviceRequest)
}

// BroadcastNotification sends a notification to workers
func (h *WorkerHandler) BroadcastNotification(notification models.Notification) {
	h.BroadcastToWorkers("notification", notification)
}
