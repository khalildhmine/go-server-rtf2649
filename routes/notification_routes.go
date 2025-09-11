package routes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"repair-service-server/database"
	"repair-service-server/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// getUserPreferredLanguage tries to read the user's preferred language.
// Falls back to "en" if not set or on any error (including missing column).
func getUserPreferredLanguage(userID uint) string {
    var lang string
    err := database.DB.Raw("SELECT COALESCE(preferred_language, '') FROM users WHERE id = ?", userID).Scan(&lang).Error
    if err != nil || lang == "" {
        if err != nil {
            log.Printf("⚠️ Could not read preferred_language for user %d (defaulting to en): %v", userID, err)
        }
        return "en"
    }
    return lang
}

// getLocalizedStatusMessage returns localized title/body/type for a status
func getLocalizedStatusMessage(status string, lang string) (string, string, string) {
    switch lang {
    case "fr", "ar", "zh":
    default:
        lang = "en"
    }

    type message struct{ title, body, ntype string }
    msgs := map[string]map[string]message{
        "en": {
            "accepted":   {"Service Request Accepted", "A professional has accepted your service request and is on the way!", "booking_accepted"},
            "in_progress": {"Work Started", "Your service professional has started working on your request.", "booking_in_progress"},
            "completed":   {"Service Completed", "Your service request has been completed. Please rate your experience.", "booking_completed"},
            "cancelled":   {"Service Cancelled", "Your service request has been cancelled.", "booking_cancelled"},
            "default":     {"Service Update", "Your service request status has been updated.", "system"},
        },
        "fr": {
            "accepted":   {"Demande acceptée", "Un professionnel a accepté votre demande et arrive !", "booking_accepted"},
            "in_progress": {"Travaux commencés", "Votre professionnel a commencé à travailler sur votre demande.", "booking_in_progress"},
            "completed":   {"Service terminé", "Votre demande est terminée. Merci d'évaluer votre expérience.", "booking_completed"},
            "cancelled":   {"Service annulé", "Votre demande de service a été annulée.", "booking_cancelled"},
            "default":     {"Mise à jour du service", "Le statut de votre demande a été mis à jour.", "system"},
        },
        "ar": {
            "accepted":   {"تم قبول الطلب", "تم قبول طلب خدمتك والمهني في الطريق!", "booking_accepted"},
            "in_progress": {"بدأ العمل", "بدأ المهني العمل على طلبك.", "booking_in_progress"},
            "completed":   {"اكتملت الخدمة", "تم إكمال طلب خدمتك. يرجى تقييم تجربتك.", "booking_completed"},
            "cancelled":   {"تم إلغاء الخدمة", "تم إلغاء طلب خدمتك.", "booking_cancelled"},
            "default":     {"تحديث الخدمة", "تم تحديث حالة طلب خدمتك.", "system"},
        },
        "zh": {
            "accepted":   {"服务请求已接受", "服务人员已接受您的请求，正在赶来！", "booking_accepted"},
            "in_progress": {"工作已开始", "服务人员已开始处理您的请求。", "booking_in_progress"},
            "completed":   {"服务已完成", "您的服务请求已完成。请为体验打分。", "booking_completed"},
            "cancelled":   {"服务已取消", "您的服务请求已被取消。", "booking_cancelled"},
            "default":     {"服务更新", "您的服务请求状态已更新。", "system"},
        },
    }

    mlang := msgs[lang]
    m, ok := mlang[status]
    if !ok {
        m = mlang["default"]
    }
    return m.title, m.body, m.ntype
}

