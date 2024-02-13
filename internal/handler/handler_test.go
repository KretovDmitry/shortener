// Package handler provides handlers.
package handler

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/KretovDmitry/shortener/internal/db"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mokeStore must implement db.Storage interface
type mockStore struct {
	expectedData string
}

// do nothing on create
func (s *mockStore) SaveURL(shortURL, url string) error {
	return nil
}

// return expected data
func (s *mockStore) RetrieveInitialURL(shortURL string) (string, error) {
	// mock not found error
	if s.expectedData == "" {
		return "", db.ErrURLNotFound
	}
	return s.expectedData, nil
}

func TestCreateShortURL(t *testing.T) {
	// we don't retrieve any data from the store
	// handler returns newly created short URL
	emptyMockStore := &mockStore{expectedData: ""}

	// should always return "text/plain; charset=utf-8" content type
	contentType := "text/plain; charset=utf-8"

	path := "/"

	type want struct {
		statusCode int
		response   string
	}

	tests := []struct {
		name        string
		contentType string
		payload     string
		want        want
	}{
		{
			name:        "positive test #1",
			contentType: "text/plain",
			payload:     "https://e.mail.ru/inbox/",
			want: want{
				statusCode: http.StatusCreated,
				response:   "be8xnp4H",
			},
		},
		{
			name:        "positive test #2",
			contentType: "text/plain",
			payload:     "https://go.dev/",
			want: want{
				statusCode: http.StatusCreated,
				response:   "eDKZ8wBC",
			},
		},
		{
			name:        "positive test #3: charset=utf-8",
			contentType: "text/plain; charset=utf-8",
			payload:     "https://go.dev/",
			want: want{
				statusCode: http.StatusCreated,
				response:   "eDKZ8wBC",
			},
		},
		{
			name:        "negative test #1: invalid Content-Type",
			contentType: "application/json",
			payload:     "https://go.dev/",
			want: want{
				statusCode: http.StatusBadRequest,
				response:   `Only "text/plain" Content-Type is allowed`,
			},
		},
		{
			name:        "negative test #2: empty body",
			contentType: "text/plain",
			payload:     "",
			want: want{
				statusCode: http.StatusBadRequest,
				response:   `Empty body, must contain URL`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create request with the content type being tested and the payload
			// the method and the path are always the same
			r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(tt.payload))
			r.Header.Set("Content-Type", tt.contentType)

			// response recorder
			w := httptest.NewRecorder()

			// context with mock store, stop test if failed to init context
			hctx, err := NewHandlerContext(emptyMockStore)
			require.NoError(t, err, "new handler context error")

			// call the handler
			hctx.CreateShortURL(w, r)

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
			assert.Equal(t, contentType, res.Header.Get("Content-Type"))
			assert.Equal(t, tt.want.response, strings.TrimSpace(strResBody))
		})
	}
}

func TestHandleShortURLRedirect(t *testing.T) {
	// should always return "text/plain; charset=utf-8" content type
	contentType := "text/plain; charset=utf-8"

	type want struct {
		statusCode int
		response   string
	}

	tests := []struct {
		name     string
		shortURL string
		store    db.Storage
		want     want
	}{
		{
			name:     "positive test #1",
			shortURL: "be8xnp4H",
			store:    &mockStore{expectedData: "https://e.mail.ru/inbox/"},
			want: want{
				statusCode: http.StatusTemporaryRedirect,
				response:   "https://e.mail.ru/inbox/",
			},
		},
		{
			name:     "positive test #2",
			shortURL: "eDKZ8wBC",
			store:    &mockStore{expectedData: "https://go.dev/"},
			want: want{
				statusCode: http.StatusTemporaryRedirect,
				response:   "https://go.dev/",
			},
		},
		{
			name:     "negative test #1: too long URL",
			shortURL: "too_long_URL", // > 8 characters
			store:    &mockStore{expectedData: ""},
			want: want{
				statusCode: http.StatusBadRequest,
				response:   "Invalid URL: too_long_URL",
			},
		},
		{
			name:     "negative test #2: too short URL",
			shortURL: "short", // < 8 characters
			store:    &mockStore{expectedData: ""},
			want: want{
				statusCode: http.StatusBadRequest,
				response:   "Invalid URL: short",
			},
		},
		{
			name:     "negative test #3: invalid base58 characters",
			shortURL: "O0Il0O", // 0OIl+/ are not used
			store:    &mockStore{expectedData: ""},
			want: want{
				statusCode: http.StatusBadRequest,
				response:   "Invalid URL: O0Il0O",
			},
		},
		{
			name:     "negative test #4: no such URL",
			shortURL: "2x1xx1x2",
			store:    &mockStore{expectedData: ""},
			want: want{
				statusCode: http.StatusBadRequest,
				response:   "No such URL: 2x1xx1x2",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/{shortURL}", http.NoBody)

			// add context to the request so that chi can identify the dynamic part of the URL
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("shortURL", tt.shortURL)

			r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

			// response recorder
			w := httptest.NewRecorder()

			// context with mock store, stop test if failed to init context
			hctx, err := NewHandlerContext(tt.store)
			require.NoError(t, err, "new handler context error")

			// call the handler
			hctx.HandleShortURLRedirect(w, r)

			// get recorded data
			res := w.Result()

			// read the data and close the body; stop test if failed to read body
			resBody, err := io.ReadAll(res.Body)
			defer res.Body.Close()
			require.NoError(t, err)

			// assert wanted data
			assert.Equal(t, contentType, res.Header.Get("Content-Type"))
			assert.Equal(t, tt.want.statusCode, res.StatusCode)

			if res.StatusCode != http.StatusBadRequest {
				assert.Equal(t, tt.want.response, res.Header.Get("Location"))
			} else {
				assert.Equal(t, tt.want.response, strings.TrimSpace(string(resBody)))
			}
		})
	}
}
