package types

import "github.com/golang-jwt/jwt/v5"

// Claims represents the JWT claims
type Claims struct {
	UserID uint `json:"user_id"`
	jwt.RegisteredClaims
}
