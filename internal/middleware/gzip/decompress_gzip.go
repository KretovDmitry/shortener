package gzip

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"github.com/KretovDmitry/shortener/internal/logger"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// compressReader implements ReadCloser interface
// and replaces Read method with a decompression one
type compressReader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

func newCompressReader(r io.ReadCloser) (*compressReader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, errors.Wrap(err, "new gzip reader")
	}

	return &compressReader{
		r:  r,
		zr: zr,
	}, nil
}

func (c compressReader) Read(p []byte) (n int, err error) {
	return c.zr.Read(p)
}

func (c *compressReader) Close() error {
	if err := c.r.Close(); err != nil {
		return errors.Wrap(err, "close failed")
	}
	return c.zr.Close()
}

// Unzip decides whether or not to decompress request
// judging by content encoding
func Unzip(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logger.Get()
		l.Sync()

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
			defer cr.Close()
		}

		next(w, r)
	}
}
