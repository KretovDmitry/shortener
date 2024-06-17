package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/KretovDmitry/shortener/internal/db"
	"github.com/KretovDmitry/shortener/internal/errs"
	"github.com/KretovDmitry/shortener/internal/logger"
	"github.com/KretovDmitry/shortener/internal/models"
	"github.com/KretovDmitry/shortener/internal/models/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostShortenText(t *testing.T) {
	path := "/"

	type want struct {
		response   string
		statusCode int
	}

	tests := []struct {
		name        string
		contentType string
		payload     string
		store       db.URLStorage
		want        want
	}{
		{
			name:        "positive test #1",
			contentType: textPlain,
			payload:     "https://e.mail.ru/inbox/",
			store:       db.NewInMemoryStore(),
			want: want{
				statusCode: http.StatusCreated,
				response:   "TZqSKV4tcyE",
			},
		},
		{
			name:        "positive test #2",
			contentType: textPlain,
			payload:     "https://go.dev/",
			store:       db.NewInMemoryStore(),
			want: want{
				statusCode: http.StatusCreated,
				response:   "YBbxJEcQ9vq",
			},
		},
		{
			name:        "positive test #3: status code 409 (Conflict)",
			contentType: textPlain,
			payload:     "https://go.dev/",
			store: initMockStore(&models.URL{
				OriginalURL: "https://go.dev/",
				ShortURL:    "YBbxJEcQ9vq",
			}),
			want: want{
				statusCode: http.StatusConflict,
				response:   "YBbxJEcQ9vq",
			},
		},
		{
			name:        "text plain with some charset: utf-16",
			contentType: "text/plain; charset=utf-16",
			payload:     "https://go.dev/",
			store:       db.NewInMemoryStore(),
			want: want{
				statusCode: http.StatusCreated,
				response:   "YBbxJEcQ9vq",
			},
		},
		{
			name:        "empty body",
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
			r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(tt.payload))
			r.Header.Set(contentType, tt.contentType)
			r = r.WithContext(user.NewContext(r.Context(), &user.User{ID: "test"}))

			w := httptest.NewRecorder()

			handler, err := New(tt.store, logger.Get(), 5)
			require.NoError(t, err, "new handler context error")

			handler.PostShortenText(w, r)

			res := w.Result()

			response := getResponseTextPayload(t, res)
			require.NoError(t, res.Body.Close(), "failed close body")

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

func TestPostShortenText_BadMethods(t *testing.T) {
	path := "/"
	payload := "https://go.dev"

	tests := []struct {
		name   string
		method string
	}{
		{"invalid method: get", http.MethodGet},
		{"invalid method: put", http.MethodPut},
		{"invalid method: head", http.MethodHead},
		{"invalid method: patch", http.MethodPatch},
		{"invalid method: trace", http.MethodTrace},
		{"invalid method: delete", http.MethodDelete},
		{"invalid method: connect", http.MethodConnect},
		{"invalid method: options", http.MethodOptions},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, path, strings.NewReader(payload))
			w := httptest.NewRecorder()

			handler, err := New(db.NewInMemoryStore(), logger.Get(), 5)
			require.NoError(t, err, "new handler context error")

			handler.PostShortenText(w, r)

			res := w.Result()

			response := getResponseTextPayload(t, res)
			require.NoError(t, res.Body.Close(), "failed close body")

			assert.Equal(t, http.StatusBadRequest, res.StatusCode)
			assert.Equal(t, textPlain, res.Header.Get(contentType))
			assert.Equal(t,
				fmt.Sprintf("%s: %s", errs.ErrInvalidRequest, tt.method),
				response,
			)
		})
	}
}

func TestPostShortenText_BadContentTypes(t *testing.T) {
	path := "/"
	payload := "https://go.dev"

	contentTypes := []string{
		"application/java-archive",
		"application/EDI-X12",
		"application/EDIFACT",
		"application/javascript (obsolete)",
		"application/octet-stream",
		"application/ogg",
		"application/pdf",
		"application/xhtml+xml",
		"application/x-shockwave-flash",
		"application/json",
		"application/ld+json",
		"application/xml",
		"application/zip",
		"application/x-www-form-urlencoded",
		"audio/mpeg",
		"audio/x-ms-wma",
		"audio/vnd.rn-realaudio",
		"audio/x-wav",
		"image/gif",
		"image/jpeg",
		"image/png",
		"image/tiff",
		"image/vnd.microsoft.icon",
		"image/x-icon",
		"image/vnd.djvu",
		"image/svg+xml",
		"multipart/mixed",
		"multipart/alternative",
		"multipart/related",
		"multipart/form-data",
		"text/css",
		"text/csv",
		"text/html",
		"text/javascript",
		"text/xml",
		"video/mpeg",
		"video/mp4",
		"video/quicktime",
		"video/x-ms-wmv",
		"video/x-msvideo",
		"video/x-flv",
		"video/webm",
		"application/vnd.android.package-archive",
		"application/vnd.oasis.opendocument.text",
		"application/vnd.oasis.opendocument.spreadsheet",
		"application/vnd.oasis.opendocument.presentation",
		"application/vnd.oasis.opendocument.graphics",
		"application/vnd.ms-excel",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"application/vnd.ms-powerpoint",
		"application/vnd.openxmlformats-officedocument.presentationml.presentation",
		"application/msword",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"application/vnd.mozilla.xul+xml",
	}
	for _, ct := range contentTypes {
		t.Run(ct, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(payload))
			r.Header.Set(contentType, ct)
			w := httptest.NewRecorder()

			handler, err := New(db.NewInMemoryStore(), logger.Get(), 5)
			require.NoError(t, err, "failed to init new handler")

			handler.PostShortenText(w, r)

			res := w.Result()

			response := getResponseTextPayload(t, res)
			require.NoError(t, res.Body.Close(), "failed to close body")

			assert.Equal(t, http.StatusBadRequest, res.StatusCode)
			assert.Equal(t, textPlain, res.Header.Get(contentType))
			assert.Equal(t,
				fmt.Sprintf("%s: %s", errs.ErrInvalidRequest, ct),
				response,
			)
		})
	}
}

func TestPostShortenText_WithoutUserInContext(t *testing.T) {
	path := "/"
	payload := "https://go.dev"

	r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(payload))
	r.Header.Set(contentType, textPlain)

	w := httptest.NewRecorder()

	handler, err := New(db.NewInMemoryStore(), logger.Get(), 5)
	require.NoError(t, err, "failed to init new handler")

	handler.PostShortenText(w, r)

	res := w.Result()

	response := getResponseTextPayload(t, res)
	require.NoError(t, res.Body.Close(), "failed close body")

	assert.Equal(t, http.StatusUnauthorized, res.StatusCode, "status code mismatch")
	assert.Equal(t, fmt.Sprintf("%s: no user found", errs.ErrUnauthorized),
		response, "response message mismatch")
}
