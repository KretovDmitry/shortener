package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShortenText(t *testing.T) {
	// we don't retrieve any data from the store
	// handler returns newly created short URL
	emptyMockStore := &mockStore{expectedData: ""}

	path := "/"

	type want struct {
		statusCode int
		response   string
	}

	tests := []struct {
		name               string
		requestContentType string
		payload            string
		want               want
	}{
		{
			name:               "positive test #1",
			requestContentType: textPlain,
			payload:            "https://e.mail.ru/inbox/",
			want: want{
				statusCode: http.StatusCreated,
				response:   "be8xnp4H",
			},
		},
		{
			name:               "positive test #2",
			requestContentType: textPlain,
			payload:            "https://go.dev/",
			want: want{
				statusCode: http.StatusCreated,
				response:   "eDKZ8wBC",
			},
		},
		{
			name:               "positive test #3: charset=utf-8",
			requestContentType: "text/plain; charset=utf-8",
			payload:            "https://go.dev/",
			want: want{
				statusCode: http.StatusCreated,
				response:   "eDKZ8wBC",
			},
		},
		{
			name:               "negative test #1: invalid Content-Type",
			requestContentType: applicationJSON,
			payload:            "https://go.dev/",
			want: want{
				statusCode: http.StatusBadRequest,
				response:   `Only "text/plain" Content-Type is allowed`,
			},
		},
		{
			name:               "negative test #2: empty body",
			requestContentType: textPlain,
			payload:            "",
			want: want{
				statusCode: http.StatusBadRequest,
				response:   `Empty body, must contain URL`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create request with the content type and the payload being tested
			// the method and the path are always the same
			r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(tt.payload))
			r.Header.Set(contentType, tt.requestContentType)

			// response recorder
			w := httptest.NewRecorder()

			// context with mock store, stop test if failed to init context
			hctx, err := NewHandlerContext(emptyMockStore)
			require.NoError(t, err, "new handler context error")

			// call the handler
			hctx.ShortenText(w, r)

			// get recorded data
			res := w.Result()

			// read the data and close the body; stop test if failed to read body
			resBody, err := io.ReadAll(res.Body)
			defer res.Body.Close()
			require.NoError(t, err)

			// if response contains URL (positive scenarios), take only short URL
			strResBody := string(resBody)
			if strings.HasPrefix(strResBody, "http") {
				g := strings.Split(strResBody, "/")
				strResBody = g[len(g)-1]
			}

			// assert wanted data
			assert.Equal(t, tt.want.statusCode, res.StatusCode)
			assert.Equal(t, textPlain, res.Header.Get(contentType))
			assert.Equal(t, tt.want.response, strings.TrimSpace(strResBody))
		})
	}
}
