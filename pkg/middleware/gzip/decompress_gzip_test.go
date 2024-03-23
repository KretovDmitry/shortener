package gzip

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestUnzip(t *testing.T) {
	var handler http.HandlerFunc = (func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf8")
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		r.Body.Close()
		w.Write(body)
	})

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
			handler = Unzip(zap.L())(handler)

			handler(w, r)

			result := w.Result()
			defer result.Body.Close()

			body, err := io.ReadAll(result.Body)
			assert.NoError(t, err)
			assert.EqualValues(t, http.StatusOK, result.StatusCode)
			assert.Equal(t, mockData, body)
		})
	}
}

func compress(data []byte) []byte {
	var b bytes.Buffer
	wr := gzip.NewWriter(&b)
	_, err := wr.Write(data)
	if err != nil {
		panic(err)
	}
	wr.Close() // DO NOT DEFER HERE

	return b.Bytes()
}
