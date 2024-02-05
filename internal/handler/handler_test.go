// Package handler provides handlers.
package handler

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/KretovDmitry/shortener/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateShortURL(t *testing.T) {
	type want struct {
		statusCode  int
		contentType string
		shortURL    string
	}
	tests := []struct {
		name         string
		URLToShorten string
		want         want
	}{
		{
			name:         "positive test #1",
			URLToShorten: "https://e.mail.ru/inbox/",
			want: want{
				statusCode:  http.StatusCreated,
				contentType: "text/plain",
				shortURL:    "be8xnp4H",
			},
		},
		{
			name:         "positive test #2",
			URLToShorten: "https://go.dev/",
			want: want{
				statusCode:  http.StatusCreated,
				contentType: "text/plain",
				shortURL:    "eDKZ8wBC",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(tt.URLToShorten)))
			recorder := httptest.NewRecorder()

			CreateShortURL(recorder, request)

			response := recorder.Result()

			assert.Equal(t, tt.want.statusCode, response.StatusCode)
			assert.Equal(t, tt.want.contentType, response.Header.Get("content-type"))

			responseBody, err := io.ReadAll(response.Body)
			defer response.Body.Close()
			require.NoError(t, err, "failed to read response body")

			assert.True(t, strings.HasSuffix(string(responseBody), tt.want.shortURL))
		})
	}
}

func TestHandleShortURLRedirect(t *testing.T) {
	type want struct {
		statusCode  int
		originalURL string
	}
	tests := []struct {
		name     string
		shortURL string
		mockDB   *db.DB
		want     want
	}{
		{
			name:     "positive test #1",
			shortURL: "be8xnp4H",
			mockDB: db.GetDB().Init(map[string]string{
				"be8xnp4H": "https://e.mail.ru/inbox/",
			}),
			want: want{
				statusCode:  http.StatusTemporaryRedirect,
				originalURL: "https://e.mail.ru/inbox/",
			},
		},
		{
			name:     "positive test #2",
			shortURL: "eDKZ8wBC",
			mockDB: db.GetDB().Init(map[string]string{
				"eDKZ8wBC": "https://go.dev/",
			}),
			want: want{
				statusCode:  http.StatusTemporaryRedirect,
				originalURL: "https://go.dev/",
			},
		},
		{
			name:     "negative test #1: invalid shortURL",
			shortURL: "some_invalid_URL",
			mockDB: db.GetDB().Init(map[string]string{
				"eDKZ8wBC": "https://go.dev/",
			}),
			want: want{
				statusCode:  http.StatusBadRequest,
				originalURL: "",
			},
		},
		{
			name:     "negative test #2: getting a non-existent record",
			shortURL: "eeeeeeee",
			want: want{
				statusCode:  http.StatusBadRequest,
				originalURL: "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/%s", tt.shortURL), nil)
			recorder := httptest.NewRecorder()

			HandleShortURLRedirect(recorder, request)

			response := recorder.Result()

			assert.Equal(t, tt.want.statusCode, response.StatusCode)
			assert.Equal(t, tt.want.originalURL, response.Header.Get("location"))
		})
	}
}
