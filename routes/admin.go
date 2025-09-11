package routes

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"repair-service-server/database"
	"repair-service-server/models"
	"repair-service-server/utils"
)

// Admin authentication middleware
func AdminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// Remove "Bearer " prefix
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}

		// Verify token
		claims, err := utils.VerifyToken(token)
		if err != nil {
			log.Printf("❌ Token verification failed: %v", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// Get user from database
		var user models.User
		if err := database.DB.First(&user, claims.UserID).Error; err != nil {
			log.Printf("❌ User not found: %v", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			c.Abort()
			return
		}

		// Check if user is admin
		if user.Role != models.RoleAdmin {
			log.Printf("❌ User %d is not admin, role: %s", user.ID, user.Role)
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			c.Abort()
			return
		}

		// Check if user is active
		if !user.IsActive {
			log.Printf("❌ Admin user %d is inactive", user.ID)
			c.JSON(http.StatusForbidden, gin.H{"error": "Account is inactive"})
			c.Abort()
			return
		}

		// Set user info in context
		c.Set("user_id", user.ID)
		c.Set("user", user)
		c.Next()
	}
}

// AdminLogin handles admin login
func AdminLogin(c *gin.Context) {
	var req struct {
		PhoneNumber string `json:"phone_number" binding:"required"`
		Password    string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Find user by phone number
	var user models.User
	if err := database.DB.Where("phone_number = ?", req.PhoneNumber).First(&user).Error; err != nil {
		log.Printf("❌ Admin login failed for phone %s: %v", req.PhoneNumber, err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Check if user is admin
	if user.Role != models.RoleAdmin {
		log.Printf("❌ Login attempt by non-admin user %d with role %s", user.ID, user.Role)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Admin access required"})
		return
	}

	// Check if user is active
	if !user.IsActive {
		log.Printf("❌ Login attempt by inactive admin user %d", user.ID)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Account is inactive"})
		return
	}

	// Verify password
	if !utils.CheckPasswordHash(req.Password, user.PasswordHash) {
		log.Printf("❌ Invalid password for admin user %d", user.ID)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Generate tokens
	token, err := utils.GenerateToken(user.ID, string(user.Role))
	if err != nil {
		log.Printf("❌ Failed to generate token for admin user %d: %v", user.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	refreshToken, err := utils.GenerateRefreshToken(user.ID)
	if err != nil {
		log.Printf("❌ Failed to generate refresh token for admin user %d: %v", user.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate refresh token"})
		return
	}

	log.Printf("✅ Admin user %d logged in successfully", user.ID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Login successful",
		"token":   token,
		"refresh_token": refreshToken,
		"user": gin.H{
			"id":                user.ID,
			"full_name":         user.FullName,
			"phone_number":      user.PhoneNumber,
			"role":              user.Role,
			"profile_picture_url": user.ProfilePictureURL,
			"is_active":         user.IsActive,
			"created_at":        user.CreatedAt,
			"updated_at":        user.UpdatedAt,
		},
	})
}

// AdminRefreshToken handles admin token refresh
func AdminRefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Verify refresh token
	claims, err := utils.VerifyRefreshToken(req.RefreshToken)
	if err != nil {
		log.Printf("❌ Refresh token verification failed: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	// Get user from database
	var user models.User
	if err := database.DB.First(&user, claims.UserID).Error; err != nil {
		log.Printf("❌ User not found: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	// Check if user is admin
	if user.Role != models.RoleAdmin {
		log.Printf("❌ User %d is not admin, role: %s", user.ID, user.Role)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Admin access required"})
		return
	}

	// Check if user is active
	if !user.IsActive {
		log.Printf("❌ Admin user %d is inactive", user.ID)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Account is inactive"})
		return
	}

	// Generate new token
	token, err := utils.GenerateToken(user.ID, string(user.Role))
	if err != nil {
		log.Printf("❌ Failed to generate token for admin user %d: %v", user.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	log.Printf("✅ Admin user %d token refreshed successfully", user.ID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Token refreshed successfully",
		"token":   token,
	})
}

// GetCurrentAdmin returns current admin user
func GetCurrentAdmin(c *gin.Context) {
	userID := c.GetUint("user_id")
	
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"user": gin.H{
			"id":                user.ID,
			"full_name":         user.FullName,
			"phone_number":      user.PhoneNumber,
			"role":              user.Role,
			"profile_picture_url": user.ProfilePictureURL,
			"is_active":         user.IsActive,
			"created_at":        user.CreatedAt,
			"updated_at":        user.UpdatedAt,
		},
	})
}

// GetAllUsers returns all users with pagination
func GetAllUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	role := c.Query("role")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}

	offset := (page - 1) * limit

	var users []models.User
	var total int64

	query := database.DB.Model(&models.User{})
	if role != "" {
		query = query.Where("role = ?", role)
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		log.Printf("❌ Failed to count users: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count users"})
		return
	}

	// Get users with pagination
	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&users).Error; err != nil {
		log.Printf("❌ Failed to fetch users: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}

	// Format response
	var userList []gin.H
	for _, user := range users {
		userList = append(userList, gin.H{
			"id":                user.ID,
			"full_name":         user.FullName,
			"phone_number":      user.PhoneNumber,
			"role":              user.Role,
			"profile_picture_url": user.ProfilePictureURL,
			"is_active":         user.IsActive,
			"created_at":        user.CreatedAt,
			"updated_at":        user.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    userList,
		"total":   total,
		"page":    page,
		"limit":   limit,
	})
}

// GetUserById returns user by ID
func GetUserById(c *gin.Context) {
	userID := c.Param("id")
	
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"id":                user.ID,
			"full_name":         user.FullName,
			"phone_number":      user.PhoneNumber,
			"role":              user.Role,
			"profile_picture_url": user.ProfilePictureURL,
			"is_active":         user.IsActive,
			"created_at":        user.CreatedAt,
			"updated_at":        user.UpdatedAt,
		},
	})
}

// UpdateUserStatus updates user status
func UpdateUserStatus(c *gin.Context) {
	userID := c.Param("id")
	
	var req struct {
		IsActive bool `json:"is_active" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Prevent admin from deactivating themselves
	adminID := c.GetUint("user_id")
	if user.ID == adminID && !req.IsActive {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot deactivate your own account"})
		return
	}

	user.IsActive = req.IsActive
	if err := database.DB.Save(&user).Error; err != nil {
		log.Printf("❌ Failed to update user status: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user status"})
		return
	}

	log.Printf("✅ User %d status updated to %v by admin %d", user.ID, req.IsActive, adminID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "User status updated successfully",
		"data": gin.H{
			"id":                user.ID,
			"full_name":         user.FullName,
			"phone_number":      user.PhoneNumber,
			"role":              user.Role,
			"profile_picture_url": user.ProfilePictureURL,
			"is_active":         user.IsActive,
			"created_at":        user.CreatedAt,
			"updated_at":        user.UpdatedAt,
		},
	})
}

// DeleteUser deletes a user
func DeleteUser(c *gin.Context) {
	userID := c.Param("id")
	adminID := c.GetUint("user_id")

	// Prevent admin from deleting themselves
	if userID == strconv.Itoa(int(adminID)) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete your own account"})
		return
	}

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Soft delete the user
	if err := database.DB.Delete(&user).Error; err != nil {
		log.Printf("❌ Failed to delete user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}

	log.Printf("✅ User %d deleted by admin %d", user.ID, adminID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "User deleted successfully",
	})
}

// GetDashboardStats returns dashboard statistics
func GetDashboardStats(c *gin.Context) {
	var stats struct {
		TotalUsers           int64 `json:"total_users"`
		TotalWorkers         int64 `json:"total_workers"`
		TotalCustomers       int64 `json:"total_customers"`
		TotalAdmins          int64 `json:"total_admins"`
		VerifiedWorkers      int64 `json:"verified_workers"`
		UnverifiedWorkers    int64 `json:"unverified_workers"`
		ActiveWorkers        int64 `json:"active_workers"`
		InactiveWorkers      int64 `json:"inactive_workers"`
		TotalServiceRequests int64 `json:"total_service_requests"`
		CompletedRequests    int64 `json:"completed_requests"`
		PendingRequests      int64 `json:"pending_requests"`
		TotalEarnings        float64 `json:"total_earnings"`
		MonthlyEarnings      float64 `json:"monthly_earnings"`
	}

	// Count users by role
	database.DB.Model(&models.User{}).Where("role = ?", models.RoleCustomer).Count(&stats.TotalCustomers)
	database.DB.Model(&models.User{}).Where("role = ?", models.RoleWorker).Count(&stats.TotalWorkers)
	database.DB.Model(&models.User{}).Where("role = ?", models.RoleAdmin).Count(&stats.TotalAdmins)
	database.DB.Model(&models.User{}).Count(&stats.TotalUsers)

	// Count workers by verification status
	database.DB.Model(&models.WorkerProfile{}).Where("is_verified = ?", true).Count(&stats.VerifiedWorkers)
	database.DB.Model(&models.WorkerProfile{}).Where("is_verified = ?", false).Count(&stats.UnverifiedWorkers)

	// Count workers by availability
	database.DB.Model(&models.WorkerProfile{}).Where("is_available = ?", true).Count(&stats.ActiveWorkers)
	database.DB.Model(&models.WorkerProfile{}).Where("is_available = ?", false).Count(&stats.InactiveWorkers)

	// Count service requests
	database.DB.Model(&models.CustomerServiceRequest{}).Count(&stats.TotalServiceRequests)
	database.DB.Model(&models.CustomerServiceRequest{}).Where("status = ?", models.RequestStatusCompleted).Count(&stats.CompletedRequests)
	database.DB.Model(&models.CustomerServiceRequest{}).Where("status IN (?)", []string{string(models.RequestStatusBroadcast), string(models.RequestStatusAccepted)}).Count(&stats.PendingRequests)

	// Calculate earnings (this would need to be implemented based on your business logic)
	// For now, we'll use placeholder values
	stats.TotalEarnings = 0.0
	stats.MonthlyEarnings = 0.0

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// GetAllServiceRequests returns all service requests with pagination and filters
func GetAllServiceRequests(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	status := c.Query("status")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}

	offset := (page - 1) * limit

	var requests []models.CustomerServiceRequest
	var total int64

	query := database.DB.Model(&models.CustomerServiceRequest{}).Preload("Customer").Preload("AssignedWorker.User").Preload("Category")
	
	// Apply status filter
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		log.Printf("❌ Failed to count service requests: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count service requests"})
		return
	}

	// Get service requests with pagination
	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&requests).Error; err != nil {
		log.Printf("❌ Failed to fetch service requests: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch service requests"})
		return
	}

	// Format response
	var requestList []gin.H
	for _, request := range requests {
		requestData := gin.H{
			"id":                request.ID,
			"customer_id":       request.CustomerID,
			"category_id":       request.CategoryID,
			"title":             request.Title,
			"description":       request.Description,
			"priority":          request.Priority,
			"budget":            request.Budget,
			"estimated_duration": request.EstimatedDuration,
			"location_address":  request.LocationAddress,
			"location_city":     request.LocationCity,
			"location_lat":      request.LocationLat,
			"location_lng":      request.LocationLng,
			"status":            request.Status,
			"assigned_worker_id": request.AssignedWorkerID,
			"started_at":        request.StartedAt,
			"completed_at":      request.CompletedAt,
			"expires_at":        request.ExpiresAt,
			"created_at":        request.CreatedAt,
			"updated_at":        request.UpdatedAt,
			"customer": gin.H{
				"id":                request.Customer.ID,
				"full_name":         request.Customer.FullName,
				"phone_number":      request.Customer.PhoneNumber,
				"profile_picture_url": request.Customer.ProfilePictureURL,
			},
			"category": gin.H{
				"id":   request.Category.ID,
				"name": request.Category.Name,
			},
		}

		if request.AssignedWorker != nil {
			requestData["assigned_worker"] = gin.H{
				"id": request.AssignedWorker.ID,
				"user": gin.H{
					"id":                request.AssignedWorker.User.ID,
					"full_name":         request.AssignedWorker.User.FullName,
					"phone_number":      request.AssignedWorker.User.PhoneNumber,
					"profile_picture_url": request.AssignedWorker.User.ProfilePictureURL,
				},
			}
		}

		requestList = append(requestList, requestData)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    requestList,
		"total":   total,
		"page":    page,
		"limit":   limit,
	})
}

// GetServiceRequestById returns service request by ID
func GetServiceRequestById(c *gin.Context) {
	requestID := c.Param("id")
	
	var request models.CustomerServiceRequest
	if err := database.DB.Preload("Customer").Preload("AssignedWorker.User").Preload("Category").First(&request, requestID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service request not found"})
		return
	}

	requestData := gin.H{
		"id":                request.ID,
		"customer_id":       request.CustomerID,
		"category_id":       request.CategoryID,
		"title":             request.Title,
		"description":       request.Description,
		"priority":          request.Priority,
		"budget":            request.Budget,
		"estimated_duration": request.EstimatedDuration,
		"location_address":  request.LocationAddress,
		"location_city":     request.LocationCity,
		"location_lat":      request.LocationLat,
		"location_lng":      request.LocationLng,
		"status":            request.Status,
		"assigned_worker_id": request.AssignedWorkerID,
		"started_at":        request.StartedAt,
		"completed_at":      request.CompletedAt,
		"expires_at":        request.ExpiresAt,
		"created_at":        request.CreatedAt,
		"updated_at":        request.UpdatedAt,
		"customer": gin.H{
			"id":                request.Customer.ID,
			"full_name":         request.Customer.FullName,
			"phone_number":      request.Customer.PhoneNumber,
			"profile_picture_url": request.Customer.ProfilePictureURL,
		},
		"category": gin.H{
			"id":   request.Category.ID,
			"name": request.Category.Name,
		},
	}

	if request.AssignedWorker != nil {
		requestData["assigned_worker"] = gin.H{
			"id": request.AssignedWorker.ID,
			"user": gin.H{
				"id":                request.AssignedWorker.User.ID,
				"full_name":         request.AssignedWorker.User.FullName,
				"phone_number":      request.AssignedWorker.User.PhoneNumber,
				"profile_picture_url": request.AssignedWorker.User.ProfilePictureURL,
			},
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    requestData,
	})
}

