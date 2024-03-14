package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/KretovDmitry/shortener/internal/db"
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
			name:        "invalid method",
			method:      http.MethodGet,
			contentType: applicationJSON,
			payload:     strings.NewReader(`{"url":"https://go.dev/"}`),
			store:       emptyMockStore,
			want: want{
				statusCode: http.StatusBadRequest,
				response:   "bad method: " + ErrOnlyPOSTMethodIsAllowed.Error(),
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
				response:   "bad content-type: " + ErrOnlyApplicationJSONContentType.Error(),
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
				response:   "url field is empty: " + ErrURLIsNotProvided.Error(),
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
				response:   "provided url isn't valid: https://test...com: " + ErrURLIsNotProvided.Error(),
			},
			wantErr: true,
		},
		{
			name:        "failed to save URL to database",
			method:      http.MethodPost,
			contentType: applicationJSON,
			payload:     strings.NewReader(`{"url":"https://go.dev/"}`),
			store:       &brokenStore{},
			want: want{
				statusCode: http.StatusInternalServerError,
				response:   "failed to save to database: intentionally not working method",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create request with the method, content type and the payload being tested
			r := httptest.NewRequest(tt.method, path, tt.payload)
			r.Header.Set(contentType, tt.contentType)

			// response recorder
			w := httptest.NewRecorder()

			// context with mock store, stop test if failed to init context
			hctx, err := New(tt.store)
			require.NoError(t, err, "new handler context error")

			// call the handler
			hctx.ShortenJSON(w, r)

			// get recorded data
			res := w.Result()

			// decode the response, stop if could not decode
			response := getShortenJSONResponsePayload(t, res)
			res.Body.Close()

			// assert wanted result
			assert.Equal(t, tt.want.statusCode, res.StatusCode)
			switch {
			case tt.wantErr:
				assert.True(t, strings.Contains(response.Message, tt.want.response))
			case !tt.wantErr:
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
