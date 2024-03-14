package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KretovDmitry/shortener/internal/db"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleShortURLRedirect(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		shortURL string
		store    db.URLStorage
		want     func(res *http.Response)
	}{
		{
			name:     "positive test #1",
			method:   http.MethodGet,
			shortURL: "be8xnp4H",
			store:    &mockStore{expectedData: "https://e.mail.ru/inbox/"},
			want: func(res *http.Response) {
				defer res.Body.Close()
				assert.Equal(t, http.StatusTemporaryRedirect, res.StatusCode)
				assert.Equal(t, "https://e.mail.ru/inbox/", res.Header.Get("Location"))
			},
		},
		{
			name:     "positive test #2",
			method:   http.MethodGet,
			shortURL: "eDKZ8wBC",
			store:    &mockStore{expectedData: "https://go.dev/"},
			want: func(res *http.Response) {
				defer res.Body.Close()
				assert.Equal(t, http.StatusTemporaryRedirect, res.StatusCode)
				assert.Equal(t, "https://go.dev/", res.Header.Get("Location"))
			},
		},
		{
			name:     "invalid method",
			method:   http.MethodPost,
			shortURL: "eDKZ8wBC",
			store:    &mockStore{expectedData: "https://go.dev/"},
			want: func(res *http.Response) {
				defer res.Body.Close()
				assert.Equal(t, http.StatusBadRequest, res.StatusCode)
				assert.Equal(t, `Only POST method is allowed`, getTextPayload(t, res))
			},
		},
		{
			name:     "invalid url: too long URL",
			method:   http.MethodGet,
			shortURL: "Too_Long_URL", // > 8 characters
			store:    &mockStore{expectedData: ""},
			want: func(res *http.Response) {
				defer res.Body.Close()
				assert.Equal(t, http.StatusBadRequest, res.StatusCode)
				resBody := getTextPayload(t, res)
				assert.Equal(t, "Invalid URL: Too_Long_URL", resBody)
			},
		},
		{
			name:     "invalid url: too short URL",
			method:   http.MethodGet,
			shortURL: "short", // < 8 characters
			store:    &mockStore{expectedData: ""},
			want: func(res *http.Response) {
				defer res.Body.Close()
				assert.Equal(t, http.StatusBadRequest, res.StatusCode)
				resBody := getTextPayload(t, res)
				assert.Equal(t, "Invalid URL: short", resBody)
			},
		},
		{
			name:     "invalid url: invalid base58 characters",
			method:   http.MethodGet,
			shortURL: "O0Il0O", // 0OIl+/ are not used
			store:    &mockStore{expectedData: ""},
			want: func(res *http.Response) {
				defer res.Body.Close()
				assert.Equal(t, http.StatusBadRequest, res.StatusCode)
				resBody := getTextPayload(t, res)
				assert.Equal(t, "Invalid URL: O0Il0O", resBody)
			},
		},
		{
			name:     "no such URL",
			method:   http.MethodGet,
			shortURL: "2x1xx1x2",
			store:    &mockStore{expectedData: ""},
			want: func(res *http.Response) {
				defer res.Body.Close()
				assert.Equal(t, http.StatusBadRequest, res.StatusCode)
				resBody := getTextPayload(t, res)
				assert.Equal(t, "No such URL: 2x1xx1x2", resBody)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, "/{shortURL}", http.NoBody)

			// add context to the request so that chi can identify the dynamic part of the URL
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("shortURL", tt.shortURL)

			r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

			// response recorder
			w := httptest.NewRecorder()

			// context with mock store, stop test if failed to init context
			hctx, err := New(tt.store)
			require.NoError(t, err, "new handler context error")

			// call the handler
			hctx.Redirect(w, r)

			// get recorded data
			res := w.Result()
			defer res.Body.Close()

			// assert wanted data
			assert.Equal(t, textPlain, res.Header.Get(contentType))
			tt.want(res)
		})
	}
}
