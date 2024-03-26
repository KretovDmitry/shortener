package jwt

import (
	"fmt"
	"strings"
	"time"

	"github.com/KretovDmitry/shortener/internal/models"
	"github.com/golang-jwt/jwt/v4"
)

// BuildJWTString creates a JWT string for the given user ID and token expiration time.
func BuildJWTString(userID, secret string, tokenExp time.Duration) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, models.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenExp)),
		},
		UserID: userID,
	})

	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Bearer %s", tokenString), nil
}

// GetUserID extracts the user ID from a JWT token.
func GetUserID(tokenString, secret string) (string, error) {
	claims := new(models.Claims)

	tokenString = strings.TrimPrefix(tokenString, "Bearer ")

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Verify that the token method is HS256
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Return the secret key
		return []byte(secret), nil
	})

	// Check for errors
	if err != nil {
		return "", fmt.Errorf("error parsing token: %w", err)
	}

	// Check if the token is valid
	if !token.Valid {
		return "", fmt.Errorf("invalid token: %w", err)
	}

	// Return the user ID
	return claims.UserID, nil
}
