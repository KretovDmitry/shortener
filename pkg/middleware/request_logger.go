package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"go.uber.org/zap"
)

type (
	loggingResponseWriter struct {
		http.ResponseWriter
		status      int
		size        int
		wroteHeader bool
	}
)

func newLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	return &loggingResponseWriter{ResponseWriter: w}
}

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.size += size
	return size, err
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	if r.wroteHeader {
		return
	}

	r.status = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
	r.wroteHeader = true
}

// RequestLogger is a middleware function that logs the request and response details.
func RequestLogger(logger *zap.Logger) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					logger.Error("handler panic",
						zap.Any("error", err),
						zap.ByteString("trace", debug.Stack()),
					)
				}
			}()

			lrw := newLoggingResponseWriter(w)

			// defer function that logs the request details
			defer func(start time.Time) {
				logger.Info(
					fmt.Sprintf(
						"%s request to %s completed",
						r.Method,
						r.RequestURI,
					),
					zap.String("url", r.RequestURI),
					zap.String("method", r.Method),
					zap.Int("status", lrw.status),
					zap.Duration("duration", time.Since(start)),
					zap.Int("size", lrw.size),
				)
			}(time.Now())

			next(lrw, r)
		}
	}
}
