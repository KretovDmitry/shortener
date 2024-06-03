// Package models contains the data models for the application.
package models

import (
	"github.com/golang-jwt/jwt/v4"
)

// Claims struct represents the custom claims structure for JWT tokens.
// It extends the RegisteredClaims from the jwt package with an additional UserID field.
//
// Fields:
//   - jwt.RegisteredClaims: Standard claims fields defined by the JWT specification.
//   - UserID string: A unique identifier for the user associated with the token.
type Claims struct {
	jwt.RegisteredClaims
	UserID string
}
