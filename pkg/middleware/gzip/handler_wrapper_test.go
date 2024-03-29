package gzip

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const handlerTestSize = 256

func newHTTPInstance(payload []byte, wrapper ...func(next http.HandlerFunc) http.HandlerFunc) http.HandlerFunc {
	var handler http.HandlerFunc = func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf8")
		_, _ = w.Write(payload)
	}

	for _, wrap := range wrapper {
		handler = wrap(handler)
	}

	return handler
}

func newEchoHTTPInstance(payload []byte, wrapper ...func(next http.HandlerFunc) http.HandlerFunc) http.HandlerFunc {
	var handler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf8")

		var buf bytes.Buffer

		_, _ = io.Copy(&buf, r.Body)
		_, _ = buf.Write(payload)
		_, _ = w.Write(buf.Bytes())
	}

	for _, wrap := range wrapper {
		handler = wrap(handler)
	}

	return handler
}

type NopWriter struct {
	header http.Header
}

func NewNopWriter() *NopWriter {
	return &NopWriter{
		header: make(http.Header),
	}
}

func (n *NopWriter) Header() http.Header {
	return n.header
}

func (n *NopWriter) Write(data []byte) (int, error) {
	return len(data), nil
}

func (n *NopWriter) WriteHeader(_ int) {
	// relax
}

func TestNewHandler_Checks(t *testing.T) {
	assert.NotPanics(t, func() {
		NewHandler(Config{
			CompressionLevel: 5,
			MinContentLength: 100,
		})
	})

	assert.Panics(t, func() {
		NewHandler(Config{
			CompressionLevel: -4,
			MinContentLength: 100,
		})
	})

	assert.Panics(t, func() {
		NewHandler(Config{
			CompressionLevel: 10,
			MinContentLength: 100,
		})
	})

	assert.Panics(t, func() {
		NewHandler(Config{
			CompressionLevel: 5,
			MinContentLength: 0,
		})
	})

	assert.Panics(t, func() {
		NewHandler(Config{
			CompressionLevel: 5,
			MinContentLength: -1,
		})
	})
}

func TestHTTPWithDefaultHandler_404(t *testing.T) {
	var (
		g = newHTTPInstance(bigPayload, DefaultHandler().WrapHandler)
		r = httptest.NewRequest(http.MethodPost, "/404", nil)
		w = httptest.NewRecorder()
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/somewhere", g)

	r.Header.Set("Accept-Encoding", "gzip")

	mux.ServeHTTP(w, r)

	result := w.Result()
	defer result.Body.Close()

	assert.EqualValues(t, http.StatusNotFound, result.StatusCode)
	assert.Equal(t, "404 page not found\n", w.Body.String())
}

func TestSoloHTTP(t *testing.T) {
	var (
		g = newHTTPInstance(bigPayload)
		r = httptest.NewRequest(http.MethodPost, "/", nil)
		w = NewNopWriter()
	)

	r.Header.Set("Accept-Encoding", "gzip")

	g.ServeHTTP(w, r)

	assert.Empty(t, w.Header().Get("Content-Encoding"))
}

func TestHTTPWithDefaultHandler(t *testing.T) {
	var (
		g = newEchoHTTPInstance(bigPayload, DefaultHandler().WrapHandler)
	)

	for i := 0; i < handlerTestSize; i++ {
		var seq = strconv.Itoa(i)
		t.Run(seq, func(t *testing.T) {
			t.Parallel()

			var (
				w = httptest.NewRecorder()
				r = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(seq))
			)

			r.Header.Set("Accept-Encoding", "gzip")
			g.ServeHTTP(w, r)

			result := w.Result()
			defer result.Body.Close()
			require.EqualValues(t, http.StatusOK, result.StatusCode)
			require.Equal(t, "gzip", result.Header.Get("Content-Encoding"))

			reader, err := gzip.NewReader(result.Body)
			require.NoError(t, err)
			defer reader.Close()
			body, err := io.ReadAll(reader)
			require.NoError(t, err)
			require.True(t, bytes.HasPrefix(body, []byte(seq)))
		})
	}
}

func TestHTTPWithDefaultHandler_TinyPayload_WriteTwice(t *testing.T) {
	var (
		handler http.HandlerFunc = func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/plain; charset=utf8")
			_, _ = io.WriteString(w, "part 1\n")
			_, _ = io.WriteString(w, "part 2\n")
		}
		r = httptest.NewRequest(http.MethodPost, "/", nil)
		w = httptest.NewRecorder()
	)

	r.Header.Set("Accept-Encoding", "gzip")
	handler = DefaultHandler().WrapHandler(handler)

	handler.ServeHTTP(w, r)

	result := w.Result()
	defer result.Body.Close()

	assert.EqualValues(t, http.StatusOK, result.StatusCode)
	assert.Empty(t, result.Header.Get("Vary"))
	assert.Empty(t, result.Header.Get("Content-Encoding"))
	assert.Equal(t, "part 1\npart 2\n", w.Body.String())
}

func TestHTTPWithDefaultHandler_TinyPayload_WriteThreeTimes(t *testing.T) {
	var (
		handler http.HandlerFunc = func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/plain; charset=utf8")
			_, _ = io.WriteString(w, "part 1\n")
			_, _ = io.WriteString(w, "part 2\n")
			_, _ = io.WriteString(w, "part 3\n")
		}
		r = httptest.NewRequest(http.MethodPost, "/", nil)
		w = httptest.NewRecorder()
	)

	r.Header.Set("Accept-Encoding", "gzip")
	handler = DefaultHandler().WrapHandler(handler)

	handler.ServeHTTP(w, r)

	result := w.Result()
	defer result.Body.Close()

	assert.EqualValues(t, http.StatusOK, result.StatusCode)
	assert.Empty(t, result.Header.Get("Vary"))
	assert.Empty(t, result.Header.Get("Content-Encoding"))
	assert.Equal(t, "part 1\npart 2\npart 3\n", w.Body.String())
}
