// Package handler provides handlers.
package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createShortURLRouter() chi.Router {
	r := chi.NewRouter()

	r.Post("/", CreateShortURL)
	return r
}

func createHandleShortURLRedirectRouter() chi.Router {
	r := chi.NewRouter()

	r.Get("/{shortURL}", HandleShortURLRedirect)
	return r
}

func testRequest(t *testing.T, ts *httptest.Server, method,
	path, contentType, payload string) (*http.Response, string) {

	req, err := http.NewRequest(method, ts.URL+path, strings.NewReader(payload))
	require.NoError(t, err)
	req.Host = "localhost:8080"
	req.Header.Set("Content-Type", contentType)

	resp, err := ts.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp, string(respBody)
}

func TestCreateShortURL(t *testing.T) {
	ts := httptest.NewServer(createShortURLRouter())
	defer ts.Close()

	path := "/"

	type want struct {
		statusCode  int
		contentType string
		response    string
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
				statusCode:  http.StatusCreated,
				contentType: "text/plain",
				response:    "http://localhost:8080/be8xnp4H",
			},
		},
		{
			name:        "positive test #2",
			contentType: "text/plain",
			payload:     "https://go.dev/",
			want: want{
				statusCode:  http.StatusCreated,
				contentType: "text/plain",
				response:    "http://localhost:8080/eDKZ8wBC",
			},
		},
		{
			name:        "positive test #3: charset=utf-8",
			contentType: "text/plain; charset=utf-8",
			payload:     "https://go.dev/",
			want: want{
				statusCode:  http.StatusCreated,
				contentType: "text/plain",
				response:    "http://localhost:8080/eDKZ8wBC",
			},
		},
		{
			name:        "negative test #1: invalid Content-Type",
			contentType: "application/json",
			payload:     "https://go.dev/",
			want: want{
				statusCode:  http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				response:    `Only "text/plain" Content-Type is allowed`,
			},
		},
		{
			name:        "negative test #2: empty body",
			contentType: "text/plain",
			payload:     "",
			want: want{
				statusCode:  http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				response:    `Empty body, must contain URL`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, get := testRequest(t, ts, http.MethodPost, path, tt.contentType, tt.payload)
			resp.Body.Close()
			assert.Equal(t, tt.want.statusCode, resp.StatusCode)
			assert.Equal(t, tt.want.contentType, resp.Header.Get("Content-Type"))
			assert.Equal(t, tt.want.response, strings.TrimSpace(get))
		})
	}
}

func TestHandleShortURLRedirect(t *testing.T) {
	tsToCreate := httptest.NewServer(createShortURLRouter())
	defer tsToCreate.Close()

	tsToRetrieve := httptest.NewServer(createHandleShortURLRedirectRouter())
	tsToRetrieve.Client().CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	defer tsToRetrieve.Close()

	contentType := "text/plain"

	type want struct {
		statusCode int
		response   string
	}
	tests := []struct {
		name       string
		path       string
		createMock bool
		want       want
	}{
		{
			name:       "positive test #1",
			path:       "/be8xnp4H",
			createMock: true,
			want: want{
				statusCode: http.StatusTemporaryRedirect,
				response:   "https://e.mail.ru/inbox/",
			},
		},
		{
			name:       "positive test #2",
			path:       "/eDKZ8wBC",
			createMock: true,
			want: want{
				statusCode: http.StatusTemporaryRedirect,
				response:   "https://go.dev/",
			},
		},
		{
			name:       "negative test #1: too long URL",
			path:       "/too_long_URL", // > 8 characters
			createMock: true,
			want: want{
				statusCode: http.StatusBadRequest,
				response:   "Invalid URL: too_long_URL",
			},
		},
		{
			name:       "negative test #2: too short URL",
			path:       "/short", // < 8 characters
			createMock: true,
			want: want{
				statusCode: http.StatusBadRequest,
				response:   "Invalid URL: short",
			},
		},
		{
			name:       "negative test #3: invalid base58 characters",
			path:       "/O0Il0O", // 0OIl+/ are not used
			createMock: true,
			want: want{
				statusCode: http.StatusBadRequest,
				response:   "Invalid URL: O0Il0O",
			},
		},
		{
			name:       "negative test #4: no such URL",
			path:       "/2x1xx1x2",
			createMock: false,
			want: want{
				statusCode: http.StatusBadRequest,
				response:   "No such URL: 2x1xx1x2",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.createMock {
				resp, _ := testRequest(t, tsToCreate, http.MethodPost, "/", contentType, tt.want.response)
				resp.Body.Close()
			}
			resp, get := testRequest(t, tsToRetrieve, http.MethodGet, tt.path, contentType, "")
			resp.Body.Close()

			assert.Equal(t, tt.want.statusCode, resp.StatusCode)

			if resp.StatusCode != http.StatusBadRequest {
				assert.Equal(t, tt.want.response, resp.Header.Get("Location"))
			} else {
				assert.Equal(t, tt.want.response, strings.TrimSpace(get))
			}
		})
	}
}
