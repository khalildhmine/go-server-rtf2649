package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiter stores rate limiters for different IPs
type RateLimiter struct {
	limiters map[string]*rate.Limiter
	lastSeen map[string]time.Time
	mutex    sync.RWMutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		lastSeen: make(map[string]time.Time),
	}
}

// GetLimiter returns a rate limiter for the given IP
func (rl *RateLimiter) GetLimiter(ip string) *rate.Limiter {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	limiter, exists := rl.limiters[ip]
	if !exists {
		// Allow 10 requests per minute, burst of 20
		limiter = rate.NewLimiter(rate.Every(time.Minute/10), 20)
		rl.limiters[ip] = limiter
	}
	// Update last seen time for this IP
	rl.lastSeen[ip] = time.Now()

	return limiter
}

// Cleanup removes old limiters to prevent memory leaks
func (rl *RateLimiter) Cleanup() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	// Remove limiters that have been idle for more than 1 hour
	now := time.Now()
	for ip, t := range rl.lastSeen {
		if now.Sub(t) > time.Hour {
			delete(rl.limiters, ip)
			delete(rl.lastSeen, ip)
		}
	}
}

var globalRateLimiter = NewRateLimiter()

// RateLimitMiddleware implements rate limiting
func RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.FullPath()
		clientIP := c.ClientIP()
		key := path + "|" + clientIP

		// Relax limits for WebSocket endpoint and worker reads to avoid 429
		var lim rate.Limit
		var burst int
		if strings.HasPrefix(path, "/api/v1/chat/ws") {
			// WebSocket upgrade - allow higher burst
			lim = rate.Every(time.Second) // ~1 req/sec
			burst = 5
		} else if c.Request.Method == http.MethodGet && strings.HasPrefix(path, "/api/v1/worker") {
			// Worker polling/reads
			lim = rate.Every(time.Second)
			burst = 3
		} else if strings.HasPrefix(path, "/api/v1/location") {
			// Location updates can be frequent - moderate limits
			lim = rate.Every(2 * time.Second)
			burst = 2
		} else {
			// Default limits
			lim = rate.Every(time.Minute / 10) // 10 req/min
			burst = 20
		}

		limiter := globalRateLimiter.GetLimiterWithConfig(key, lim, burst)

		if !limiter.Allow() {
			log.Printf("🚫 Rate limit exceeded for %s %s from %s", c.Request.Method, path, clientIP)
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Rate limit exceeded",
				"message": "Too many requests. Please try again later.",
				"retry_after": 60,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// AuthRateLimitMiddleware implements stricter rate limiting for auth endpoints
func AuthRateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		
		// Create a stricter limiter for auth endpoints
		limiter := rate.NewLimiter(rate.Every(time.Minute/5), 5) // 5 requests per minute, burst of 5

		if !limiter.Allow() {
			log.Printf("🚫 Auth rate limit exceeded for IP: %s", clientIP)
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Authentication rate limit exceeded",
				"message": "Too many authentication attempts. Please try again later.",
				"retry_after": 300,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// SecurityHeadersMiddleware adds security headers
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent XSS attacks
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		
		// Content Security Policy
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; connect-src 'self' ws: wss:;")
		
		// HSTS (HTTP Strict Transport Security)
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		
		// Remove server information
		c.Header("Server", "")
		
		c.Next()
	}
}

// CORSMiddleware implements secure CORS
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		
		// Define allowed origins (in production, use environment variables)
		allowedOrigins := []string{
			"http://localhost:3000",
			"http://localhost:8081",
			"exp://192.168.100.5:8081",
			// Add your production domains here
		}
		
		// Check if origin is allowed
		allowed := false
		for _, allowedOrigin := range allowedOrigins {
			if origin == allowedOrigin {
				allowed = true
				break
			}
		}
		
		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
		}
		
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Length, Content-Type, Authorization, Accept, User-Agent, X-Requested-With")
		c.Header("Access-Control-Expose-Headers", "Content-Length, Content-Type")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400")
		
		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		
		c.Next()
	}
}

