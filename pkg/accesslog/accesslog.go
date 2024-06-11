// Package accesslog provides a middleware that records every RESTful API
// call in a log message.
package accesslog

import (
	"fmt"
	"net/http"
	"time"

	"github.com/KretovDmitry/shortener/internal/logger"
	"github.com/go-chi/chi/v5/middleware"
)

// sugaredLogFormat is the format the Chi logs will use when
// a sugared Zap logger is passed. Uses fmt.Printf templating.
var sugaredLogFormat = "%s %s %s from %s - %s %dB in %s"

// Handler returns a middleware that records an access log message
// for every HTTP request being processed.
func Handler(log logger.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		f := func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			// associate request ID and session ID with the request context
			// so that they can be added to the log messages
			ctx := logger.WithRequest(r.Context(), r)
			r = r.WithContext(ctx)

			// defer function that logs the request details
			defer func(start time.Time) {
				log.With(ctx).Infof(sugaredLogFormat,
					r.Method,                 // Method
					r.URL.Path,               // Path
					r.Proto,                  // Protocol
					r.RemoteAddr,             // RemoteAddr
					statusLabel(ww.Status()), // "200 OK"
					ww.BytesWritten(),        // Bytes Written
					time.Since(start),        // Elapsed
				)
			}(time.Now())

			next.ServeHTTP(ww, r)
		}
		return http.HandlerFunc(f)
	}
}

func statusLabel(status int) string {
	switch {
	case status >= 100 && status < 300:
		return fmt.Sprintf("%d OK", status)
	case status >= 300 && status < 400:
		return fmt.Sprintf("%d Redirect", status)
	case status >= 400 && status < 500:
		return fmt.Sprintf("%d Client Error", status)
	case status >= 500:
		return fmt.Sprintf("%d Server Error", status)
	default:
		return fmt.Sprintf("%d Unknown", status)
	}
}
