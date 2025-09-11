package jobs

import (
	"log"
	"time"
	"repair-service-server/database"
	"repair-service-server/models"
)

// ExpirationJob handles expired service requests
type ExpirationJob struct {
	stopChan chan bool
}

// NewExpirationJob creates a new expiration job
func NewExpirationJob() *ExpirationJob {
	return &ExpirationJob{
		stopChan: make(chan bool),
	}
}

// Start begins the expiration job
func (j *ExpirationJob) Start() {
	go j.run()
	log.Println("🚀 Expiration job started")
}

// Stop stops the expiration job
func (j *ExpirationJob) Stop() {
	j.stopChan <- true
	log.Println("🛑 Expiration job stopped")
}

// run executes the expiration job
func (j *ExpirationJob) run() {
	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			j.checkExpiredRequests()
		case <-j.stopChan:
			return
		}
	}
}

// checkExpiredRequests finds and expires service requests
func (j *ExpirationJob) checkExpiredRequests() {
	var expiredRequests []models.CustomerServiceRequest
	
	// Find requests that have expired but are still in broadcast status
	err := database.DB.Where("status = ? AND expires_at <= ?", 
		models.RequestStatusBroadcast, time.Now()).Find(&expiredRequests).Error
	
	if err != nil {
		log.Printf("❌ Error checking expired requests: %v", err)
		return
	}

	if len(expiredRequests) > 0 {
		log.Printf("⏰ Found %d expired service requests", len(expiredRequests))
		
		for _, request := range expiredRequests {
			j.expireRequest(request)
		}
	}
}

// expireRequest marks a request as expired
func (j *ExpirationJob) expireRequest(request models.CustomerServiceRequest) {
	// Update status to expired
	request.Status = models.RequestStatusExpired
	
	err := database.DB.Save(&request).Error
	if err != nil {
		log.Printf("❌ Failed to expire request %d: %v", request.ID, err)
		return
	}

	log.Printf("✅ Request %d expired successfully", request.ID)
	
	// TODO: Send notification to customer about expired request
	// TODO: Send notification to workers that the request is no longer available
}

// GetExpiredRequests returns all expired requests for testing/debugging
func (j *ExpirationJob) GetExpiredRequests() ([]models.CustomerServiceRequest, error) {
	var requests []models.CustomerServiceRequest
	
	err := database.DB.Where("status = ?", models.RequestStatusExpired).
		Order("expires_at DESC").
		Find(&requests).Error
	
	return requests, err
}