// InputValidationMiddleware validates and sanitizes input
func InputValidationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Validate request size
		if c.Request.ContentLength > 10*1024*1024 { // 10MB limit
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error":   "Request too large",
				"message": "Request body exceeds maximum size limit",
			})
			c.Abort()
			return
		}
		
		// Validate content type for POST/PUT requests
		if c.Request.Method == "POST" || c.Request.Method == "PUT" {
			contentType := c.GetHeader("Content-Type")
			if !strings.Contains(contentType, "application/json") && 
			   !strings.Contains(contentType, "multipart/form-data") &&
			   !strings.Contains(contentType, "application/x-www-form-urlencoded") {
				c.JSON(http.StatusUnsupportedMediaType, gin.H{
					"error":   "Invalid content type",
					"message": "Content-Type must be application/json, multipart/form-data, or application/x-www-form-urlencoded",
				})
				c.Abort()
				return
			}
		}
		
		c.Next()
	}
}

// AuditLogMiddleware logs security events
func AuditLogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		
		// Log the request
		log.Printf("🔍 AUDIT: %s %s from %s", c.Request.Method, c.Request.URL.Path, c.ClientIP())
		
		c.Next()
		
		// Log the response
		duration := time.Since(start)
		status := c.Writer.Status()
		
		if status >= 400 {
			log.Printf("⚠️ AUDIT: %s %s returned %d in %v", c.Request.Method, c.Request.URL.Path, status, duration)
		} else {
			log.Printf("✅ AUDIT: %s %s returned %d in %v", c.Request.Method, c.Request.URL.Path, status, duration)
		}
	}
}

// GenerateSecureToken generates a cryptographically secure random token
func GenerateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// ValidatePhoneNumber validates phone number format
func ValidatePhoneNumber(phoneNumber string) bool {
	// Remove all non-digit characters except +
	cleaned := strings.ReplaceAll(phoneNumber, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	cleaned = strings.ReplaceAll(cleaned, "(", "")
	cleaned = strings.ReplaceAll(cleaned, ")", "")
	
	// Check if it starts with +222 and has 8-11 digits after
	if !strings.HasPrefix(cleaned, "+222") {
		return false
	}
	
	// Extract the number part after +222
	numberPart := cleaned[4:]
	
	// Check if it's all digits and has correct length
	if len(numberPart) < 8 || len(numberPart) > 11 {
		return false
	}
	
	for _, char := range numberPart {
		if char < '0' || char > '9' {
			return false
		}
	}
	
	return true
}

// SanitizeInput sanitizes user input to prevent injection attacks
func SanitizeInput(input string) string {
	// Remove potentially dangerous characters
	input = strings.ReplaceAll(input, "<", "&lt;")
	input = strings.ReplaceAll(input, ">", "&gt;")
	input = strings.ReplaceAll(input, "\"", "&quot;")
	input = strings.ReplaceAll(input, "'", "&#x27;")
	input = strings.ReplaceAll(input, "&", "&amp;")
	
	// Trim whitespace
	input = strings.TrimSpace(input)
	
	return input
}

// ValidatePasswordStrength validates password strength
func ValidatePasswordStrength(password string) (bool, []string) {
	var errors []string
	
	if len(password) < 8 {
		errors = append(errors, "Password must be at least 8 characters long")
	}
	
	if len(password) > 128 {
		errors = append(errors, "Password must be less than 128 characters")
	}
	
	hasUpper := false
	hasLower := false
	hasDigit := false
	hasSpecial := false
	
	for _, char := range password {
		switch {
		case char >= 'A' && char <= 'Z':
			hasUpper = true
		case char >= 'a' && char <= 'z':
			hasLower = true
		case char >= '0' && char <= '9':
			hasDigit = true
		case strings.ContainsRune("!@#$%^&*()_+-=[]{}|;:,.<>?", char):
			hasSpecial = true
		}
	}
	
	if !hasUpper {
		errors = append(errors, "Password must contain at least one uppercase letter")
	}
	if !hasLower {
		errors = append(errors, "Password must contain at least one lowercase letter")
	}
	if !hasDigit {
		errors = append(errors, "Password must contain at least one digit")
	}
	if !hasSpecial {
		errors = append(errors, "Password must contain at least one special character")
	}
	
	return len(errors) == 0, errors
}

// GetLimiterWithConfig returns a limiter for a composite key with dynamic limits
func (rl *RateLimiter) GetLimiterWithConfig(key string, limit rate.Limit, burst int) *rate.Limiter {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	limiter, exists := rl.limiters[key]
	if !exists {
		limiter = rate.NewLimiter(limit, burst)
		rl.limiters[key] = limiter
	}
	rl.lastSeen[key] = time.Now()
	return limiter
}
