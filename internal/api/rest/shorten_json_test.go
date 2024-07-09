package rest

import (
	"encoding/json"
	"fmt"
	"io"
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

func TestPostShortenJSON(t *testing.T) {
	path := "/api/shorten"

	type want struct {
		response   string
		statusCode int
	}

	tests := []struct {
		name        string
		method      string
		contentType string
		payload     io.Reader
		store       repository.URLStorage
		want        want
		wantErr     bool
	}{
		{
			name:        "positive test #1",
			method:      http.MethodPost,
			contentType: applicationJSON,
			payload:     strings.NewReader(`{"url":"https://e.mail.ru/inbox/"}`),
			store:       memstore.NewURLRepository(),
			want: want{
				statusCode: http.StatusCreated,
				response:   "TZqSKV4tcyE",
			},
			wantErr: false,
		},
		{
			name:        "positive test #2",
			method:      http.MethodPost,
			contentType: applicationJSON,
			payload:     strings.NewReader(`{"url":"https://go.dev/"}`),
			store:       memstore.NewURLRepository(),
			want: want{
				statusCode: http.StatusCreated,
				response:   "YBbxJEcQ9vq",
			},
			wantErr: false,
		},
		{
			name:        "positive test #3: status code 409 (Conflict)",
			method:      http.MethodPost,
			contentType: applicationJSON,
			payload:     strings.NewReader(`{"url":"https://go.dev/"}`),
			store: initMockStore(&models.URL{
				OriginalURL: "https://go.dev/",
				ShortURL:    "YBbxJEcQ9vq",
			}),
			want: want{
				statusCode: http.StatusConflict,
				response:   "YBbxJEcQ9vq",
			},
			wantErr: false,
		},
		{
			name:        "invalid method: method get",
			method:      http.MethodGet,
			contentType: applicationJSON,
			payload:     strings.NewReader(`{"url":"https://go.dev/"}`),
			store:       memstore.NewURLRepository(),
			want: want{
				statusCode: http.StatusBadRequest,
				response:   fmt.Sprintf("%s: %s", errs.ErrInvalidRequest, http.MethodGet),
			},
			wantErr: true,
		},
		{
			name:        "invalid method: method put",
			method:      http.MethodPut,
			contentType: applicationJSON,
			payload:     strings.NewReader(`{"url":"https://go.dev/"}`),
			store:       memstore.NewURLRepository(),
			want: want{
				statusCode: http.StatusBadRequest,
				response:   fmt.Sprintf("%s: %s", errs.ErrInvalidRequest, http.MethodPut),
			},
			wantErr: true,
		},
		{
			name:        "invalid method: method patch",
			method:      http.MethodPatch,
			contentType: applicationJSON,
			payload:     strings.NewReader(`{"url":"https://go.dev/"}`),
			store:       memstore.NewURLRepository(),
			want: want{
				statusCode: http.StatusBadRequest,
				response:   fmt.Sprintf("%s: %s", errs.ErrInvalidRequest, http.MethodPatch),
			},
			wantErr: true,
		},
		{
			name:        "invalid method: method delete",
			method:      http.MethodDelete,
			contentType: applicationJSON,
			payload:     strings.NewReader(`{"url":"https://go.dev/"}`),
			store:       memstore.NewURLRepository(),
			want: want{
				statusCode: http.StatusBadRequest,
				response:   fmt.Sprintf("%s: %s", errs.ErrInvalidRequest, http.MethodDelete),
			},
			wantErr: true,
		},
		{
			name:        "invalid content-type",
			method:      http.MethodPost,
			contentType: textPlain,
			payload:     strings.NewReader(`{"url":"https://go.dev/"}`),
			store:       memstore.NewURLRepository(),
			want: want{
				statusCode: http.StatusBadRequest,
				response:   fmt.Sprintf("%s: %s", errs.ErrInvalidRequest, textPlain),
			},
			wantErr: true,
		},
		{
			name:        "invalid payload: invalid JSON",
			method:      http.MethodPost,
			contentType: applicationJSON,
			payload:     strings.NewReader(`{"url";"https://test.com"}`),
			store:       memstore.NewURLRepository(),
			want: want{
				statusCode: http.StatusInternalServerError,
				response:   "failed to decode request",
			},
			wantErr: true,
		},
		{
			name:        "invalid payload: empty url field",
			method:      http.MethodPost,
			contentType: applicationJSON,
			payload:     strings.NewReader(`{"url":""}`),
			store:       memstore.NewURLRepository(),
			want: want{
				statusCode: http.StatusBadRequest,
				response:   fmt.Sprintf("%s: URL is not provided", errs.ErrInvalidRequest),
			},
			wantErr: true,
		},
		{
			name:        "invalid payload: invalid url",
			method:      http.MethodPost,
			contentType: applicationJSON,
			payload:     strings.NewReader(`{"url":"https://test...com"}`),
			store:       memstore.NewURLRepository(),
			want: want{
				statusCode: http.StatusBadRequest,
				response:   fmt.Sprintf("%s: invalid URL", errs.ErrInvalidRequest),
			},
			wantErr: true,
		},
		{
			name:        "failed to save url to database",
			method:      http.MethodPost,
			contentType: applicationJSON,
			payload:     strings.NewReader(`{"url":"https://go.dev/"}`),
			store:       &brokenStore{},
			want: want{
				statusCode: http.StatusInternalServerError,
				response:   fmt.Sprintf("%s: failed to save to database", errIntentionallyNotWorkingMethod),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, path, tt.payload)
			r.Header.Set(contentType, tt.contentType)
			r = r.WithContext(user.NewContext(r.Context(), &user.User{ID: "test"}))

			w := httptest.NewRecorder()

			l, _ := logger.NewForTest()
			c := config.NewForTest()

			handler, err := NewHandler(tt.store, c, l)
			require.NoError(t, err, "new handler context error")

			handler.PostShortenJSON(w, r)

			res := w.Result()

			response := getShortenJSONResponsePayload(t, res)
			require.NoError(t, res.Body.Close(), "failed close body")

			assert.Equal(t, tt.want.statusCode, res.StatusCode)
			switch {
			case tt.wantErr:
				assert.Equal(t, tt.wantErr, !response.Success)
				assert.True(t, strings.Contains(response.Message, tt.want.response))
			case !tt.wantErr:
				assert.Equal(t, !tt.wantErr, response.Success)
				assert.Equal(t, tt.want.response, getShortURL(response.Result))
			}
		})
	}
}

