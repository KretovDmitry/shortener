package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KretovDmitry/shortener/internal/db"
	"github.com/KretovDmitry/shortener/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPingDB(t *testing.T) {
	path := "/"

	type want struct {
		statusCode int
		response   string
	}

	tests := []struct {
		name   string
		method string
		store  db.URLStorage
		want   want
	}{
		{
			name:   "positive test #1",
			method: http.MethodGet,
			store:  emptyMockStore,
			want: want{
				statusCode: http.StatusOK,
				response:   "",
			},
		},
		{
			name:   "positive test #2",
			method: http.MethodGet,
			store:  emptyMockStore,
			want: want{
				statusCode: http.StatusOK,
				response:   "",
			},
		},
		{
			name:   "invalid method: method post",
			method: http.MethodPost,
			store:  emptyMockStore,
			want: want{
				statusCode: http.StatusBadRequest,
				response:   fmt.Sprintf("bad method: %s: %s", http.MethodPost, ErrOnlyGETMethodIsAllowed),
			},
		},
		{
			name:   "invalid method: method put",
			method: http.MethodPut,
			store:  emptyMockStore,
			want: want{
				statusCode: http.StatusBadRequest,
				response:   fmt.Sprintf("bad method: %s: %s", http.MethodPut, ErrOnlyGETMethodIsAllowed),
			},
		},
		{
			name:   "invalid method: method patch",
			method: http.MethodPatch,
			store:  emptyMockStore,
			want: want{
				statusCode: http.StatusBadRequest,
				response:   fmt.Sprintf("bad method: %s: %s", http.MethodPatch, ErrOnlyGETMethodIsAllowed),
			},
		},
		{
			name:   "invalid method: method delete",
			method: http.MethodDelete,
			store:  emptyMockStore,
			want: want{
				statusCode: http.StatusBadRequest,
				response:   fmt.Sprintf("bad method: %s: %s", http.MethodDelete, ErrOnlyGETMethodIsAllowed),
			},
		},
		{
			name:   "DB not connected",
			method: http.MethodGet,
			store:  &notConnectedStore{},
			want: want{
				statusCode: http.StatusInternalServerError,
				response:   fmt.Sprintf("DB not connected: %s", models.ErrDBNotConnected),
			},
		},
		{
			name:   "connection error",
			method: http.MethodGet,
			store:  &brokenStore{},
			want: want{
				statusCode: http.StatusInternalServerError,
				response:   fmt.Sprintf("connection error: %s", errIntentionallyNotWorkingMethod),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create request with the method, content type and the payload being tested
			r := httptest.NewRequest(tt.method, path, http.NoBody)

			// response recorder
			w := httptest.NewRecorder()

			// context with mock store, stop test if failed to init context
			hctx, err := New(tt.store, 5)
			require.NoError(t, err, "new handler context error")

			// call the handler
			hctx.PingDB(w, r)

			// get recorded data
			res := w.Result()

			// read the response and close the body; stop test if failed to read body
			response := getResponseTextPayload(t, res)
			res.Body.Close()

			// assert wanted data
			assert.Equal(t, tt.want.statusCode, res.StatusCode)
			assert.Equal(t, tt.want.response, response)
		})
	}
}
