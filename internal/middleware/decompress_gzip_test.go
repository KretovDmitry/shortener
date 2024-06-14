package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnzip(t *testing.T) {
	var handler http.Handler = http.HandlerFunc((func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf8")
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.NoError(t, r.Body.Close(), "failed close body")
		_, err = w.Write(body)
		require.NoError(t, err)
	}))

	mockData := []byte("https://test.com")

	tests := []struct {
		contentEncoding string
		payload         []byte
	}{
		{
			contentEncoding: "gzip",
			payload:         compress(mockData),
		},
		{
			contentEncoding: "text/plain; charset=utf8",
			payload:         mockData,
		},
	}

	for _, tt := range tests {
		t.Run(tt.contentEncoding, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(tt.payload))
			w := httptest.NewRecorder()

			r.Header.Set("Content-Encoding", tt.contentEncoding)

			handler = Unzip(handler)

			handler.ServeHTTP(w, r)

			result := w.Result()
			require.NoError(t, result.Body.Close(), "failed close body")

			body, err := io.ReadAll(result.Body)
			assert.NoError(t, err)
			assert.EqualValues(t, http.StatusOK, result.StatusCode)
			assert.Equal(t, mockData, body)
		})
	}
}

func compress(data []byte) []byte {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	_, err := gz.Write(data)
	if err != nil {
		log.Fatal(err)
	}
	err = gz.Close() // DO NOT DEFER HERE
	if err != nil {
		log.Fatal(err)
	}
	return b.Bytes()
}