func TestShortenJSON_WithoutUserInContext(t *testing.T) {
	path := "/"
	payload := strings.NewReader(`{"url":"https://go.dev/"}`)

	r := httptest.NewRequest(http.MethodPost, path, payload)
	r.Header.Set(contentType, applicationJSON)

	w := httptest.NewRecorder()

	l, _ := logger.NewForTest()
	c := config.NewForTest()

	handler, err := NewHandler(memstore.NewURLRepository(), c, l)
	require.NoError(t, err, "new handler error")

	handler.PostShortenJSON(w, r)

	res := w.Result()

	response := getShortenJSONResponsePayload(t, res)
	require.NoError(t, res.Body.Close(), "failed close body")

	assert.Equal(t, http.StatusUnauthorized, res.StatusCode, "status code mismatch")
	assert.Equal(t, fmt.Sprintf("%s: no user found", errs.ErrUnauthorized),
		response.Message, "response message mismatch")
	assert.False(t, response.Success)
}

func getShortenJSONResponsePayload(t *testing.T, r *http.Response) shortenJSONResponsePayload {
	var res shortenJSONResponsePayload
	err := json.NewDecoder(r.Body).Decode(&res)
	require.NoError(t, err, "failed to decode response JSON")
	require.NoError(t, r.Body.Close(), "failed close body")
	return res
}
