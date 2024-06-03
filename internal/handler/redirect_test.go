package handler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KretovDmitry/shortener/internal/db"
	"github.com/KretovDmitry/shortener/internal/errs"
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
			shortURL: "TZqSKV4t",
			store: initMockStore(&models.URL{
				OriginalURL: "https://e.mail.ru/inbox/",
				ShortURL:    "TZqSKV4t",
			}),
			assertResponse: func(res *http.Response) {
				defer res.Body.Close()
				assert.Equal(t, http.StatusTemporaryRedirect, res.StatusCode)
				assert.Equal(t, "https://e.mail.ru/inbox/", res.Header.Get("Location"))
			},
		},
		{
			name:     "positive test #2",
			method:   http.MethodGet,
			shortURL: "YBbxJEcQ",
			store: initMockStore(&models.URL{
				OriginalURL: "https://go.dev/",
				ShortURL:    "YBbxJEcQ",
			}),
			assertResponse: func(res *http.Response) {
				defer res.Body.Close()
				assert.Equal(t, http.StatusTemporaryRedirect, res.StatusCode)
				assert.Equal(t, "https://go.dev/", res.Header.Get("Location"))
			},
		},
		{
			name:     "invalid method: method post",
			method:   http.MethodPost,
			shortURL: "YBbxJEcQ",
			store:    initMockStore(&models.URL{OriginalURL: "https://go.dev/"}),
			assertResponse: func(res *http.Response) {
				defer res.Body.Close()
				assert.Equal(t, http.StatusBadRequest, res.StatusCode)
				assert.Equal(t,
					fmt.Sprintf("%s: %s", errs.ErrInvalidRequest, http.MethodPost),
					getResponseTextPayload(t, res))
			},
		},
		{
			name:     "invalid method: method put",
			method:   http.MethodPut,
			shortURL: "YBbxJEcQ",
			store:    initMockStore(&models.URL{OriginalURL: "https://go.dev/"}),
			assertResponse: func(res *http.Response) {
				defer res.Body.Close()
				assert.Equal(t, http.StatusBadRequest, res.StatusCode)
				assert.Equal(t,
					fmt.Sprintf("%s: %s", errs.ErrInvalidRequest, http.MethodPut),
					getResponseTextPayload(t, res))
			},
		},
		{
			name:     "invalid method: method patch",
			method:   http.MethodPatch,
			shortURL: "YBbxJEcQ",
			store:    initMockStore(&models.URL{OriginalURL: "https://go.dev/"}),
			assertResponse: func(res *http.Response) {
				defer res.Body.Close()
				assert.Equal(t, http.StatusBadRequest, res.StatusCode)
				assert.Equal(t,
					fmt.Sprintf("%s: %s", errs.ErrInvalidRequest, http.MethodPatch),
					getResponseTextPayload(t, res))
			},
		},
		{
			name:     "invalid method: method delete",
			method:   http.MethodDelete,
			shortURL: "YBbxJEcQ",
			store:    initMockStore(&models.URL{OriginalURL: "https://go.dev/"}),
			assertResponse: func(res *http.Response) {
				defer res.Body.Close()
				assert.Equal(t, http.StatusBadRequest, res.StatusCode)
				assert.Equal(t,
					fmt.Sprintf("%s: %s", errs.ErrInvalidRequest, http.MethodDelete),
					getResponseTextPayload(t, res))
			},
		},
		{
			name:     "invalid url: too long URL",
			method:   http.MethodGet,
			shortURL: "Too_Long_URL", // > 8 characters
			store:    db.NewInMemoryStore(),
			assertResponse: func(res *http.Response) {
				defer res.Body.Close()
				assert.Equal(t, http.StatusBadRequest, res.StatusCode)
				resBody := getResponseTextPayload(t, res)
				assert.Equal(t, fmt.Sprintf("%s: invalid URL", errs.ErrInvalidRequest), resBody)
			},
		},
		{
			name:     "invalid url: too short URL",
			method:   http.MethodGet,
			shortURL: "short", // < 8 characters
			store:    db.NewInMemoryStore(),
			assertResponse: func(res *http.Response) {
				defer res.Body.Close()
				assert.Equal(t, http.StatusBadRequest, res.StatusCode)
				resBody := getResponseTextPayload(t, res)
				assert.Equal(t, fmt.Sprintf("%s: invalid URL", errs.ErrInvalidRequest), resBody)
			},
		},
		{
			name:     "invalid url: invalid base58 characters",
			method:   http.MethodGet,
			shortURL: "O0Il0O", // 0OIl+/ are not used
			store:    db.NewInMemoryStore(),
			assertResponse: func(res *http.Response) {
				defer res.Body.Close()
				assert.Equal(t, http.StatusBadRequest, res.StatusCode)
				resBody := getResponseTextPayload(t, res)
				assert.Equal(t, fmt.Sprintf("%s: invalid URL", errs.ErrInvalidRequest), resBody)
			},
		},
		{
			name:     "no such URL",
			method:   http.MethodGet,
			shortURL: "2x1xx1x2",
			store:    db.NewInMemoryStore(),
			assertResponse: func(res *http.Response) {
				defer res.Body.Close()
				assert.Equal(t, http.StatusBadRequest, res.StatusCode)
				resBody := getResponseTextPayload(t, res)
				assert.Equal(t, fmt.Sprintf("%s: no such URL", errs.ErrNotFound), resBody)
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
				assert.Equal(t, fmt.Sprintf("%s: failed to retrieve url", errIntentionallyNotWorkingMethod), resBody)
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
			handler, err := New(tt.store, 5)
			require.NoError(t, err, "new handler context error")

			// call the handler
			handler.Redirect(w, r)

			// get recorded data
			res := w.Result()
			defer res.Body.Close()

			// assert wanted data
			assert.Equal(t, textPlain, res.Header.Get(contentType))
			tt.assertResponse(res)
		})
	}
}
