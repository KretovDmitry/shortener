package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/KretovDmitry/shortener/internal/config"
	"github.com/KretovDmitry/shortener/internal/errs"
	"github.com/KretovDmitry/shortener/internal/logger"
	"github.com/KretovDmitry/shortener/internal/models"
	"github.com/KretovDmitry/shortener/internal/models/user"
	"github.com/KretovDmitry/shortener/internal/repository"
	"github.com/KretovDmitry/shortener/internal/repository/memstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostShortenBatch(t *testing.T) {
	path := "/api/shorten/batch"

	const goodPayload = `
	[
		{"correlation_id":"42b4cb1b-abf0-44e7-89f9-72ad3a277e0a","original_url":"https://go.dev/"},{"correlation_id":"229d9603-8540-4925-83f6-5cb1f239a72b","original_url":"https://e.mail.ru/inbox/"}
	]`

	happyResponse := fmt.Sprintf(`[{"correlation_id":"42b4cb1b-abf0-44e7-89f9-72ad3a277e0a","short_url":"http://%[1]s/YBbxJEcQ9vq"},{"correlation_id":"229d9603-8540-4925-83f6-5cb1f239a72b","short_url":"http://%[1]s/TZqSKV4tcyE"}]`,
		config.DefaultAddress)

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
		response   string
		statusCode int
	}

	tests := []struct {
		name        string
		method      string
		contentType string
		payload     string
		store       repository.URLStorage
		want        want
		wantErr     bool
	}{
		{
			name:        "positive test #1",
			method:      http.MethodPost,
			contentType: applicationJSON,
			payload:     goodPayload,
			store:       memstore.NewURLRepository(),
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
			store:       initMockStore(&models.URL{OriginalURL: "https://go.dev/"}),
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
			store:       memstore.NewURLRepository(),
			want: want{
				statusCode: http.StatusBadRequest,
				response:   fmt.Sprintf("%s: %s", errs.ErrInvalidRequest, http.MethodGet),
			},
		},
		{
			name:        "invalid method: method put",
			method:      http.MethodPut,
			contentType: applicationJSON,
			payload:     goodPayload,
			store:       memstore.NewURLRepository(),
			want: want{
				statusCode: http.StatusBadRequest,
				response:   fmt.Sprintf("%s: %s", errs.ErrInvalidRequest, http.MethodPut),
			},
		},
		{
			name:        "invalid method: method patch",
			method:      http.MethodPatch,
			contentType: applicationJSON,
			payload:     goodPayload,
			store:       memstore.NewURLRepository(),
			want: want{
				statusCode: http.StatusBadRequest,
				response:   fmt.Sprintf("%s: %s", errs.ErrInvalidRequest, http.MethodPatch),
			},
		},
		{
			name:        "invalid method: method delete",
			method:      http.MethodDelete,
			contentType: applicationJSON,
			payload:     goodPayload,
			store:       memstore.NewURLRepository(),
			want: want{
				statusCode: http.StatusBadRequest,
				response:   fmt.Sprintf("%s: %s", errs.ErrInvalidRequest, http.MethodDelete),
			},
		},
		{
			name:        "invalid content-type: text/plain",
			method:      http.MethodPost,
			contentType: textPlain,
			payload:     goodPayload,
			store:       memstore.NewURLRepository(),
			want: want{
				statusCode: http.StatusBadRequest,
				response:   fmt.Sprintf("%s: %s", errs.ErrInvalidRequest, textPlain),
			},
		},
		{
			name:        "invalid JSON",
			method:      http.MethodPost,
			contentType: applicationJSON,
			payload:     invalidJSON,
			store:       memstore.NewURLRepository(),
			want: want{
				statusCode: http.StatusInternalServerError,
				response:   errs.ErrInvalidRequest.Error(),
			},
			wantErr: true,
		},
		{
			name:        "empty body",
			method:      http.MethodPost,
			contentType: applicationJSON,
			payload:     "",
			store:       memstore.NewURLRepository(),
			want: want{
				statusCode: http.StatusInternalServerError,
				response:   errs.ErrInvalidRequest.Error(),
			},
			wantErr: true,
		},
		{
			name:        "empty url",
			method:      http.MethodPost,
			contentType: applicationJSON,
			payload:     emptyURL,
			store:       memstore.NewURLRepository(),
			want: want{
				statusCode: http.StatusBadRequest,
				response:   fmt.Sprintf("%s: URL is not provided", errs.ErrInvalidRequest),
			},
		},
		{
			name:        "invalid url",
			method:      http.MethodPost,
			contentType: applicationJSON,
			payload:     invalidURL,
			store:       memstore.NewURLRepository(),
			want: want{
				statusCode: http.StatusBadRequest,
				response:   fmt.Sprintf("%s: invalid URL", errs.ErrInvalidRequest),
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
				response:   fmt.Sprintf("%s: failed to save to database", errIntentionallyNotWorkingMethod),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, path, strings.NewReader(tt.payload))
			r.Header.Set(contentType, tt.contentType)
			r = r.WithContext(user.NewContext(r.Context(), &user.User{ID: "test"}))

			w := httptest.NewRecorder()

			l, _ := logger.NewForTest()
			c := config.NewForTest()

			handler, err := New(tt.store, c, l)
			require.NoError(t, err, "new handler error")

			handler.PostShortenBatch(w, r)

			res := w.Result()

			response := getResponseTextPayload(t, res)
			require.NoError(t, res.Body.Close(), "failed close body")

			assert.Equal(t, tt.want.statusCode, res.StatusCode)
			switch {
			case tt.wantErr:
				if !assert.True(t, strings.Contains(response, tt.want.response)) {
					fmt.Println(response)
					fmt.Println(tt.want.response)
				}
			case !tt.wantErr:
				assert.Equal(t, tt.want.response, response)
			}
		})
	}
}

func TestShortenBatch_WithoutUserInContext(t *testing.T) {
	path := "/api/shorten/batch"

	payload, err := json.Marshal([]shortenBatchRequestPayload{
		{
			CorrelationID: "42b4cb1b-abf0-44e7-89f9-72ad3a277e0a",
			OriginalURL:   "https://go.dev/",
		},
	})
	require.NoError(t, err, "failed marshal payload")

	r := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(payload))
	r.Header.Set(contentType, applicationJSON)

	w := httptest.NewRecorder()

	l, _ := logger.NewForTest()
	c := config.NewForTest()

	handler, err := New(memstore.NewURLRepository(), c, l)
	require.NoError(t, err, "new handler error")

	handler.PostShortenBatch(w, r)

	res := w.Result()

	response := getResponseTextPayload(t, res)
	require.NoError(t, res.Body.Close(), "failed close body")

	assert.Equal(t, http.StatusUnauthorized, res.StatusCode, "status code mismatch")
	assert.Equal(t, fmt.Sprintf("%s: no user found", errs.ErrUnauthorized),
		response, "response message mismatch")
}
