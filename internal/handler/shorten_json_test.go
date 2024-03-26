package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/KretovDmitry/shortener/internal/db"
	"github.com/KretovDmitry/shortener/internal/models/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShortenJSON(t *testing.T) {
	path := "/api/shorten"

	type want struct {
		statusCode int
		response   string
	}

	tests := []struct {
		name        string
		method      string
		contentType string
		payload     io.Reader
		store       db.URLStorage
		want        want
		wantErr     bool
	}{
		{
			name:        "positive test #1",
			method:      http.MethodPost,
			contentType: applicationJSON,
			payload:     strings.NewReader(`{"url":"https://e.mail.ru/inbox/"}`),
			store:       emptyMockStore,
			want: want{
				statusCode: http.StatusCreated,
				response:   "be8xnp4H",
			},
			wantErr: false,
		},
		{
			name:        "positive test #2",
			method:      http.MethodPost,
			contentType: applicationJSON,
			payload:     strings.NewReader(`{"url":"https://go.dev/"}`),
			store:       emptyMockStore,
			want: want{
				statusCode: http.StatusCreated,
				response:   "eDKZ8wBC",
			},
			wantErr: false,
		},
		{
			name:        "positive test #3: status code 409 (Conflict)",
			method:      http.MethodPost,
			contentType: applicationJSON,
			payload:     strings.NewReader(`{"url":"https://go.dev/"}`),
			store:       &mockStore{expectedData: "https://go.dev/"},
			want: want{
				statusCode: http.StatusConflict,
				response:   "eDKZ8wBC",
			},
			wantErr: false,
		},
		{
			name:        "invalid method: method get",
			method:      http.MethodGet,
			contentType: applicationJSON,
			payload:     strings.NewReader(`{"url":"https://go.dev/"}`),
			store:       emptyMockStore,
			want: want{
				statusCode: http.StatusBadRequest,
				response:   fmt.Sprintf("bad method: %s: %s", http.MethodGet, ErrOnlyPOSTMethodIsAllowed),
			},
			wantErr: true,
		},
		{
			name:        "invalid method: method put",
			method:      http.MethodPut,
			contentType: applicationJSON,
			payload:     strings.NewReader(`{"url":"https://go.dev/"}`),
			store:       emptyMockStore,
			want: want{
				statusCode: http.StatusBadRequest,
				response:   fmt.Sprintf("bad method: %s: %s", http.MethodPut, ErrOnlyPOSTMethodIsAllowed),
			},
			wantErr: true,
		},
		{
			name:        "invalid method: method patch",
			method:      http.MethodPatch,
			contentType: applicationJSON,
			payload:     strings.NewReader(`{"url":"https://go.dev/"}`),
			store:       emptyMockStore,
			want: want{
				statusCode: http.StatusBadRequest,
				response:   fmt.Sprintf("bad method: %s: %s", http.MethodPatch, ErrOnlyPOSTMethodIsAllowed),
			},
			wantErr: true,
		},
		{
			name:        "invalid method: method delete",
			method:      http.MethodDelete,
			contentType: applicationJSON,
			payload:     strings.NewReader(`{"url":"https://go.dev/"}`),
			store:       emptyMockStore,
			want: want{
				statusCode: http.StatusBadRequest,
				response:   fmt.Sprintf("bad method: %s: %s", http.MethodDelete, ErrOnlyPOSTMethodIsAllowed),
			},
			wantErr: true,
		},
		{
			name:        "invalid content-type",
			method:      http.MethodPost,
			contentType: textPlain,
			payload:     strings.NewReader(`{"url":"https://go.dev/"}`),
			store:       emptyMockStore,
			want: want{
				statusCode: http.StatusBadRequest,
				response:   fmt.Sprintf("bad content-type: %s: %s", textPlain, ErrOnlyApplicationJSONContentType),
			},
			wantErr: true,
		},
		{
			name:        "invalid payload: invalid JSON",
			method:      http.MethodPost,
			contentType: applicationJSON,
			payload:     strings.NewReader(`{"url";"https://test.com"}`),
			store:       emptyMockStore,
			want: want{
				statusCode: http.StatusInternalServerError,
				response:   "failed to decode request",
			},
			wantErr: true,
		},
		{
			name:        "invalid payload: empty url field",
			method:      http.MethodPost,
			contentType: applicationJSON,
			payload:     strings.NewReader(`{"url":""}`),
			store:       emptyMockStore,
			want: want{
				statusCode: http.StatusBadRequest,
				response:   fmt.Sprintf("url field is empty: %s", ErrURLIsNotProvided),
			},
			wantErr: true,
		},
		{
			name:        "invalid payload: invalid url",
			method:      http.MethodPost,
			contentType: applicationJSON,
			payload:     strings.NewReader(`{"url":"https://test...com"}`),
			store:       emptyMockStore,
			want: want{
				statusCode: http.StatusBadRequest,
				response:   fmt.Sprintf("shorten url: https://test...com: %s", ErrNotValidURL),
			},
			wantErr: true,
		},
		{
			name:        "failed to save url to database",
			method:      http.MethodPost,
			contentType: applicationJSON,
			payload:     strings.NewReader(`{"url":"https://go.dev/"}`),
			store:       &brokenStore{},
			want: want{
				statusCode: http.StatusInternalServerError,
				response: fmt.Sprintf(
					"failed to save to database: https://go.dev/: %s", errIntentionallyNotWorkingMethod),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, path, tt.payload)
			r.Header.Set(contentType, tt.contentType)
			r = r.WithContext(user.NewContext(r.Context(), &user.User{ID: "test"}))

			w := httptest.NewRecorder()

			hctx, err := New(tt.store)
			require.NoError(t, err, "new handler context error")

			hctx.ShortenJSON(w, r)

			res := w.Result()

			response := getShortenJSONResponsePayload(t, res)
			res.Body.Close()

			assert.Equal(t, tt.want.statusCode, res.StatusCode)
			switch {
			case tt.wantErr:
				assert.Equal(t, tt.wantErr, !response.Success)
				assert.True(t, strings.Contains(response.Message, tt.want.response))
			case !tt.wantErr:
				assert.Equal(t, !tt.wantErr, response.Success)
				assert.Equal(t, tt.want.response, getShortURL(string(response.Result)))
			}
		})
	}
}

func getShortenJSONResponsePayload(t *testing.T, r *http.Response) (res shortenJSONResponsePayload) {
	err := json.NewDecoder(r.Body).Decode(&res)
	require.NoError(t, err, "failed to decode response JSON")
	r.Body.Close()
	return
}
