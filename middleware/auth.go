package middleware

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"repair-service-server/config"
	"repair-service-server/database"
	"repair-service-server/models"
	"repair-service-server/types"
)

// Claims represents the JWT claims (using shared types)
type Claims = types.Claims

// AuthMiddleware validates JWT tokens and sets user context
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Printf("🔍 AuthMiddleware: %s %s", c.Request.Method, c.Request.URL.Path)
		log.Printf("🔍 AuthMiddleware: Full URL: %s", c.Request.URL.String())
		
		// Get the Authorization header
		authHeader := c.GetHeader("Authorization")
		log.Printf("🔍 AuthMiddleware: Authorization header: %s", authHeader)
		
		// Log all headers for debugging
		log.Printf("🔍 AuthMiddleware: All headers: %v", c.Request.Header)
		
		if authHeader == "" {
			log.Printf("🔍 AuthMiddleware: No Authorization header")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Authorization header required",
				"message": "Please provide a valid token",
			})
			c.Abort()
			return
		}

		// Check if the header starts with "Bearer "
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Invalid token format",
				"message": "Token must be in format: Bearer <token>",
			})
			c.Abort()
			return
		}

		// Parse and validate the token
		if len(tokenString) > 20 {
			log.Printf("🔍 AuthMiddleware: Parsing token: %s...", tokenString[:20])
		} else {
			log.Printf("🔍 AuthMiddleware: Parsing token: %s", tokenString)
		}
		if len(config.AppConfig.JWT.Secret) > 10 {
			log.Printf("🔍 AuthMiddleware: Using JWT secret: %s...", config.AppConfig.JWT.Secret[:10])
		} else {
			log.Printf("🔍 AuthMiddleware: Using JWT secret: %s", config.AppConfig.JWT.Secret)
		}
		
		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(config.AppConfig.JWT.Secret), nil
		})

		if err != nil {
			log.Printf("🔍 AuthMiddleware: Token parsing error: %v", err)
			log.Printf("🔍 AuthMiddleware: Token string: %s", tokenString)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Invalid token",
				"message": "Token is invalid or expired",
			})
			c.Abort()
			return
		}

		// Extract claims
		claims, ok := token.Claims.(*Claims)
		if !ok || !token.Valid {
			log.Printf("🔍 AuthMiddleware: Token validation failed - ok: %v, valid: %v", ok, token.Valid)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Invalid token claims",
				"message": "Token claims are invalid",
			})
			c.Abort()
			return
		}

		log.Printf("🔍 AuthMiddleware: Token claims extracted - UserID: %d", claims.UserID)

		// Get user from database
		var user models.User
		if err := database.DB.First(&user, claims.UserID).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "User not found",
				"message": "User associated with token not found",
			})
			c.Abort()
			return
		}

		// Check if user is active
		if !user.IsActive {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "User inactive",
				"message": "User account is deactivated",
			})
			c.Abort()
			return
		}

		// Set user in context
		c.Set("user", user)
		c.Set("user_id", user.ID)
		
		log.Printf("🔍 AuthMiddleware: User authenticated successfully: %d", user.ID)

		c.Next()
	}
}

// OptionalAuthMiddleware is like AuthMiddleware but doesn't require authentication
func OptionalAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.Next()
			return
		}

		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(config.AppConfig.JWT.Secret), nil
		})

		if err != nil {
			c.Next()
			return
		}

		claims, ok := token.Claims.(*Claims)
		if !ok || !token.Valid {
			c.Next()
			return
		}

		var user models.User
		if err := database.DB.First(&user, claims.UserID).Error; err != nil {
			c.Next()
			return
		}

		if user.IsActive {
			c.Set("user", user)
			c.Set("user_id", user.ID)
		}

		c.Next()
	}
}

// WebSocketAuthMiddleware validates JWT tokens from query parameters for WebSocket connections
func WebSocketAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Printf("🔌 WebSocketAuthMiddleware: %s %s", c.Request.Method, c.Request.URL.Path)
		
		// Get token from query parameters for WebSocket connections
		tokenString := c.Query("token")
		if tokenString == "" {
			log.Printf("🔌 WebSocketAuthMiddleware: No token in query parameters")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Token required",
				"message": "Please provide a valid token in query parameters",
			})
			c.Abort()
			return
		}

		// Parse and validate the token
		log.Printf("🔌 WebSocketAuthMiddleware: Parsing token: %s...", tokenString[:20])
		
		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(config.AppConfig.JWT.Secret), nil
		})

		if err != nil {
			log.Printf("🔌 WebSocketAuthMiddleware: Token parsing error: %v", err)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Invalid token",
				"message": "Token is invalid or expired",
			})
			c.Abort()
			return
		}

		// Extract claims
		claims, ok := token.Claims.(*Claims)
		if !ok || !token.Valid {
			log.Printf("🔌 WebSocketAuthMiddleware: Token validation failed - ok: %v, valid: %v", ok, token.Valid)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Invalid token claims",
				"message": "Token claims are invalid",
			})
			c.Abort()
			return
		}

		log.Printf("🔌 WebSocketAuthMiddleware: Token claims extracted - UserID: %d", claims.UserID)

		// Get user from database
		var user models.User
		if err := database.DB.First(&user, claims.UserID).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "User not found",
				"message": "User associated with token not found",
			})
			c.Abort()
			return
		}

		// Check if user is active
		if !user.IsActive {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "User inactive",
				"message": "User account is deactivated",
			})
			c.Abort()
			return
		}

		// Set user in context
		c.Set("user", user)
		c.Set("user_id", user.ID)
		
		log.Printf("🔌 WebSocketAuthMiddleware: User authenticated successfully: %d", user.ID)

		c.Next()
	}
}

