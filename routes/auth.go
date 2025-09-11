package routes

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"repair-service-server/database"
	"repair-service-server/models"
	"repair-service-server/utils"
)

// AuthRequest represents the authentication request
type AuthRequest struct {
	PhoneNumber string `json:"phone_number" binding:"required"`
	Password    string `json:"password" binding:"required,min=6"`
	FullName    string `json:"full_name" binding:"required"`
}

// SignInRequest represents the sign in request
type SignInRequest struct {
	PhoneNumber string `json:"phone_number" binding:"required"`
	Password    string `json:"password" binding:"required"`
}

// AuthResponse represents the authentication response
type AuthResponse struct {
	Token        string      `json:"token"`
	RefreshToken string      `json:"refresh_token"`
	ExpiresIn    int64       `json:"expires_in"`
	User         models.User `json:"user"`
	WorkerProfile *models.WorkerProfile `json:"worker_profile,omitempty"`
	RedirectTo   string      `json:"redirect_to,omitempty"`
}

// RegisterAuthRoutes registers authentication routes
func RegisterAuthRoutes(router *gin.RouterGroup) {
	router.POST("/signup", signUp)
	router.POST("/signin", signIn)
	router.POST("/register", signUp)  // Alias for signup
	router.POST("/login", signIn)     // Alias for signin
	router.POST("/refresh", refreshToken) // Token refresh endpoint
	router.POST("/logout", logout)    // Logout endpoint
}

// signUp handles user registration
func signUp(c *gin.Context) {
	var req AuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"message": err.Error(),
		})
		return
	}

	// Format phone number
	phoneNumber := utils.FormatPhoneNumber(req.PhoneNumber)

	// Validate phone number
	if !utils.ValidatePhoneNumber(phoneNumber) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid phone number",
			"message": "Phone number must be in +222 format",
		})
		return
	}

	// Check if user already exists
	var existingUser models.User
	if err := database.DB.Where("phone_number = ?", phoneNumber).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"error":   "User already exists",
			"message": "A user with this phone number already exists",
		})
		return
	}

	// Hash password
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Password hashing failed",
			"message": "Failed to process password",
		})
		return
	}

	// Create user
	user := models.User{
		FullName:     req.FullName,
		PhoneNumber:  phoneNumber,
		PasswordHash: hashedPassword,
		Role:         models.RoleCustomer,
		IsActive:     true,
	}

	if err := database.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "User creation failed",
			"message": "Failed to create user account",
		})
		return
	}

	// Generate token
	token, err := utils.GenerateToken(user.ID, string(user.Role))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Token generation failed",
			"message": "Failed to generate authentication token",
		})
		return
	}

	// Determine redirect based on user role
	var redirectTo string
	if user.Role == models.RoleWorker {
		redirectTo = "worker-setup" // New worker needs to set up profile
	} else {
		redirectTo = "customer"
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "User registered successfully",
		"token": token,
		"refresh_token": token, // For now, use same token as refresh token
		"expires_in": 24 * 60 * 60, // 24 hours in seconds
		"user": user,
		"redirect_to": redirectTo,
	})
}

// signIn handles user authentication
func signIn(c *gin.Context) {
	var req SignInRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"message": err.Error(),
		})
		return
	}

	// Format phone number
	phoneNumber := utils.FormatPhoneNumber(req.PhoneNumber)

	// Validate phone number
	if !utils.ValidatePhoneNumber(phoneNumber) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid phone number",
			"message": "Phone number must be in +222 format",
		})
		return
	}

	// Find user by phone number
	var user models.User
	if err := database.DB.Where("phone_number = ?", phoneNumber).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Authentication failed",
			"message": "Invalid phone number or password",
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
	if !utils.CheckPasswordHash(req.Password, user.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Authentication failed",
			"message": "Invalid phone number or password",
		})
		return
	}

	// Generate token
	token, err := utils.GenerateToken(user.ID, string(user.Role))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Token generation failed",
			"message": "Failed to generate authentication token",
		})
		return
	}

	// Check if user is a worker and has a profile
	var workerProfile *models.WorkerProfile
	var redirectTo string

	if user.Role == models.RoleWorker {
		var profile models.WorkerProfile
		if err := database.DB.Where("user_id = ?", user.ID).First(&profile).Error; err == nil {
			// Worker has profile, redirect to worker dashboard
			workerProfile = &profile
			redirectTo = "worker"
		} else {
			// Worker doesn't have profile, redirect to profile setup
			redirectTo = "worker-setup"
		}
	} else {
		// Customer, redirect to customer dashboard
		redirectTo = "customer"
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Authentication successful",
		"token": token,
		"refresh_token": token, // For now, use same token as refresh token
		"expires_in": 24 * 60 * 60, // 24 hours in seconds
		"user": user,
		"worker_profile": workerProfile,
		"redirect_to": redirectTo,
	})
}

// GetCurrentUser returns the current authenticated user's profile
func GetCurrentUser(c *gin.Context) {
	// Get user from context (set by AuthMiddleware)
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "User not authenticated",
			"message": "Please log in to access your profile",
		})
		return
	}

	// Cast user to models.User
	userModel, ok := user.(models.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Invalid user data",
			"message": "Failed to retrieve user information",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User profile retrieved successfully",
		"data": userModel,
	})
}

// refreshToken handles token refresh
func refreshToken(c *gin.Context) {
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

	log.Printf("🔄 Token refresh request received for token: %s...", req.RefreshToken[:20])

	// Validate refresh token (in production, this should be a separate refresh token)
	// For now, we'll treat it as a regular token and validate it
	userID, err := utils.ValidateToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Invalid refresh token",
			"message": "Refresh token is invalid or expired",
		})
		return
	}

	// Get user from database
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "User not found",
			"message": "User associated with refresh token not found",
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

	// Generate new token
	newToken, err := utils.GenerateToken(user.ID, string(user.Role))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Token generation failed",
			"message": "Failed to generate new authentication token",
		})
		return
	}

	log.Printf("✅ New token generated for user %d: %s...", user.ID, newToken[:20])

	c.JSON(http.StatusOK, gin.H{
		"message": "Token refreshed successfully",
		"token": newToken,
		"refresh_token": newToken, // For now, use same token
		"expires_in": 24 * 60 * 60, // 24 hours in seconds
		"user": user,
	})
}

// logout handles user logout
func logout(c *gin.Context) {
	// In a real application, you might want to:
	// 1. Add the token to a blacklist
	// 2. Clear any server-side sessions
	// 3. Log the logout event
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Logout successful",
		"success": true,
	})
}