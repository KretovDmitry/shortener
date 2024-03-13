package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/KretovDmitry/shortener/internal/logger"
	"go.uber.org/zap"
)

type (
	responseData struct {
		status int
		size   int
	}

	loggingResponseWriter struct {
		http.ResponseWriter
		responseData *responseData
	}
)

func newLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	return &loggingResponseWriter{w, &responseData{}}
}

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size
	return size, err
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode
}

// RequestLogger is a middleware function that logs the request and response details
func RequestLogger(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logger.Get()
		defer l.Sync()

		lrw := newLoggingResponseWriter(w)

		// defer function that logs the request details
		defer func(start time.Time) {
			l.Info(
				fmt.Sprintf(
					"%s request to %s completed",
					r.Method,
					r.RequestURI,
				),
				zap.String("url", r.RequestURI),
				zap.String("method", r.Method),
				zap.Int("status", lrw.responseData.status),
				zap.Duration("duration", time.Since(start)),
				zap.Int("size", lrw.responseData.size),
			)
		}(time.Now())

		next(lrw, r)
	}
}
