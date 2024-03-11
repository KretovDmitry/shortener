package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShortenJSON(t *testing.T) {
	// we don't retrieve any data from the store
	// handler returns newly created short URL
	emptyMockStore := &mockStore{expectedData: ""}

	path := "/api/shorten"

	createJSONRequestPayload := func(url string) shortenJSONRequestPayload {
		return shortenJSONRequestPayload{URL: url}
	}

	getJSONResponsePayload := func(r *http.Response) shortenJSONResponsePayload {
		decoder := json.NewDecoder(r.Body)
		defer r.Body.Close()
		var res shortenJSONResponsePayload
		err := decoder.Decode(&res)
		require.NoError(t, err, "failed to read JSON body")
		return res
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
		requestContentType string
		payload            shortenJSONRequestPayload
		want               func(r *http.Response)
	}{
		{
			name:               "positive test #1",
			requestContentType: applicationJSON,
			payload:            createJSONRequestPayload("https://e.mail.ru/inbox/"),
			want: func(r *http.Response) {
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
			requestContentType: applicationJSON,
			payload:            createJSONRequestPayload("https://go.dev/"),
			want: func(r *http.Response) {
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
			name:               "negative test #1: invalid Content-Type",
			requestContentType: textPlain,
			payload:            createJSONRequestPayload("https://go.dev/"),
			want: func(r *http.Response) {
				defer r.Body.Close()
				assert.Equal(t, http.StatusBadRequest, r.StatusCode)
				if assert.Equal(t, textPlain, r.Header.Get(contentType)) {
					payload := getTextPayload(t, r)
					expectedResponse := fmt.Sprintf(
						`Only "%s" Content-Type is allowed`, applicationJSON,
					)
					assert.Equal(t, expectedResponse, payload)
				}
			},
		},
		{
			name:               "negative test #2: empty URL field",
			requestContentType: applicationJSON,
			payload:            createJSONRequestPayload(""),
			want: func(r *http.Response) {
				defer r.Body.Close()
				assert.Equal(t, http.StatusBadRequest, r.StatusCode)
				if assert.Equal(t, textPlain, r.Header.Get(contentType)) {
					payload := getTextPayload(t, r)
					expectedResponse := "The URL field in the JSON body of the request is empty"
					assert.Equal(t, expectedResponse, payload)
				}
			},
		},
		{
			name:               "negative test #3: invalid URL",
			requestContentType: applicationJSON,
			payload:            createJSONRequestPayload("https://test...com"),
			want: func(r *http.Response) {
				defer r.Body.Close()
				assert.Equal(t, http.StatusBadRequest, r.StatusCode)
				if assert.Equal(t, textPlain, r.Header.Get(contentType)) {
					payload := getTextPayload(t, r)
					expectedResponse := "The provided string is not a URL: https://test...com"
					assert.Equal(t, expectedResponse, payload)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := json.Marshal(tt.payload)
			require.NoError(t, err, "failed to Marshall payload")
			payload := bytes.NewBuffer(p)

			// create request with the content type and the payload being tested
			// the method and the path are always the same
			r := httptest.NewRequest(http.MethodPost, path, payload)
			r.Header.Set(contentType, tt.requestContentType)

			// response recorder
			w := httptest.NewRecorder()

			// context with mock store, stop test if failed to init context
			hctx, err := New(emptyMockStore)
			require.NoError(t, err, "new handler context error")

			// call the handler
			hctx.ShortenJSON(w, r)

			// get recorded data
			res := w.Result()
			defer res.Body.Close()

			// assert wanted result
			tt.want(res)
		})
	}
}
