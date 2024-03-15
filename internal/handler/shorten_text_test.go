package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/KretovDmitry/shortener/internal/db"
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
			store:       emptyMockStore,
			want: want{
				statusCode: http.StatusCreated,
				response:   "be8xnp4H",
			},
		},
		{
			name:        "positive test #2",
			method:      http.MethodPost,
			contentType: textPlain,
			payload:     "https://go.dev/",
			store:       emptyMockStore,
			want: want{
				statusCode: http.StatusCreated,
				response:   "eDKZ8wBC",
			},
		},
		{
			name:        "positive test #3: status code 409 (Conflict)",
			method:      http.MethodPost,
			contentType: textPlain,
			payload:     "https://go.dev/",
			store:       &mockStore{expectedData: "https://go.dev/"},
			want: want{
				statusCode: http.StatusConflict,
				response:   "eDKZ8wBC",
			},
		},
		{
			name:        "invalid method: method get",
			method:      http.MethodGet,
			contentType: textPlain,
			payload:     "https://go.dev/",
			store:       emptyMockStore,
			want: want{
				statusCode: http.StatusBadRequest,
				response:   fmt.Sprintf("bad method: %s: %s", http.MethodGet, ErrOnlyPOSTMethodIsAllowed),
			},
		},
		{
			name:        "invalid method: method put",
			method:      http.MethodPut,
			contentType: textPlain,
			payload:     "https://go.dev/",
			store:       emptyMockStore,
			want: want{
				statusCode: http.StatusBadRequest,
				response:   fmt.Sprintf("bad method: %s: %s", http.MethodPut, ErrOnlyPOSTMethodIsAllowed),
			},
		},
		{
			name:        "invalid method: method patch",
			method:      http.MethodPatch,
			contentType: textPlain,
			payload:     "https://go.dev/",
			store:       emptyMockStore,
			want: want{
				statusCode: http.StatusBadRequest,
				response:   fmt.Sprintf("bad method: %s: %s", http.MethodPatch, ErrOnlyPOSTMethodIsAllowed),
			},
		},
		{
			name:        "invalid method: method delete",
			method:      http.MethodDelete,
			contentType: textPlain,
			payload:     "https://go.dev/",
			store:       emptyMockStore,
			want: want{
				statusCode: http.StatusBadRequest,
				response:   fmt.Sprintf("bad method: %s: %s", http.MethodDelete, ErrOnlyPOSTMethodIsAllowed),
			},
		},
		{
			name:        "invalid content-type: application/json",
			method:      http.MethodPost,
			contentType: applicationJSON,
			payload:     "https://go.dev/",
			store:       emptyMockStore,
			want: want{
				statusCode: http.StatusBadRequest,
				response:   fmt.Sprintf("bad content-type: %s: %s", applicationJSON, ErrOnlyTextPlainContentType),
			},
		},
		{
			name:        "text plain with some charset: utf-16",
			method:      http.MethodPost,
			contentType: "text/plain; charset=utf-16",
			payload:     "https://go.dev/",
			store:       emptyMockStore,
			want: want{
				statusCode: http.StatusCreated,
				response:   "eDKZ8wBC",
			},
		},
		{
			name:        "empty body",
			method:      http.MethodPost,
			contentType: textPlain,
			payload:     "",
			store:       emptyMockStore,
			want: want{
				statusCode: http.StatusBadRequest,
				response:   fmt.Sprintf("body is empty: %s", ErrURLIsNotProvided),
			},
		},
		{
			name:        "invalid url",
			method:      http.MethodPost,
			contentType: textPlain,
			payload:     "https://test...com",
			store:       emptyMockStore,
			want: want{
				statusCode: http.StatusBadRequest,
				response:   fmt.Sprintf("shorten url: https://test...com: %s", ErrNotValidURL),
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
				response: fmt.Sprintf(
					"failed to save to database: https://go.dev/: %s", errIntentionallyNotWorkingMethod),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create request with the method, content type and the payload being tested
			r := httptest.NewRequest(tt.method, path, strings.NewReader(tt.payload))
			r.Header.Set(contentType, tt.contentType)

			// response recorder
			w := httptest.NewRecorder()

			// context with mock store, stop test if failed to init context
			hctx, err := New(tt.store)
			require.NoError(t, err, "new handler context error")

			// call the handler
			hctx.ShortenText(w, r)

			// get recorded data
			res := w.Result()

			// read the response and close the body; stop test if failed to read body
			response := getResponseTextPayload(t, res)
			res.Body.Close()

			// if response contains URL (positive scenarios), take only short URL
			if strings.HasPrefix(response, "http") {
				response = getShortURL(response)
			}

			// assert wanted data
			assert.Equal(t, tt.want.statusCode, res.StatusCode)
			assert.Equal(t, textPlain, res.Header.Get(contentType))
			assert.Equal(t, tt.want.response, response)
		})
	}
}
