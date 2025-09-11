package services

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"repair-service-server/config"
	"repair-service-server/database"
	"repair-service-server/models"
	"repair-service-server/types"
)

// JWTService handles JWT token operations
type JWTService struct{}

// NewJWTService creates a new JWT service
func NewJWTService() *JWTService {
	return &JWTService{}
}

// TokenPair represents a pair of access and refresh tokens
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// GenerateTokenPair generates both access and refresh tokens
func (js *JWTService) GenerateTokenPair(userID uint, deviceID, userAgent, ipAddress string) (*TokenPair, error) {
	// Generate access token (short-lived)
	accessToken, expiresIn, err := js.generateAccessToken(userID)
	if err != nil {
		return nil, err
	}

	// Generate refresh token (long-lived)
	refreshToken, err := js.generateRefreshToken(userID, deviceID, userAgent, ipAddress)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
		TokenType:    "Bearer",
	}, nil
}

// generateAccessToken generates a short-lived access token
func (js *JWTService) generateAccessToken(userID uint) (string, int64, error) {
	// Create claims
	claims := &types.Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(config.AppConfig.JWT.ExpiryHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "repair-service-server",
			Subject:   string(rune(userID)),
		},
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token with secret
	tokenString, err := token.SignedString([]byte(config.AppConfig.JWT.Secret))
	if err != nil {
		return "", 0, err
	}

	expiresIn := int64(config.AppConfig.JWT.ExpiryHours * 3600) // Convert to seconds
	return tokenString, expiresIn, nil
}

// generateRefreshToken generates a long-lived refresh token
func (js *JWTService) generateRefreshToken(userID uint, deviceID, userAgent, ipAddress string) (string, error) {
	// Generate a secure random token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", err
	}
	tokenString := hex.EncodeToString(tokenBytes)

	// Create refresh token record
	refreshToken := &models.RefreshToken{
		Token:     tokenString,
		UserID:    userID,
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour), // 30 days
		DeviceID:  deviceID,
		UserAgent: userAgent,
		IPAddress: ipAddress,
	}

	// Save to database
	if err := database.DB.Create(refreshToken).Error; err != nil {
		return "", err
	}

	log.Printf("✅ Refresh token generated for user %d", userID)
	return tokenString, nil
}

// ValidateAccessToken validates an access token
func (js *JWTService) ValidateAccessToken(tokenString string) (uint, error) {
	// Parse token
	token, err := jwt.ParseWithClaims(tokenString, &types.Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(config.AppConfig.JWT.Secret), nil
	})

	if err != nil {
		return 0, err
	}

	// Extract claims
	claims, ok := token.Claims.(*types.Claims)
	if !ok || !token.Valid {
		return 0, errors.New("invalid token claims")
	}

	return claims.UserID, nil
}

// ValidateRefreshToken validates a refresh token
func (js *JWTService) ValidateRefreshToken(tokenString string) (*models.RefreshToken, error) {
	var refreshToken models.RefreshToken
	
	// Find refresh token in database
	if err := database.DB.Where("token = ?", tokenString).First(&refreshToken).Error; err != nil {
		return nil, errors.New("refresh token not found")
	}

	// Check if token is valid
	if !refreshToken.IsValid() {
		return nil, errors.New("refresh token is invalid or expired")
	}

	return &refreshToken, nil
}

// RefreshAccessToken generates a new access token using a refresh token
func (js *JWTService) RefreshAccessToken(refreshTokenString string) (*TokenPair, error) {
	// Validate refresh token
	refreshToken, err := js.ValidateRefreshToken(refreshTokenString)
	if err != nil {
		return nil, err
	}

	// Generate new access token
	accessToken, expiresIn, err := js.generateAccessToken(refreshToken.UserID)
	if err != nil {
		return nil, err
	}

	// Update refresh token's last used time
	refreshToken.UpdatedAt = time.Now()
	database.DB.Save(refreshToken)

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshTokenString, // Keep the same refresh token
		ExpiresIn:    expiresIn,
		TokenType:    "Bearer",
	}, nil
}

// RevokeRefreshToken revokes a refresh token
func (js *JWTService) RevokeRefreshToken(tokenString string) error {
	var refreshToken models.RefreshToken
	
	// Find refresh token
	if err := database.DB.Where("token = ?", tokenString).First(&refreshToken).Error; err != nil {
		return errors.New("refresh token not found")
	}

	// Revoke token
	refreshToken.Revoke()
	database.DB.Save(&refreshToken)

	log.Printf("✅ Refresh token revoked for user %d", refreshToken.UserID)
	return nil
}

// RevokeAllUserTokens revokes all refresh tokens for a user
func (js *JWTService) RevokeAllUserTokens(userID uint) error {
	// Revoke all tokens for user
	if err := database.DB.Model(&models.RefreshToken{}).
		Where("user_id = ? AND is_revoked = ?", userID, false).
		Update("is_revoked", true).Error; err != nil {
		return err
	}

	log.Printf("✅ All refresh tokens revoked for user %d", userID)
	return nil
}

// CleanupExpiredTokens removes expired refresh tokens
func (js *JWTService) CleanupExpiredTokens() error {
	// Delete expired tokens
	if err := database.DB.Where("expires_at < ?", time.Now()).Delete(&models.RefreshToken{}).Error; err != nil {
		return err
	}

	log.Printf("✅ Expired refresh tokens cleaned up")
	return nil
}

// HashPassword hashes a password using bcrypt
func (js *JWTService) HashPassword(password string) (string, error) {
	// Use higher cost for better security
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(bytes), err
}

// CheckPasswordHash compares a password with its hash
func (js *JWTService) CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateSecureSecret generates a cryptographically secure JWT secret
func (js *JWTService) GenerateSecureSecret() (string, error) {
	bytes := make([]byte, 64) // 512 bits
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