// RegisterPushToken registers a push token for a user
func RegisterPushToken(c *gin.Context) {
	userID := c.GetUint("user_id")
	
	var request struct {
		PushToken string `json:"push_token" binding:"required"`
		Platform  string `json:"platform" binding:"required"`
		DeviceID  string `json:"device_id"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if token already exists
	var existingToken models.PushToken
	err := database.DB.Where("token = ?", request.PushToken).First(&existingToken).Error
	
	if err == gorm.ErrRecordNotFound {
		// Create new token
		token := models.PushToken{
			UserID:   userID,
			Token:    request.PushToken,
			Platform: request.Platform,
			DeviceID: request.DeviceID,
			Active:   true,
		}
		
		if err := database.DB.Create(&token).Error; err != nil {
			log.Printf("❌ Error creating push token: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register push token"})
			return
		}
		
		log.Printf("✅ Push token registered for user %d", userID)
	} else if err != nil {
		log.Printf("❌ Error checking existing token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	} else {
		// Update existing token
		existingToken.UserID = userID
		existingToken.Platform = request.Platform
		existingToken.DeviceID = request.DeviceID
		existingToken.Active = true
		existingToken.UpdatedAt = time.Now()
		
		if err := database.DB.Save(&existingToken).Error; err != nil {
			log.Printf("❌ Error updating push token: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update push token"})
			return
		}
		
		log.Printf("✅ Push token updated for user %d", userID)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Push token registered successfully",
	})
}

// HasPushToken checks if the authenticated user has at least one active push token
func HasPushToken(c *gin.Context) {
    userID := c.GetUint("user_id")

    var count int64
    if err := database.DB.Model(&models.PushToken{}).Where("user_id = ? AND active = ?", userID, true).Count(&count).Error; err != nil {
        log.Printf("❌ Error checking push token existence: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "hasToken": count > 0,
    })
}

// GetUserNotifications gets all notifications for a user
func GetUserNotifications(c *gin.Context) {
	userID := c.GetUint("user_id")
	log.Printf("🔍 GetUserNotifications called for user ID: %d", userID)
	
	var notifications []models.Notification
	err := database.DB.Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(50).
		Find(&notifications).Error
	
	if err != nil {
		log.Printf("❌ Error fetching notifications: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch notifications"})
		return
	}

	log.Printf("📱 Found %d notifications for user %d", len(notifications), userID)
	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"notifications": notifications,
	})
}

// MarkNotificationAsRead marks a specific notification as read
func MarkNotificationAsRead(c *gin.Context) {
	userID := c.GetUint("user_id")
	notificationID := c.Param("id")
	
	// Convert string to uint
	id, err := strconv.ParseUint(notificationID, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification ID"})
		return
	}

	var notification models.Notification
	err = database.DB.Where("id = ? AND user_id = ?", id, userID).First(&notification).Error
	
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Notification not found"})
		} else {
			log.Printf("❌ Error finding notification: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		}
		return
	}

	notification.Read = true
	notification.UpdatedAt = time.Now()
	
	if err := database.DB.Save(&notification).Error; err != nil {
		log.Printf("❌ Error updating notification: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update notification"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Notification marked as read",
	})
}

// MarkAllNotificationsAsRead marks all notifications as read for a user
func MarkAllNotificationsAsRead(c *gin.Context) {
	userID := c.GetUint("user_id")
	
	err := database.DB.Model(&models.Notification{}).
		Where("user_id = ? AND read = ?", userID, false).
		Updates(map[string]interface{}{
			"read":       true,
			"updated_at": time.Now(),
		}).Error
	
	if err != nil {
		log.Printf("❌ Error marking all notifications as read: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark notifications as read"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "All notifications marked as read",
	})
}

// GetUnreadCount returns the count of unread notifications for the user
func GetUnreadCount(c *gin.Context) {
	userID := c.GetUint("user_id")
	log.Printf("🔍 GetUnreadCount called for user ID: %d", userID)
	
	var count int64
	err := database.DB.Model(&models.Notification{}).
		Where("user_id = ? AND read = ?", userID, false).
		Count(&count).Error

	if err != nil {
		log.Printf("❌ Error getting unread count: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get unread count"})
		return
	}

	log.Printf("📊 Unread count for user %d: %d", userID, count)
	c.JSON(http.StatusOK, gin.H{
		"count": count,
	})
}

// SendPushNotification sends a push notification to a user (internal function)
func SendPushNotification(userID uint, title, body, notificationType string, data map[string]interface{}) error {
	log.Printf("🔔 SendPushNotification called for user %d: %s - %s", userID, title, body)
	
	// Get user's push tokens
	var tokens []models.PushToken
	err := database.DB.Where("user_id = ? AND active = ?", userID, true).Find(&tokens).Error
	if err != nil {
		log.Printf("❌ Error fetching push tokens for user %d: %v", userID, err)
		return err
	}

	log.Printf("🔍 Found %d active push tokens for user %d", len(tokens), userID)
	for i, token := range tokens {
		log.Printf("🔑 Token %d: %s (platform: %s)", i+1, token.Token, token.Platform)
	}

	if len(tokens) == 0 {
		log.Printf("⚠️ No push tokens found for user %d", userID)
		return nil
	}

	// Create notification record
	dataJSON, _ := json.Marshal(data)
	notification := models.Notification{
		UserID: userID,
		Title:  title,
		Body:   body,
		Type:   notificationType,
		Data:   string(dataJSON),
		Read:   false,
	}

	if err := database.DB.Create(&notification).Error; err != nil {
		log.Printf("❌ Error creating notification record for user %d: %v", userID, err)
		return err
	}

	log.Printf("✅ Notification record created in database for user %d", userID)

	// Send push notifications
	successCount := 0
	for i, token := range tokens {
		log.Printf("📱 Sending push notification %d/%d to user %d", i+1, len(tokens), userID)
		err := sendExpoPushNotification(token.Token, title, body, data)
		if err != nil {
			log.Printf("❌ Error sending push notification to token %s: %v", token.Token, err)
		} else {
			successCount++
			log.Printf("✅ Push notification %d/%d sent successfully to user %d", i+1, len(tokens), userID)
		}
	}

	log.Printf("📊 Push notification summary: %d/%d sent successfully to user %d", successCount, len(tokens), userID)
	return nil
}

// sendExpoPushNotification sends a notification via Expo Push API
func sendExpoPushNotification(token, title, body string, data map[string]interface{}) error {
	// Send to Expo Push API
	payload := map[string]interface{}{
		"to":          token,
		"title":       title,
		"body":        body,
		"data":        data,
		"sound":       "default",
		"priority":    "high",
		"channelId":   "service_updates",
	}

	bodyBytes, _ := json.Marshal(payload)
	log.Printf("📤 Sending Expo push notification to token: %s", token)
	log.Printf("📤 Payload: %s", string(bodyBytes))
	
	req, err := http.NewRequest("POST", "https://exp.host/--/api/v2/push/send", bytes.NewReader(bodyBytes))
	if err != nil {
		log.Printf("❌ Failed to create Expo request: %v", err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("❌ Expo request failed: %v", err)
		return err
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("❌ Failed to read Expo response: %v", err)
	} else {
		log.Printf("📥 Expo response (%d): %s", resp.StatusCode, string(respBody))
	}

	if resp.StatusCode >= 400 {
		log.Printf("❌ Expo push send failed: %s - %s", resp.Status, string(respBody))
		return fmt.Errorf("expo push failed: %s", resp.Status)
	}
	
	log.Printf("✅ Expo push notification sent successfully")
	return nil
}

// SendServiceStatusNotification sends a notification when service status changes
func SendServiceStatusNotification(userID uint, serviceRequestID uint, status string) error {
	log.Printf("🔔 SendServiceStatusNotification called: userID=%d, serviceRequestID=%d, status=%s", userID, serviceRequestID, status)
	
	// Check if notification already exists for this service request and status
	var existingNotification models.Notification
	err := database.DB.Where("user_id = ? AND type = ? AND data LIKE ?", 
		userID, 
		fmt.Sprintf("booking_%s", status),
		fmt.Sprintf("%%\"service_request_id\":%d%%", serviceRequestID)).
		First(&existingNotification).Error
	
	if err == nil {
		log.Printf("⚠️ Notification already exists for user %d, service request %d, status %s - skipping", userID, serviceRequestID, status)
		return nil // Don't send duplicate notification
	}
	
	// Localize message by user preferred language
	lang := getUserPreferredLanguage(userID)
	title, body, notificationType := getLocalizedStatusMessage(status, lang)

	log.Printf("📝 Notification content: %s - %s (type: %s)", title, body, notificationType)

	data := map[string]interface{}{
		"service_request_id": serviceRequestID,
		"status":            status,
		"type":              "status_update",
	}

	err = SendPushNotification(userID, title, body, notificationType, data)
	if err != nil {
		log.Printf("❌ SendServiceStatusNotification failed for user %d: %v", userID, err)
	} else {
		log.Printf("✅ SendServiceStatusNotification completed for user %d", userID)
	}
	
	return err
}

// Campaign notification structures
type NotificationCampaign struct {
	ID               string    `json:"id"`
	Type             string    `json:"type"`
	Title            string    `json:"title"`
	Body             string    `json:"body"`
	Action           string    `json:"action"`
	Data             map[string]interface{} `json:"data,omitempty"`
	ScheduledFor     *time.Time `json:"scheduledFor,omitempty"`
	UserID           uint      `json:"userId"`
	ServiceRequestID *uint     `json:"serviceRequestId,omitempty"`
}

type UserActivity struct {
	UserID         uint      `json:"userId"`
	LastActiveAt   time.Time `json:"lastActiveAt"`
	LastServiceAt  *time.Time `json:"lastServiceAt,omitempty"`
	TotalServices  int       `json:"totalServices"`
	IsActive       bool      `json:"isActive"`
}

type FeedbackSubmission struct {
	ServiceRequestID *uint   `json:"service_request_id,omitempty"`
	Feedback         string  `json:"feedback"`
	WorkerName       *string `json:"worker_name,omitempty"`
	ServiceTitle     *string `json:"service_title,omitempty"`
}

// SendCampaignNotification sends a campaign notification immediately
func SendCampaignNotification(c *gin.Context) {
	userID := c.GetUint("user_id")
	
	var campaign NotificationCampaign
	if err := c.ShouldBindJSON(&campaign); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set user ID from context
	campaign.UserID = userID

	// Send the notification
	err := SendPushNotification(userID, campaign.Title, campaign.Body, "system", campaign.Data)
	if err != nil {
		log.Printf("❌ SendCampaignNotification failed for user %d: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send notification"})
		return
	}

	log.Printf("✅ Campaign notification sent: %s to user %d", campaign.Type, userID)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Campaign notification sent"})
}

// ScheduleCampaignNotification schedules a campaign notification for later
func ScheduleCampaignNotification(c *gin.Context) {
	userID := c.GetUint("user_id")
	
	var campaign NotificationCampaign
	if err := c.ShouldBindJSON(&campaign); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set user ID from context
	campaign.UserID = userID

	// Convert data to JSON string
	var dataJSON string
	if campaign.Data != nil {
		dataBytes, err := json.Marshal(campaign.Data)
		if err != nil {
			log.Printf("❌ Error marshaling campaign data: %v", err)
			dataJSON = "{}"
		} else {
			dataJSON = string(dataBytes)
		}
	} else {
		dataJSON = "{}"
	}

	// Store the scheduled notification in database
	notification := models.Notification{
		UserID: userID,
		Title:  campaign.Title,
		Body:   campaign.Body,
		Type:   "system",
		Data:   dataJSON,
		Read:   false,
	}

	// If scheduled for later, set a flag or use a separate scheduled notifications table
	// For now, we'll store it as a regular notification
	if err := database.DB.Create(&notification).Error; err != nil {
		log.Printf("❌ ScheduleCampaignNotification failed for user %d: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to schedule notification"})
		return
	}

	log.Printf("✅ Campaign notification scheduled: %s for user %d", campaign.Type, userID)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Campaign notification scheduled"})
}

// TrackUserActivity tracks user activity for inactivity detection
func TrackUserActivity(c *gin.Context) {
	userID := c.GetUint("user_id")
	
	var activity UserActivity
	if err := c.ShouldBindJSON(&activity); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set user ID from context
	activity.UserID = userID

	// Store or update user activity
	// You might want to create a separate UserActivity model/table
	// For now, we'll just log it
	log.Printf("📊 User activity tracked: User %d, Last Active: %v, Total Services: %d, Active: %v", 
		activity.UserID, activity.LastActiveAt, activity.TotalServices, activity.IsActive)

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "User activity tracked"})
}

// SubmitFeedback handles feedback submission from users
func SubmitFeedback(c *gin.Context) {
	userID := c.GetUint("user_id")
	
	var feedback FeedbackSubmission
	if err := c.ShouldBindJSON(&feedback); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Store feedback in database
	// You might want to create a separate Feedback model/table
	// For now, we'll just log it
	log.Printf("💬 Feedback submitted: User %d, Service Request: %v, Feedback: %s, Worker: %v, Service: %v", 
		userID, feedback.ServiceRequestID, feedback.Feedback, feedback.WorkerName, feedback.ServiceTitle)

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Feedback submitted successfully"})
}

// CreateTestNotifications creates some test notifications for development
func CreateTestNotifications(c *gin.Context) {
	userID := c.GetUint("user_id")
	
	// Create some test notifications
	testNotifications := []models.Notification{
		{
			UserID: userID,
			Title:  "Service Complete!",
			Body:   "Your service is complete! How was your experience? Rate your technician & help us improve.",
			Type:   "booking_completed",
			Data:   `{"serviceRequestId": 123, "action": "rate_service"}`,
			Read:   false,
		},
		{
			UserID: userID,
			Title:  "Thank You for Trusting Us!",
			Body:   "Thank you for trusting us today! Here's your service summary & technician details for your records.",
			Type:   "system",
			Data:   `{"serviceRequestId": 123, "action": "view_summary"}`,
			Read:   false,
		},
		{
			UserID: userID,
			Title:  "Rainy Season Alert",
			Body:   "Rainy season is here! Get your roof inspected before leaks happen.",
			Type:   "promotion",
			Data:   `{"campaignType": "rainy_season", "action": "book_inspection"}`,
			Read:   true,
		},
	}

	// Create notifications in database
	for _, notification := range testNotifications {
		if err := database.DB.Create(&notification).Error; err != nil {
			log.Printf("❌ Error creating test notification: %v", err)
		}
	}

	log.Printf("✅ Created %d test notifications for user %d", len(testNotifications), userID)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Test notifications created"})
}
