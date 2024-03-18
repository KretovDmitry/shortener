package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/KretovDmitry/shortener/internal/config"
	"github.com/KretovDmitry/shortener/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShortenBatch(t *testing.T) {
	path := "/api/shorten/batch"

	const goodPayload = `
	[
		{"correlation_id":"42b4cb1b-abf0-44e7-89f9-72ad3a277e0a","original_url":"https://go.dev/"},{"correlation_id":"229d9603-8540-4925-83f6-5cb1f239a72b","original_url":"https://e.mail.ru/inbox/"}
	]`

	happyResponse := fmt.Sprintf(`[{"correlation_id":"42b4cb1b-abf0-44e7-89f9-72ad3a277e0a","short_url":"http://%[1]s/eDKZ8wBC"},{"correlation_id":"229d9603-8540-4925-83f6-5cb1f239a72b","short_url":"http://%[1]s/be8xnp4H"}]`,
		config.AddrToReturn)

	const invalidJSON = `
	[
		"correlation_id":"42b4cb1b-abf0-44e7-89f9-72ad3a277e0a","original_url":"https://go.dev/"},{"correlation_id":"229d9603-8540-4925-83f6-5cb1f239a72b","original_url":"https://e.mail.ru/inbox/"}
	]`

	const emptyURL = `
	[
		{"correlation_id":"42b4cb1b-abf0-44e7-89f9-72ad3a277e0a","original_url":"https://go.dev/"},{"correlation_id":"229d9603-8540-4925-83f6-5cb1f239a72b","original_url":""}
	]`

	const invalidURL = `
	[
		{"correlation_id":"42b4cb1b-abf0-44e7-89f9-72ad3a277e0a","original_url":"https://go.dev/"},{"correlation_id":"229d9603-8540-4925-83f6-5cb1f239a72b","original_url":"https://test...com"}
	]`

	type want struct {
		statusCode int
		response   string
	}

	tests := []struct {
		name        string
		method      string
		contentType string
		payload     string
		store       db.URLStorage
		want        want
		wantErr     bool
	}{
		{
			name:        "positive test #1",
			method:      http.MethodPost,
			contentType: applicationJSON,
			payload:     goodPayload,
			store:       emptyMockStore,
			want: want{
				statusCode: http.StatusCreated,
				response:   happyResponse,
			},
		},
		{
			name:        "positive test #2",
			method:      http.MethodPost,
			contentType: applicationJSON,
			payload:     goodPayload,
			store:       &mockStore{expectedData: "https://go.dev/"},
			want: want{
				statusCode: http.StatusCreated,
				response:   happyResponse,
			},
		},
		{
			name:        "invalid method: method get",
			method:      http.MethodGet,
			contentType: applicationJSON,
			payload:     goodPayload,
			store:       emptyMockStore,
			want: want{
				statusCode: http.StatusBadRequest,
				response:   fmt.Sprintf("bad method: %s: %s", http.MethodGet, ErrOnlyPOSTMethodIsAllowed),
			},
		},
		{
			name:        "invalid method: method put",
			method:      http.MethodPut,
			contentType: applicationJSON,
			payload:     goodPayload,
			store:       emptyMockStore,
			want: want{
				statusCode: http.StatusBadRequest,
				response:   fmt.Sprintf("bad method: %s: %s", http.MethodPut, ErrOnlyPOSTMethodIsAllowed),
			},
		},
		{
			name:        "invalid method: method patch",
			method:      http.MethodPatch,
			contentType: applicationJSON,
			payload:     goodPayload,
			store:       emptyMockStore,
			want: want{
				statusCode: http.StatusBadRequest,
				response:   fmt.Sprintf("bad method: %s: %s", http.MethodPatch, ErrOnlyPOSTMethodIsAllowed),
			},
		},
		{
			name:        "invalid method: method delete",
			method:      http.MethodDelete,
			contentType: applicationJSON,
			payload:     goodPayload,
			store:       emptyMockStore,
			want: want{
				statusCode: http.StatusBadRequest,
				response:   fmt.Sprintf("bad method: %s: %s", http.MethodDelete, ErrOnlyPOSTMethodIsAllowed),
			},
		},
		{
			name:        "invalid content-type: text/plain",
			method:      http.MethodPost,
			contentType: textPlain,
			payload:     goodPayload,
			store:       emptyMockStore,
			want: want{
				statusCode: http.StatusBadRequest,
				response:   fmt.Sprintf("bad content-type: %s: %s", textPlain, ErrOnlyApplicationJSONContentType),
			},
		},
		{
			name:        "invalid JSON",
			method:      http.MethodPost,
			contentType: applicationJSON,
			payload:     invalidJSON,
			store:       emptyMockStore,
			want: want{
				statusCode: http.StatusInternalServerError,
				response:   "failed to decode request",
			},
			wantErr: true,
		},
		{
			name:        "empty body",
			method:      http.MethodPost,
			contentType: applicationJSON,
			payload:     "",
			store:       emptyMockStore,
			want: want{
				statusCode: http.StatusInternalServerError,
				response:   "failed to decode request",
			},
			wantErr: true,
		},
		{
			name:        "empty url",
			method:      http.MethodPost,
			contentType: applicationJSON,
			payload:     emptyURL,
			store:       emptyMockStore,
			want: want{
				statusCode: http.StatusBadRequest,
				response:   fmt.Sprintf("url field is empty: %s", ErrURLIsNotProvided),
			},
		},
		{
			name:        "invalid url",
			method:      http.MethodPost,
			contentType: applicationJSON,
			payload:     invalidURL,
			store:       emptyMockStore,
			want: want{
				statusCode: http.StatusBadRequest,
				response:   fmt.Sprintf("shorten url: https://test...com: %s", ErrNotValidURL),
			},
		},
		{
			name:        "failed to save URL to database",
			method:      http.MethodPost,
			contentType: applicationJSON,
			payload:     goodPayload,
			store:       &brokenStore{},
			want: want{
				statusCode: http.StatusInternalServerError,
				response:   fmt.Sprintf("failed to save to database: %s", errIntentionallyNotWorkingMethod),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, path, strings.NewReader(tt.payload))
			r.Header.Set(contentType, tt.contentType)

			w := httptest.NewRecorder()

			hctx, err := New(tt.store)
			require.NoError(t, err, "new handler context error")

			hctx.ShortenBatch(w, r)

			res := w.Result()

			response := getResponseTextPayload(t, res)
			res.Body.Close()

			assert.Equal(t, tt.want.statusCode, res.StatusCode)
			switch {
			case tt.wantErr:
				assert.True(t, strings.Contains(response, tt.want.response))
			case !tt.wantErr:
				assert.Equal(t, tt.want.response, response)
			}

		})
	}
}
