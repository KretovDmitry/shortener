package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleShortURLRedirect(t *testing.T) {
	tests := []struct {
		name     string
		shortURL string
		store    URLStore
		want     func(res *http.Response)
	}{
		{
			name:     "positive test #1",
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
			shortURL: "eDKZ8wBC",
			store:    &mockStore{expectedData: "https://go.dev/"},
			want: func(res *http.Response) {
				defer res.Body.Close()
				assert.Equal(t, http.StatusTemporaryRedirect, res.StatusCode)
				assert.Equal(t, "https://go.dev/", res.Header.Get("Location"))
			},
		},
		{
			name:     "negative test #1: too long URL",
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
			name:     "negative test #2: too short URL",
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
			name:     "negative test #3: invalid base58 characters",
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
			name:     "negative test #4: no such URL",
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
			defer res.Body.Close()

			// assert wanted data
			assert.Equal(t, textPlain, res.Header.Get(contentType))
			tt.want(res)
		})
	}
}
