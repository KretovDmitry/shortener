package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/KretovDmitry/shortener/internal/db"
	"github.com/KretovDmitry/shortener/internal/errs"
	"github.com/KretovDmitry/shortener/internal/models"
	"github.com/KretovDmitry/shortener/internal/models/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShortenText(t *testing.T) {
	path := "/"

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
	}{
		{
			name:        "positive test #1",
			method:      http.MethodPost,
			contentType: textPlain,
			payload:     "https://e.mail.ru/inbox/",
			store:       db.NewInMemoryStore(),
			want: want{
				statusCode: http.StatusCreated,
				response:   "TZqSKV4t",
			},
		},
		{
			name:        "positive test #2",
			method:      http.MethodPost,
			contentType: textPlain,
			payload:     "https://go.dev/",
			store:       db.NewInMemoryStore(),
			want: want{
				statusCode: http.StatusCreated,
				response:   "YBbxJEcQ",
			},
		},
		{
			name:        "positive test #3: status code 409 (Conflict)",
			method:      http.MethodPost,
			contentType: textPlain,
			payload:     "https://go.dev/",
			store: initMockStore(&models.URL{
				OriginalURL: "https://go.dev/",
				ShortURL:    "YBbxJEcQ",
			}),
			want: want{
				statusCode: http.StatusConflict,
				response:   "YBbxJEcQ",
			},
		},
		{
			name:        "invalid method: method get",
			method:      http.MethodGet,
			contentType: textPlain,
			payload:     "https://go.dev/",
			store:       db.NewInMemoryStore(),
			want: want{
				statusCode: http.StatusBadRequest,
				response:   fmt.Sprintf("%s: %s", errs.ErrInvalidRequest, http.MethodGet),
			},
		},
		{
			name:        "invalid method: method put",
			method:      http.MethodPut,
			contentType: textPlain,
			payload:     "https://go.dev/",
			store:       db.NewInMemoryStore(),
			want: want{
				statusCode: http.StatusBadRequest,
				response:   fmt.Sprintf("%s: %s", errs.ErrInvalidRequest, http.MethodPut),
			},
		},
		{
			name:        "invalid method: method patch",
			method:      http.MethodPatch,
			contentType: textPlain,
			payload:     "https://go.dev/",
			store:       db.NewInMemoryStore(),
			want: want{
				statusCode: http.StatusBadRequest,
				response:   fmt.Sprintf("%s: %s", errs.ErrInvalidRequest, http.MethodPatch),
			},
		},
		{
			name:        "invalid method: method delete",
			method:      http.MethodDelete,
			contentType: textPlain,
			payload:     "https://go.dev/",
			store:       db.NewInMemoryStore(),
			want: want{
				statusCode: http.StatusBadRequest,
				response:   fmt.Sprintf("%s: %s", errs.ErrInvalidRequest, http.MethodDelete),
			},
		},
		{
			name:        "invalid content-type: application/json",
			method:      http.MethodPost,
			contentType: applicationJSON,
			payload:     "https://go.dev/",
			store:       db.NewInMemoryStore(),
			want: want{
				statusCode: http.StatusBadRequest,
				response:   fmt.Sprintf("%s: %s", errs.ErrInvalidRequest, applicationJSON),
			},
		},
		{
			name:        "text plain with some charset: utf-16",
			method:      http.MethodPost,
			contentType: "text/plain; charset=utf-16",
			payload:     "https://go.dev/",
			store:       db.NewInMemoryStore(),
			want: want{
				statusCode: http.StatusCreated,
				response:   "YBbxJEcQ",
			},
		},
		{
			name:        "empty body",
			method:      http.MethodPost,
			contentType: textPlain,
			payload:     "",
			store:       db.NewInMemoryStore(),
			want: want{
				statusCode: http.StatusBadRequest,
				response:   fmt.Sprintf("%s: URL is not provided", errs.ErrInvalidRequest),
			},
		},
		{
			name:        "invalid url",
			method:      http.MethodPost,
			contentType: textPlain,
			payload:     "https://test...com",
			store:       db.NewInMemoryStore(),
			want: want{
				statusCode: http.StatusBadRequest,
				response:   fmt.Sprintf("%s: invalid URL", errs.ErrInvalidRequest),
			},
		},
		{
			name:        "failed to save URL to database",
			method:      http.MethodPost,
			contentType: textPlain,
			payload:     "https://go.dev/",
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

			handler, err := New(tt.store, 5)
			require.NoError(t, err, "new handler context error")

			handler.ShortenText(w, r)

			res := w.Result()

			response := getResponseTextPayload(t, res)
			res.Body.Close()

			// if response contains URL (positive scenarios), take only short URL
			if strings.HasPrefix(response, "http") {
				response = getShortURL(response)
			}

			assert.Equal(t, tt.want.statusCode, res.StatusCode)
			assert.Equal(t, textPlain, res.Header.Get(contentType))
			assert.Equal(t, tt.want.response, response)
		})
	}
}

func TestShortenText_WithoutUserInContext(t *testing.T) {
	path := "/"
	payload := "https://go.dev/"

	r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(payload))
	r.Header.Set(contentType, textPlain)

	w := httptest.NewRecorder()

	handler, err := New(db.NewInMemoryStore(), 5)
	require.NoError(t, err, "new handler error")

	handler.ShortenText(w, r)

	res := w.Result()

	response := getResponseTextPayload(t, res)
	res.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, res.StatusCode, "status code mismatch")
	assert.Equal(t, fmt.Sprintf("%s: no user found", errs.ErrUnauthorized),
		response, "response message mismatch")
}
