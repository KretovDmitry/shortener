package middleware

import (
	"context"
	"errors"
	"net"
	"net/http"

	"github.com/KretovDmitry/shortener/internal/config"
	"github.com/KretovDmitry/shortener/internal/jwt"
	"github.com/KretovDmitry/shortener/internal/logger"
	"github.com/KretovDmitry/shortener/internal/models/user"
	"google.golang.org/grpc"

	"github.com/google/uuid"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/selector"
	"go.uber.org/zap"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

const realIPHeader = "X-Real-IP"

// Authorization is a middleware function that checks for an "Authorization" cookie
// and extracts the user ID from the JWT token. If the user ID is found, it adds
// it to the request context as a value associated with the UserIDCtxKey.
// It will not let pass through if a token is not provided or couldn't be parsed.
func OnlyWithTokenHTTP(config *config.Config, logger logger.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		f := func(w http.ResponseWriter, r *http.Request) {
			authCookie, err := r.Cookie("Authorization")
			if err != nil {
				if errors.Is(err, http.ErrNoCookie) {
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

// AuthorizationHTTP is a middleware function that checks for an "AuthorizationHTTP" cookie
// and extracts the user ID from the JWT token. If the user ID is found, it adds
// it to the request context as a value associated with the UserIDCtxKey.
// It will create new user id if cookie is not provided.
func AuthorizationHTTP(config *config.Config, logger logger.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		f := func(w http.ResponseWriter, r *http.Request) {
			authCookie, err := r.Cookie("Authorization")
			if err != nil {
				if errors.Is(err, http.ErrNoCookie) {
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

// AuthorizationRPC checks if authorization metadata is set.
// If set it populates context with user id.
// Otherwise rejects call with codes.Unauthenticated.
func AuthorizationRPC(config *config.Config, logger logger.Logger,
) grpc.UnaryServerInterceptor {
	// Setup auth matcher.
	allButHealthZ := func(_ context.Context, callMeta interceptors.CallMeta) bool {
		return healthpb.Health_ServiceDesc.ServiceName != callMeta.Service
	}

	authFn := func(ctx context.Context) (context.Context, error) {
		token, err := auth.AuthFromMD(ctx, "Bearer")
		if err != nil {
			logger.Errorf("auth failed: %v", err)
			return nil, err
		}

		id, err := jwt.GetUserID(token, config.JWT.SigningKey)
		if err != nil {
			logger.Errorf("failed to get user from context: %v", err)
			return nil, err
		}

		return user.NewContext(ctx, &user.User{ID: id}), nil
	}

	return selector.UnaryServerInterceptor(
		auth.UnaryServerInterceptor(authFn),
		selector.MatchFunc(allButHealthZ),
	)
}

// OnlyTrustedSubnetHTTP rejects all untrusted IP addresses for a HTTP server.
func OnlyTrustedSubnetHTTP(config *config.Config, logger logger.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		f := func(w http.ResponseWriter, r *http.Request) {
			ipStr := r.Header.Get(realIPHeader)
			ip := net.ParseIP(ipStr)
			if ip == nil {
				logger.Errorf(
					"invalid nginx configuration: invalid %q: %q",
					realIPHeader, ipStr)
				w.WriteHeader(http.StatusForbidden)
				return
			}

			if !config.TrustedSubnet.Contains(ip) {
				logger.Infof("untrusted IP address has been accessed: %q", ip)
				w.WriteHeader(http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(f)
	}
}
