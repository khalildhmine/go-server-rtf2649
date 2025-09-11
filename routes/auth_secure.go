package routes

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"repair-service-server/database"
	"repair-service-server/middleware"
	"repair-service-server/models"
	"repair-service-server/services"
)

// RegisterSecureAuthRoutes registers secure authentication routes
func RegisterSecureAuthRoutes(router *gin.RouterGroup) {
	jwtService := services.NewJWTService()

	// Sign up endpoint
	router.POST("/signup", func(c *gin.Context) {
		var req struct {
			FullName         string `json:"full_name" binding:"required,min=2,max=100"`
			PhoneNumber      string `json:"phone_number" binding:"required"`
			Password         string `json:"password" binding:"required,min=8,max=128"`
			ConfirmPassword  string `json:"confirm_password" binding:"required"`
			Role             string `json:"role" binding:"omitempty,oneof=customer worker"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid request",
				"message": err.Error(),
			})
			return
		}

		// Sanitize input
		req.FullName = middleware.SanitizeInput(req.FullName)
		req.PhoneNumber = strings.TrimSpace(req.PhoneNumber)

		// Validate phone number
		if !middleware.ValidatePhoneNumber(req.PhoneNumber) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid phone number",
				"message": "Phone number must be in format +222XXXXXXXX",
			})
			return
		}

		// Validate password strength
		isStrong, errors := middleware.ValidatePasswordStrength(req.Password)
		if !isStrong {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Weak password",
				"message": "Password does not meet security requirements",
				"details": errors,
			})
			return
		}

		// Check password confirmation
		if req.Password != req.ConfirmPassword {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Password mismatch",
				"message": "Passwords do not match",
			})
			return
		}

		// Check if user already exists
		var existingUser models.User
		if err := database.DB.Where("phone_number = ?", req.PhoneNumber).First(&existingUser).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{
				"error":   "User already exists",
				"message": "An account with this phone number already exists",
			})
			return
		}

		// Hash password
		hashedPassword, err := jwtService.HashPassword(req.Password)
		if err != nil {
			log.Printf("❌ Password hashing failed: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Internal server error",
				"message": "Failed to process password",
			})
			return
		}

		// Determine user role
		userRole := models.RoleCustomer
		if strings.ToLower(req.Role) == "worker" {
			userRole = models.RoleWorker
		}

		// Create user
		user := models.User{
			FullName:     req.FullName,
			PhoneNumber:  req.PhoneNumber,
			PasswordHash: hashedPassword,
			Role:         userRole,
			IsActive:     true,
		}

		if err := database.DB.Create(&user).Error; err != nil {
			log.Printf("❌ User creation failed: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Internal server error",
				"message": "Failed to create account",
			})
			return
		}

		// Worker profile creation is now manual - user must create it themselves

		// Generate tokens
		deviceID := c.GetHeader("X-Device-ID")
		userAgent := c.GetHeader("User-Agent")
		ipAddress := c.ClientIP()

		tokenPair, err := jwtService.GenerateTokenPair(user.ID, deviceID, userAgent, ipAddress)
		if err != nil {
			log.Printf("❌ Token generation failed: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Internal server error",
				"message": "Failed to generate authentication tokens",
			})
			return
		}

		log.Printf("✅ User created successfully: %d", user.ID)

		c.JSON(http.StatusCreated, gin.H{
			"success": true,
			"message": "Account created successfully",
			"data": gin.H{
				"user": gin.H{
					"id":           user.ID,
					"full_name":    user.FullName,
					"phone_number": user.PhoneNumber,
					"role":         user.Role,
					"is_active":    user.IsActive,
					"created_at":   user.CreatedAt,
				},
				"tokens": tokenPair,
			},
		})
	})

	// Sign in endpoint
	router.POST("/signin", func(c *gin.Context) {
		var req struct {
			PhoneNumber string `json:"phone_number" binding:"required"`
			Password    string `json:"password" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid request",
				"message": err.Error(),
			})
			return
		}

		// Sanitize input
		req.PhoneNumber = strings.TrimSpace(req.PhoneNumber)

		// Validate phone number
		if !middleware.ValidatePhoneNumber(req.PhoneNumber) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid phone number",
				"message": "Phone number must be in format +222XXXXXXXX",
			})
			return
		}

		// Find user
		var user models.User
		if err := database.DB.Where("phone_number = ?", req.PhoneNumber).First(&user).Error; err != nil {
			log.Printf("❌ User not found: %s", req.PhoneNumber)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Invalid credentials",
				"message": "Phone number or password is incorrect",
			})
			return
		}

		// Check if user is active
		if !user.IsActive {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Account deactivated",
				"message": "Your account has been deactivated",
			})
			return
		}

		// Verify password
		if !jwtService.CheckPasswordHash(req.Password, user.PasswordHash) {
			log.Printf("❌ Invalid password for user: %d", user.ID)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Invalid credentials",
				"message": "Phone number or password is incorrect",
			})
			return
		}

		// Revoke all existing tokens for security
		if err := jwtService.RevokeAllUserTokens(user.ID); err != nil {
			log.Printf("⚠️ Failed to revoke existing tokens for user %d: %v", user.ID, err)
		}

		// Generate new tokens
		deviceID := c.GetHeader("X-Device-ID")
		userAgent := c.GetHeader("User-Agent")
		ipAddress := c.ClientIP()

		tokenPair, err := jwtService.GenerateTokenPair(user.ID, deviceID, userAgent, ipAddress)
		if err != nil {
			log.Printf("❌ Token generation failed: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Internal server error",
				"message": "Failed to generate authentication tokens",
			})
			return
		}

		log.Printf("✅ User signed in successfully: %d", user.ID)

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Sign in successful",
			"data": gin.H{
				"user": gin.H{
					"id":           user.ID,
					"full_name":    user.FullName,
					"phone_number": user.PhoneNumber,
					"role":         user.Role,
					"is_active":    user.IsActive,
					"created_at":   user.CreatedAt,
				},
				"tokens": tokenPair,
			},
		})
	})

	// Refresh token endpoint
	router.POST("/refresh", func(c *gin.Context) {
		var req struct {
			RefreshToken string `json:"refresh_token" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid request",
				"message": err.Error(),
			})
			return
		}

		// Refresh access token
		tokenPair, err := jwtService.RefreshAccessToken(req.RefreshToken)
		if err != nil {
			log.Printf("❌ Token refresh failed: %v", err)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Invalid refresh token",
				"message": "Refresh token is invalid or expired",
			})
			return
		}

		log.Printf("✅ Token refreshed successfully")

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Token refreshed successfully",
			"data": gin.H{
				"tokens": tokenPair,
			},
		})
	})

	// Sign out endpoint
	router.POST("/signout", middleware.AuthMiddleware(), func(c *gin.Context) {
		userID := c.GetUint("user_id")
		
		// Get refresh token from request
		var req struct {
			RefreshToken string `json:"refresh_token"`
		}
		
		if err := c.ShouldBindJSON(&req); err == nil && req.RefreshToken != "" {
			// Revoke specific refresh token
			if err := jwtService.RevokeRefreshToken(req.RefreshToken); err != nil {
				log.Printf("⚠️ Failed to revoke refresh token: %v", err)
			}
		} else {
			// Revoke all tokens for user
			if err := jwtService.RevokeAllUserTokens(userID); err != nil {
				log.Printf("⚠️ Failed to revoke all tokens for user %d: %v", userID, err)
			}
		}

		log.Printf("✅ User signed out: %d", userID)

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Sign out successful",
		})
	})

	// Get current user endpoint
	router.GET("/me", middleware.AuthMiddleware(), func(c *gin.Context) {
		userID := c.GetUint("user_id")
		
		var user models.User
		if err := database.DB.First(&user, userID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "User not found",
				"message": "User not found",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"user": gin.H{
					"id":           user.ID,
					"full_name":    user.FullName,
					"phone_number": user.PhoneNumber,
					"role":         user.Role,
					"is_active":    user.IsActive,
					"created_at":   user.CreatedAt,
					"updated_at":   user.UpdatedAt,
				},
			},
		})
	})

	// Change password endpoint
	router.POST("/change-password", middleware.AuthMiddleware(), func(c *gin.Context) {
		userID := c.GetUint("user_id")
		
		var req struct {
			CurrentPassword string `json:"current_password" binding:"required"`
			NewPassword     string `json:"new_password" binding:"required,min=8,max=128"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid request",
				"message": err.Error(),
			})
			return
		}

		// Get user
		var user models.User
		if err := database.DB.First(&user, userID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "User not found",
				"message": "User not found",
			})
			return
		}

		// Verify current password
		if !jwtService.CheckPasswordHash(req.CurrentPassword, user.PasswordHash) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Invalid current password",
				"message": "Current password is incorrect",
			})
			return
		}

		// Validate new password strength
		isStrong, errors := middleware.ValidatePasswordStrength(req.NewPassword)
		if !isStrong {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Weak password",
				"message": "New password does not meet security requirements",
				"details": errors,
			})
			return
		}

		// Hash new password
		hashedPassword, err := jwtService.HashPassword(req.NewPassword)
		if err != nil {
			log.Printf("❌ Password hashing failed: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Internal server error",
				"message": "Failed to process new password",
			})
			return
		}

		// Update password
		user.PasswordHash = hashedPassword
		if err := database.DB.Save(&user).Error; err != nil {
			log.Printf("❌ Password update failed: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Internal server error",
				"message": "Failed to update password",
			})
			return
		}

		// Revoke all tokens for security
		if err := jwtService.RevokeAllUserTokens(userID); err != nil {
			log.Printf("⚠️ Failed to revoke tokens after password change: %v", err)
		}

		log.Printf("✅ Password changed for user: %d", userID)

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Password changed successfully. Please sign in again.",
		})
	})
}
