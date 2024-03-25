package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Claims struct {
	jwt.RegisteredClaims
	UserID string
}

const TOKEN_EXP = time.Hour * 3
const SECRET_KEY = "supersecretkey"

// BuildJWTString создаёт токен и возвращает его в виде строки.
func BuildJWTString(userID string, tokenExp time.Duration) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenExp)),
		},
		UserID: userID,
	})

	tokenString, err := token.SignedString([]byte(SECRET_KEY))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
func GetUserID(tokenString string) (string, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(SECRET_KEY), nil
		})
	if err != nil {
		return "", fmt.Errorf("error parsing token: %w", err)
	}

	if !token.Valid {
		return "", fmt.Errorf("invalid token: %w", err)
	}

	return claims.UserID, nil
}

// AuthenticationMiddleware checks if the user has a valid JWT token
func AuthenticationMiddleware(logger *zap.Logger) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			authCookie, err := r.Cookie("Authorization")
			if err != nil {
				if err == http.ErrNoCookie {
					userID := uuid.NewString()
					newAuthCookie, err := BuildJWTString(userID, TOKEN_EXP)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					http.SetCookie(w, &http.Cookie{
						Name:     "Authorization",
						Value:    newAuthCookie,
						Expires:  time.Now().Add(TOKEN_EXP),
						HttpOnly: true,
					})

					r = r.WithContext(context.WithValue(r.Context(), "userID", userID))

					next(w, r)
				}
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			if authCookie.Value == "" {
				http.Error(w, "missing authorization cookie", http.StatusUnauthorized)
				return
			}

			userID, err := GetUserID(authCookie.Value)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			// TODO: define own type for key to avoid collisions
			r = r.WithContext(context.WithValue(r.Context(), "userID", userID))

			http.SetCookie(w, &http.Cookie{
				Name: "Authorization",
				// Value:   newCookie,
				Expires: time.Now().Add(TOKEN_EXP),
			})

			next(w, r)
		}
	}
}
