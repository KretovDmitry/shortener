package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/KretovDmitry/shortener/internal/models"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Claims struct {
	jwt.RegisteredClaims
	UserID string
}

// BuildJWTString creates a JWT string for the given user ID and token expiration time.
func BuildJWTString(userID, secret string, tokenExp time.Duration) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
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
	claims := &Claims{}

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

// DumbRegistration checks if the user has a valid JWT token,
// if not sets a new one with a new user ID.
func DumbRegistration(
	logger *zap.Logger, secret string, tokenExp time.Duration,
) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// If neither authorization cookie found, nor valid token is provided
			// create a new authorization cookie with the new token
			// containing new user ID.
			authCookie, err := r.Cookie("Authorization")
			if err == nil {
				if id, err := GetUserID(authCookie.Value, secret); err == nil {
					logger.Info("JWT token contains user ID", zap.String("id", id))
					ctx := context.WithValue(r.Context(), models.UserIDCtxKey{}, id)
					next(w, r.WithContext(ctx))
					return
				}
			}

			if err != http.ErrNoCookie {
				logger.Error("error parsing cookie", zap.Error(err))
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			logger.Info("Invalid JWT token", zap.Error(err))

			id := uuid.NewString()

			JWTtoken, err := BuildJWTString(id, secret, tokenExp)
			if err != nil {
				logger.Error("error building JWT", zap.Error(err))
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			http.SetCookie(w, &http.Cookie{
				Name:     "Authorization",
				Value:    JWTtoken,
				Expires:  time.Now().Add(tokenExp),
				HttpOnly: true,
			})

			logger.Info("New user", zap.String("id", id))

			ctx := context.WithValue(r.Context(), models.UserIDCtxKey{}, id)

			next(w, r.WithContext(ctx))
		}
	}
}

func DumbAuthorization(logger *zap.Logger, secret string, tokenExp time.Duration) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			authCookie, err := r.Cookie("Authorization")
			if err != nil {
				if err == http.ErrNoCookie {
					http.Error(w, "Authorization cookie not found", http.StatusUnauthorized)
					logger.Info("Authorization cookie not found")
					return
				}
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			id, err := GetUserID(authCookie.Value, secret)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			logger.Info("JWT token contains user ID", zap.String("id", id))
			ctx := context.WithValue(r.Context(), models.UserIDCtxKey{}, id)

			next(w, r.WithContext(ctx))
		}
	}
}