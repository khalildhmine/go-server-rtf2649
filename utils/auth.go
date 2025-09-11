package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"repair-service-server/config"
	"repair-service-server/types"
)

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPasswordHash compares a password with its hash
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateToken generates a JWT token for a user
func GenerateToken(userID uint, role string) (string, error) {
	// Create claims
	claims := &types.Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(config.AppConfig.JWT.ExpiryHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token with secret
	tokenString, err := token.SignedString([]byte(config.AppConfig.JWT.Secret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// GenerateRefreshToken generates a refresh token for a user
func GenerateRefreshToken(userID uint) (string, error) {
	// Create claims for refresh token (longer expiry)
	claims := &types.Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * 24 * time.Hour)), // 30 days
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token with secret
	tokenString, err := token.SignedString([]byte(config.AppConfig.JWT.Secret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// VerifyToken verifies a JWT token and returns the claims
func VerifyToken(tokenString string) (*types.Claims, error) {
	// Parse token
	token, err := jwt.ParseWithClaims(tokenString, &types.Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.AppConfig.JWT.Secret), nil
	})

	if err != nil {
		return nil, err
	}

	// Extract claims
	claims, ok := token.Claims.(*types.Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}

// VerifyRefreshToken verifies a refresh token and returns the claims
func VerifyRefreshToken(tokenString string) (*types.Claims, error) {
	return VerifyToken(tokenString)
}

// ValidateToken validates a JWT token and returns the user ID
func ValidateToken(tokenString string) (uint, error) {
	// Parse token
	token, err := jwt.ParseWithClaims(tokenString, &types.Claims{}, func(token *jwt.Token) (interface{}, error) {
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

// ValidatePhoneNumber validates phone number format with country code
func ValidatePhoneNumber(phoneNumber string) bool {
	// Basic validation for +222 format
	if len(phoneNumber) < 10 || len(phoneNumber) > 15 {
		return false
	}

	// Check if it starts with +222
	if len(phoneNumber) >= 4 && phoneNumber[:4] == "+222" {
		return true
	}

	return false
}

// FormatPhoneNumber formats phone number to include country code if not present
func FormatPhoneNumber(phoneNumber string) string {
	if len(phoneNumber) >= 4 && phoneNumber[:4] == "+222" {
		return phoneNumber
	}

	// Remove any existing + if present
	if len(phoneNumber) > 0 && phoneNumber[0] == '+' {
		phoneNumber = phoneNumber[1:]
	}

	// Add +222 prefix
	return "+222" + phoneNumber
}