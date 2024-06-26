package middleware

import (
	"net/http"

	"github.com/KretovDmitry/shortener/internal/config"
	"github.com/KretovDmitry/shortener/internal/jwt"
	"github.com/KretovDmitry/shortener/internal/logger"
	"github.com/KretovDmitry/shortener/internal/models/user"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Authorization is a middleware function that checks for an "Authorization" cookie
// and extracts the user ID from the JWT token. If the user ID is found, it adds
// it to the request context as a value associated with the UserIDCtxKey.
// It will not let pass through if a token is not provided or couldn't be parsed.
func OnlyWithToken(config *config.Config, logger logger.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		f := func(w http.ResponseWriter, r *http.Request) {
			authCookie, err := r.Cookie("Authorization")
			if err != nil {
				if err == http.ErrNoCookie {
					http.Error(w, "Authorization cookie not found", http.StatusUnauthorized)
					logger.Debug("Authorization cookie not found")
					return
				}
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			id, err := jwt.GetUserID(authCookie.Value, config.JWT.SigningKey)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			logger.Debug("JWT token contains user ID", zap.String("id", id))
			ctx := user.NewContext(r.Context(), &user.User{ID: id})

			next.ServeHTTP(w, r.WithContext(ctx))
		}

		return http.HandlerFunc(f)
	}
}

// Authorization is a middleware function that checks for an "Authorization" cookie
// and extracts the user ID from the JWT token. If the user ID is found, it adds
// it to the request context as a value associated with the UserIDCtxKey.
// It will create new user id if cookie is not provided.
func Authorization(config *config.Config, logger logger.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		f := func(w http.ResponseWriter, r *http.Request) {
			authCookie, err := r.Cookie("Authorization")
			if err != nil {
				if err == http.ErrNoCookie {
					logger.Debug("Authorization cookie not found")
					ctx := user.NewContext(r.Context(), &user.User{ID: uuid.NewString()})

					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			id, err := jwt.GetUserID(authCookie.Value, config.JWT.SigningKey)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			logger.Debug("JWT token contains user ID", zap.String("id", id))
			ctx := user.NewContext(r.Context(), &user.User{ID: id})

			next.ServeHTTP(w, r.WithContext(ctx))
		}

		return http.HandlerFunc(f)
	}
}
