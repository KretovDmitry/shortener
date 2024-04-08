package handler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KretovDmitry/shortener/internal/db"
	"github.com/KretovDmitry/shortener/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleShortURLRedirect(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		shortURL       string
		store          db.URLStorage
		assertResponse func(res *http.Response)
	}{
		{
			name:     "positive test #1",
			method:   http.MethodGet,
			shortURL: "be8xnp4H",
			store:    &mockStore{expectedData: "https://e.mail.ru/inbox/"},
			assertResponse: func(res *http.Response) {
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
			assertResponse: func(res *http.Response) {
				defer res.Body.Close()
				assert.Equal(t, http.StatusTemporaryRedirect, res.StatusCode)
				assert.Equal(t, "https://go.dev/", res.Header.Get("Location"))
			},
		},
		{
			name:     "invalid method: method post",
			method:   http.MethodPost,
			shortURL: "eDKZ8wBC",
			store:    &mockStore{expectedData: "https://go.dev/"},
			assertResponse: func(res *http.Response) {
				defer res.Body.Close()
				assert.Equal(t, http.StatusBadRequest, res.StatusCode)
				assert.Equal(t,
					fmt.Sprintf("bad method: %s: %s", http.MethodPost, ErrOnlyGETMethodIsAllowed), getResponseTextPayload(t, res))
			},
		},
		{
			name:     "invalid method: method put",
			method:   http.MethodPut,
			shortURL: "eDKZ8wBC",
			store:    &mockStore{expectedData: "https://go.dev/"},
			assertResponse: func(res *http.Response) {
				defer res.Body.Close()
				assert.Equal(t, http.StatusBadRequest, res.StatusCode)
				assert.Equal(t,
					fmt.Sprintf("bad method: %s: %s", http.MethodPut, ErrOnlyGETMethodIsAllowed),
					getResponseTextPayload(t, res))
			},
		},
		{
			name:     "invalid method: method patch",
			method:   http.MethodPatch,
			shortURL: "eDKZ8wBC",
			store:    &mockStore{expectedData: "https://go.dev/"},
			assertResponse: func(res *http.Response) {
				defer res.Body.Close()
				assert.Equal(t, http.StatusBadRequest, res.StatusCode)
				assert.Equal(t,
					fmt.Sprintf("bad method: %s: %s", http.MethodPatch, ErrOnlyGETMethodIsAllowed),
					getResponseTextPayload(t, res))
			},
		},
		{
			name:     "invalid method: method delete",
			method:   http.MethodDelete,
			shortURL: "eDKZ8wBC",
			store:    &mockStore{expectedData: "https://go.dev/"},
			assertResponse: func(res *http.Response) {
				defer res.Body.Close()
				assert.Equal(t, http.StatusBadRequest, res.StatusCode)
				assert.Equal(t,
					fmt.Sprintf("bad method: %s: %s", http.MethodDelete, ErrOnlyGETMethodIsAllowed),
					getResponseTextPayload(t, res))
			},
		},
		{
			name:     "invalid url: too long URL",
			method:   http.MethodGet,
			shortURL: "Too_Long_URL", // > 8 characters
			store:    &mockStore{expectedData: ""},
			assertResponse: func(res *http.Response) {
				defer res.Body.Close()
				assert.Equal(t, http.StatusBadRequest, res.StatusCode)
				resBody := getResponseTextPayload(t, res)
				assert.Equal(t, fmt.Sprintf("redirect with url: Too_Long_URL: %s", ErrNotValidURL), resBody)
			},
		},
		{
			name:     "invalid url: too short URL",
			method:   http.MethodGet,
			shortURL: "short", // < 8 characters
			store:    &mockStore{expectedData: ""},
			assertResponse: func(res *http.Response) {
				defer res.Body.Close()
				assert.Equal(t, http.StatusBadRequest, res.StatusCode)
				resBody := getResponseTextPayload(t, res)
				assert.Equal(t, fmt.Sprintf("redirect with url: short: %s", ErrNotValidURL), resBody)
			},
		},
		{
			name:     "invalid url: invalid base58 characters",
			method:   http.MethodGet,
			shortURL: "O0Il0O", // 0OIl+/ are not used
			store:    &mockStore{expectedData: ""},
			assertResponse: func(res *http.Response) {
				defer res.Body.Close()
				assert.Equal(t, http.StatusBadRequest, res.StatusCode)
				resBody := getResponseTextPayload(t, res)
				assert.Equal(t, fmt.Sprintf("redirect with url: O0Il0O: %s", ErrNotValidURL), resBody)
			},
		},
		{
			name:     "no such URL",
			method:   http.MethodGet,
			shortURL: "2x1xx1x2",
			store:    &mockStore{expectedData: ""},
			assertResponse: func(res *http.Response) {
				defer res.Body.Close()
				assert.Equal(t, http.StatusBadRequest, res.StatusCode)
				resBody := getResponseTextPayload(t, res)
				assert.Equal(t, fmt.Sprintf("redirect with url: 2x1xx1x2: %s", models.ErrNotFound), resBody)
			},
		},
		{
			name:     "failed to get url from database",
			method:   http.MethodGet,
			shortURL: "2x1xx1x2",
			store:    &brokenStore{},
			assertResponse: func(res *http.Response) {
				defer res.Body.Close()
				assert.Equal(t, http.StatusInternalServerError, res.StatusCode)
				resBody := getResponseTextPayload(t, res)
				assert.Equal(t,
					fmt.Sprintf("failed to retrieve url: 2x1xx1x2: %s", errIntentionallyNotWorkingMethod), resBody)
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
			hctx, err := New(tt.store, 5)
			require.NoError(t, err, "new handler context error")

			// call the handler
			hctx.Redirect(w, r)

			// get recorded data
			res := w.Result()
			defer res.Body.Close()

			// assert wanted data
			assert.Equal(t, textPlain, res.Header.Get(contentType))
			tt.assertResponse(res)
		})
	}
}
