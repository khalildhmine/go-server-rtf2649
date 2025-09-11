package websocket

import (
	"fmt"
	"log"
	"net/http"
	"repair-service-server/database"
	"repair-service-server/models"
	"repair-service-server/services"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var aiUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

type AIChatHandler struct {
	aiService *services.AIService
	clients   map[*websocket.Conn]bool
	broadcast chan []byte
}

func NewAIChatHandler() *AIChatHandler {
	return &AIChatHandler{
		aiService: services.NewAIService(),
		clients:   make(map[*websocket.Conn]bool),
		broadcast: make(chan []byte),
	}
}

func (h *AIChatHandler) HandleAIChat(c *gin.Context) {
	conn, err := aiUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("❌ WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	h.clients[conn] = true
	log.Printf("🔌 AI Chat WebSocket connected")

	// Handle messages
	for {
		var msg map[string]interface{}
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("❌ WebSocket read error: %v", err)
			delete(h.clients, conn)
			break
		}

		h.handleMessage(conn, msg)
	}
}

func (h *AIChatHandler) handleMessage(conn *websocket.Conn, msg map[string]interface{}) {
	msgType, ok := msg["type"].(string)
	if !ok {
		log.Printf("⚠️ Invalid message type")
		return
	}

	switch msgType {
	case "user_input":
		h.handleUserInput(conn, msg)
	case "card_action":
		h.handleCardAction(conn, msg)
	case "ping":
		h.handlePing(conn)
	default:
		log.Printf("⚠️ Unknown message type: %s", msgType)
	}
}

func (h *AIChatHandler) handleUserInput(conn *websocket.Conn, msg map[string]interface{}) {
	// Extract message data
	message, _ := msg["message"].(string)
	messageType, _ := msg["messageType"].(string)
	imageUri, _ := msg["imageUri"].(string)
	voiceUri, _ := msg["voiceUri"].(string)
	userID, _ := msg["userId"].(float64)
	language, _ := msg["language"].(string)
	conversationHistory, _ := msg["conversationHistory"].([]interface{})

	// Convert conversation history
	var history []map[string]interface{}
	for _, h := range conversationHistory {
		if hMap, ok := h.(map[string]interface{}); ok {
			history = append(history, hMap)
		}
	}

	// Process with AI service
	response, err := h.aiService.ProcessUserInput(
		message,
		messageType,
		imageUri,
		voiceUri,
		uint(userID),
		language,
		history,
	)

	if err != nil {
		log.Printf("❌ AI processing error: %v", err)
		h.sendError(conn, "Failed to process your request. Please try again.")
		return
	}

	// Send response back to client
	h.sendResponse(conn, response)
}

func (h *AIChatHandler) handlePing(conn *websocket.Conn) {
	h.sendMessage(conn, map[string]interface{}{
		"type": "pong",
		"timestamp": time.Now().Unix(),
	})
}

func (h *AIChatHandler) sendResponse(conn *websocket.Conn, response *services.AIResponse) {
	msg := map[string]interface{}{
		"type": "ai_response",
		"text": response.Text,
	}

	if response.Card != nil {
		msg["card"] = response.Card
	}

	h.sendMessage(conn, msg)
}

func (h *AIChatHandler) sendError(conn *websocket.Conn, errorMsg string) {
	h.sendMessage(conn, map[string]interface{}{
		"type": "ai_error",
		"error": errorMsg,
	})
}

func (h *AIChatHandler) handleCardAction(conn *websocket.Conn, msg map[string]interface{}) {
	// Extract card action data
	action, _ := msg["action"].(string)
	// Accept workerId as number or string; also capture workerName as fallback
	var workerIDNum uint
	if widFloat, ok := msg["workerId"].(float64); ok && widFloat > 0 {
		workerIDNum = uint(widFloat)
	} else if widStr, ok := msg["workerId"].(string); ok {
		if n, err := strconv.Atoi(widStr); err == nil && n > 0 {
			workerIDNum = uint(n)
		}
	}
	workerName, _ := msg["workerName"].(string)
	userID, _ := msg["userId"].(float64)

	log.Printf("🔍 Card action received: %s for workerId=%v workerName=%s by user %v", action, workerIDNum, workerName, userID)

	if action == "Accept" {
		// Check if worker is still available
		var worker models.WorkerProfile
		var err error
		if workerIDNum > 0 {
			err = database.DB.Where("id = ? AND is_available = ?", workerIDNum, true).First(&worker).Error
		} else if workerName != "" {
			err = database.DB.Joins("JOIN users ON users.id = worker_profiles.user_id").
				Where("users.full_name = ? AND worker_profiles.is_available = ?", workerName, true).
				First(&worker).Error
		} else {
			err = fmt.Errorf("no worker identifier provided")
		}
		if err != nil {
			log.Printf("⚠️ Worker not available: %v", err)
			h.sendMessage(conn, map[string]interface{}{
				"type": "ai_response",
				"text": "Désolé, ce professionnel n'est plus disponible. Je vais vous trouver un autre professionnel.",
				"card": nil,
			})
			return
		}

		// Check if worker has no active requests
		var activeCount int64
		database.DB.Model(&models.CustomerServiceRequest{}).Where("assigned_worker_id = ? AND status IN (?)", worker.ID, []string{"accepted", "in_progress"}).Count(&activeCount)
		if activeCount > 0 {
			log.Printf("⚠️ Worker %v is busy with %d active requests", worker.ID, activeCount)
			h.sendMessage(conn, map[string]interface{}{
				"type": "ai_response",
				"text": "Désolé, ce professionnel est actuellement occupé. Je vais vous trouver un autre professionnel disponible.",
				"card": nil,
			})
			return
		}

		// Create service request as broadcast (unassigned) so the worker can choose to accept
		lat := 18.117001
		lng := -15.949912
		expiresAt := time.Now().Add(15 * time.Minute)
		serviceRequest := models.CustomerServiceRequest{
			CustomerID:      uint(userID),
			CategoryID:      worker.CategoryID,
			Title:           "Demande via IA",
			Description:     "Service demandé via l'assistant IA",
			Status:          "broadcast",
			Priority:        "normal",
			LocationAddress: "Adresse par défaut", // You might want to get this from user's default address
			LocationCity:    "Nouakchott",
			LocationLat:     &lat, // You might want to get this from user's location
			LocationLng:     &lng,
			ExpiresAt:       &expiresAt,
		}

		err = database.DB.Create(&serviceRequest).Error
		if err != nil {
			log.Printf("❌ Failed to create service request: %v", err)
			h.sendMessage(conn, map[string]interface{}{
				"type": "ai_error",
				"error": "Erreur lors de la création de la demande de service",
			})
			return
		}

		log.Printf("✅ Service request created in broadcast: %v for category %v", serviceRequest.ID, worker.CategoryID)

		// Watch for status changes and notify client
		go func(requestID uint, client *websocket.Conn) {
			deadline := time.Now().Add(15 * time.Minute)
			for time.Now().Before(deadline) {
				var req models.CustomerServiceRequest
				if err := database.DB.Where("id = ?", requestID).First(&req).Error; err != nil {
					log.Printf("⚠️ Watcher: failed to load request %v: %v", requestID, err)
					return
				}
				if req.Status == "accepted" && req.AssignedWorkerID != nil {
					h.sendMessage(client, map[string]interface{}{
						"type": "ai_response",
						"text": "Le professionnel a accepté votre demande et est en route.",
						"card": nil,
					})
					return
				}
				if req.Status == "declined" || req.Status == "cancelled" || req.Status == "expired" {
					h.sendMessage(client, map[string]interface{}{
						"type": "ai_response",
						"text": "Le professionnel a refusé ou la demande a expiré. Je cherche d'autres options pour vous.",
						"card": nil,
					})
					return
				}
				time.Sleep(2 * time.Second)
			}
		}(serviceRequest.ID, conn)
		h.sendMessage(conn, map[string]interface{}{
			"type": "ai_response",
			"text": "Parfait ! Votre demande a été envoyée au professionnel. Nous attendons sa confirmation.",
			"card": nil,
		})

	} else if action == "Decline" {
		h.sendMessage(conn, map[string]interface{}{
			"type": "ai_response",
			"text": "D'accord, je vais vous trouver d'autres options.",
			"card": nil,
		})
	}
}

func (h *AIChatHandler) sendMessage(conn *websocket.Conn, msg map[string]interface{}) {
	err := conn.WriteJSON(msg)
	if err != nil {
		log.Printf("❌ WebSocket write error: %v", err)
		delete(h.clients, conn)
	}
}

func (h *AIChatHandler) BroadcastToAll(msg map[string]interface{}) {
	for client := range h.clients {
		h.sendMessage(client, msg)
	}
}
