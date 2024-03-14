package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShortenJSON(t *testing.T) {
	emptyMockStore := &mockStore{expectedData: ""}
	path := "/api/shorten"

	getJSONResponsePayload := func(r *http.Response) (res shortenJSONResponsePayload) {
		err := json.NewDecoder(r.Body).Decode(&res)
		require.NoError(t, err, "failed to decode response JSON")
		r.Body.Close()
		return
	}

	getShortURL := func(s string) (res string) {
		if strings.HasPrefix(s, "http") {
			slice := strings.Split(s, "/")
			res = slice[len(slice)-1]
		}
		return
	}

	tests := []struct {
		name               string
		method             string
		requestContentType string
		payload            io.Reader
		store              *mockStore
		assertResult       func(r *http.Response)
	}{
		{
			name:               "positive test #1",
			method:             http.MethodPost,
			requestContentType: applicationJSON,
			payload:            strings.NewReader(`{"url":"https://e.mail.ru/inbox/"}`),
			store:              emptyMockStore,
			assertResult: func(r *http.Response) {
				defer r.Body.Close()
				assert.Equal(t, http.StatusCreated, r.StatusCode)
				if assert.Equal(t, applicationJSON, r.Header.Get(contentType)) {
					payload := getJSONResponsePayload(r)
					shortURL := getShortURL(string(payload.Result))
					assert.Equal(t, "be8xnp4H", shortURL)
				}
			},
		},
		{
			name:               "positive test #2",
			method:             http.MethodPost,
			requestContentType: applicationJSON,
			payload:            strings.NewReader(`{"url":"https://go.dev/"}`),
			store:              emptyMockStore,
			assertResult: func(r *http.Response) {
				defer r.Body.Close()
				assert.Equal(t, http.StatusCreated, r.StatusCode)
				if assert.Equal(t, applicationJSON, r.Header.Get(contentType)) {
					payload := getJSONResponsePayload(r)
					shortURL := getShortURL(string(payload.Result))
					assert.Equal(t, "eDKZ8wBC", shortURL)
				}
			},
		},
		{
			name:               "positive test #3: status code 409 (Conflict)",
			method:             http.MethodPost,
			requestContentType: applicationJSON,
			payload:            strings.NewReader(`{"url":"https://go.dev/"}`),
			store:              &mockStore{expectedData: "https://go.dev/"},
			assertResult: func(r *http.Response) {
				defer r.Body.Close()
				assert.Equal(t, http.StatusConflict, r.StatusCode)
				if assert.Equal(t, applicationJSON, r.Header.Get(contentType)) {
					payload := getJSONResponsePayload(r)
					shortURL := getShortURL(string(payload.Result))
					assert.Equal(t, "eDKZ8wBC", shortURL)
				}
			},
		},
		{
			name:               "invalid method",
			method:             http.MethodGet,
			requestContentType: applicationJSON,
			payload:            strings.NewReader(`{"url":"https://go.dev/"}`),
			store:              emptyMockStore,
			assertResult: func(r *http.Response) {
				defer r.Body.Close()
				assert.Equal(t, http.StatusBadRequest, r.StatusCode)
				if assert.Equal(t, textPlain, r.Header.Get(contentType)) {
					payload := getTextPayload(t, r)
					expectedResponse := "Only POST method is allowed"
					assert.Equal(t, expectedResponse, payload)
				}
			},
		},
		{
			name:               "invalid content-type",
			method:             http.MethodPost,
			requestContentType: textPlain,
			payload:            strings.NewReader(`{"url":"https://go.dev/"}`),
			store:              emptyMockStore,
			assertResult: func(r *http.Response) {
				defer r.Body.Close()
				assert.Equal(t, http.StatusBadRequest, r.StatusCode)
				if assert.Equal(t, textPlain, r.Header.Get(contentType)) {
					payload := getTextPayload(t, r)
					expectedResponse := `Only "application/json" Content-Type is allowed`
					assert.Equal(t, expectedResponse, payload)
				}
			},
		},
		{
			name:               "invalid payload: invalid JSON",
			method:             http.MethodPost,
			requestContentType: applicationJSON,
			payload:            strings.NewReader(`{"url";"https://test.com"}`),
			store:              emptyMockStore,
			assertResult: func(r *http.Response) {
				defer r.Body.Close()
				assert.Equal(t, http.StatusInternalServerError, r.StatusCode)
				if assert.Equal(t, textPlain, r.Header.Get(contentType)) {
					payload := getTextPayload(t, r)
					expectedResponse := `failed decode request JSON body`
					assert.Equal(t, expectedResponse, payload)
				}
			},
		},
		{
			name:               "invalid payload: empty url field",
			method:             http.MethodPost,
			requestContentType: applicationJSON,
			payload:            strings.NewReader(`{"url":""}`),
			store:              emptyMockStore,
			assertResult: func(r *http.Response) {
				defer r.Body.Close()
				assert.Equal(t, http.StatusBadRequest, r.StatusCode)
				if assert.Equal(t, textPlain, r.Header.Get(contentType)) {
					payload := getTextPayload(t, r)
					expectedResponse := "url field is empty"
					assert.Equal(t, expectedResponse, payload)
				}
			},
		},
		{
			name:               "invalid payload: invalid url field",
			method:             http.MethodPost,
			requestContentType: applicationJSON,
			payload:            strings.NewReader(`{"url":"https://test...com"}`),
			store:              emptyMockStore,
			assertResult: func(r *http.Response) {
				defer r.Body.Close()
				assert.Equal(t, http.StatusBadRequest, r.StatusCode)
				if assert.Equal(t, textPlain, r.Header.Get(contentType)) {
					payload := getTextPayload(t, r)
					expectedResponse := "Provided string is not a URL: https://test...com"
					assert.Equal(t, expectedResponse, payload)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create request with the method, content type and the payload being tested
			r := httptest.NewRequest(tt.method, path, tt.payload)
			r.Header.Set(contentType, tt.requestContentType)

			// response recorder
			w := httptest.NewRecorder()

			// context with mock store, stop test if failed to init context
			hctx, err := New(tt.store)
			require.NoError(t, err, "new handler context error")

			// call the handler
			hctx.ShortenJSON(w, r)

			// get recorded data
			res := w.Result()
			defer res.Body.Close()

			// assert wanted result
			tt.assertResult(res)
		})
	}
}
