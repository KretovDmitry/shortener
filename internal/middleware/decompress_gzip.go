package middleware

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/KretovDmitry/shortener/internal/logger"
	"go.uber.org/zap"
)

// compressReader implements ReadCloser interface
// and replaces Read method with a decompression one
type compressReader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

// NewCompressReader creates a new compressReader instance.
// It takes an io.ReadCloser and returns a new compressReader
// with a new gzip.Reader.
func newCompressReader(r io.ReadCloser) (*compressReader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("new gzip reader: %w", err)
	}

	return &compressReader{
		r:  r,
		zr: zr,
	}, nil
}

// Read reads data from the underlying gzip.Reader and writes it to the provided byte slice.
// It returns the number of bytes read and any error that occurs during the read operation.
func (c compressReader) Read(p []byte) (n int, err error) {
	return c.zr.Read(p)
}

// Close closes the underlying gzip.Reader and io.ReadCloser.
// It returns any error that occurs during the close operation.
func (c *compressReader) Close() error {
	if err := c.r.Close(); err != nil {
		return fmt.Errorf("close failed: %w", err)
	}
	return c.zr.Close()
}

// Unzip decides whether or not to decompress request judging by content encoding.
func Unzip(next http.Handler) http.Handler {
	l := logger.Get()

	f := func(w http.ResponseWriter, r *http.Request) {
		contentEncoding := r.Header.Get("Content-Encoding")
		sendsGzip := strings.Contains(contentEncoding, "gzip")
		if sendsGzip {
			cr, err := newCompressReader(r.Body)
			if err != nil {
				l.Error("new compress reader", zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			r.Body = cr
			defer func() {
				if err = cr.Close(); err != nil {
					l.Errorf("close compress reader: %v", err)
				}
			}()
		}

		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(f)
}
