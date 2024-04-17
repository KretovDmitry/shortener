package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/KretovDmitry/shortener/internal/logger"
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
func Logger(next http.Handler) http.Handler {
	f := func(w http.ResponseWriter, r *http.Request) {
		l := logger.Get()
		defer l.Sync()

		defer func() {
			if err := recover(); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				l.Error("handler panic",
					zap.Any("error", err),
					zap.ByteString("trace", debug.Stack()),
				)
			}
		}()

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
				zap.Int("status", lrw.status),
				zap.Duration("duration", time.Since(start)),
				zap.Int("size", lrw.size),
			)
		}(time.Now())

		next.ServeHTTP(lrw, r)
	}

	return http.HandlerFunc(f)
}
