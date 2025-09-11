package websocket

import (
	"log"
	"time"

	"repair-service-server/database"
	"repair-service-server/models"
)

// ServiceBroadcaster handles broadcasting service requests to workers
type ServiceBroadcaster struct {
	hub *Hub
}

// NewServiceBroadcaster creates a new service broadcaster
func NewServiceBroadcaster(hub *Hub) *ServiceBroadcaster {
	return &ServiceBroadcaster{
		hub: hub,
	}
}

// BroadcastServiceRequest broadcasts a new service request to all connected workers
func (sb *ServiceBroadcaster) BroadcastServiceRequest(serviceRequest models.CustomerServiceRequest) {
	if sb.hub == nil {
		log.Printf("⚠️ WebSocket hub not available for service request broadcast")
		return
	}
	
	// Load service request with relationships for complete data
	var fullRequest models.CustomerServiceRequest
	if err := database.DB.
		Preload("Customer").
		Preload("Category").
		Preload("ServiceOption").
		First(&fullRequest, serviceRequest.ID).Error; err != nil {
		log.Printf("❌ Failed to load service request details: %v", err)
		return
	}
	
	// Create WebSocket message for service request
	websocketMessage := &Message{
		Type: "service_request",
		Data: map[string]interface{}{
			"request_id":           fullRequest.ID,
			"title":                fullRequest.Title,
			"description":          fullRequest.Description,
			"category_id":          fullRequest.CategoryID,
			"service_option_id":    fullRequest.ServiceOptionID,
			"location_address":     fullRequest.LocationAddress,
			"location_city":        fullRequest.LocationCity,
			"location_lat":         fullRequest.LocationLat,
			"location_lng":         fullRequest.LocationLng,
			"priority":             fullRequest.Priority,
			"budget":               fullRequest.Budget,
			"estimated_duration":   fullRequest.EstimatedDuration,
			"customer_name":        fullRequest.Customer.FullName,
			"category_name":        fullRequest.Category.Name,
			"created_at":           fullRequest.CreatedAt,
			"status":               fullRequest.Status,
		},
		Timestamp: time.Now(),
	}
	
	// Broadcast to all connected workers
	sb.hub.Broadcast <- websocketMessage
	
	log.Printf("📡 Service request %d broadcasted via WebSocket to all connected workers", serviceRequest.ID)
}

// NotifyWorker sends a service request notification to a specific worker
func (sb *ServiceBroadcaster) NotifyWorker(worker models.WorkerProfile, request models.CustomerServiceRequest, distance float64) {
	if sb.hub == nil {
		log.Printf("⚠️ WebSocket hub not available for worker notification")
		return
	}
	
	// Load service request with relationships for complete data
	var fullRequest models.CustomerServiceRequest
	if err := database.DB.
		Preload("Customer").
		Preload("Category").
		Preload("ServiceOption").
		First(&fullRequest, request.ID).Error; err != nil {
		log.Printf("❌ Failed to load service request details: %v", err)
		return
	}
	
	// Create WebSocket message for individual worker notification
	websocketMessage := &Message{
		Type: "service_request",
		Data: map[string]interface{}{
			"request_id":           fullRequest.ID,
			"title":                fullRequest.Title,
			"description":          fullRequest.Description,
			"category_id":          fullRequest.CategoryID,
			"service_option_id":    fullRequest.ServiceOptionID,
			"location_address":     fullRequest.LocationAddress,
			"location_city":        fullRequest.LocationCity,
			"location_lat":         fullRequest.LocationLat,
			"location_lng":         fullRequest.LocationLng,
			"priority":             fullRequest.Priority,
			"budget":               fullRequest.Budget,
			"estimated_duration":   fullRequest.EstimatedDuration,
			"customer_name":        fullRequest.Customer.FullName,
			"category_name":        fullRequest.Category.Name,
			"created_at":           fullRequest.CreatedAt,
			"status":               fullRequest.Status,
			"distance":             distance,
		},
		Timestamp: time.Now(),
	}
	
	// Send to specific worker
	sb.hub.SendToUser(worker.UserID, websocketMessage)
	
	log.Printf("📱 Service request %d sent to worker %d via WebSocket (%.2f km away)", 
		request.ID, worker.UserID, distance)
}
