package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KretovDmitry/shortener/internal/db"
	"github.com/KretovDmitry/shortener/internal/errs"
	"github.com/KretovDmitry/shortener/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPingDB(t *testing.T) {
	path := "/"

	type want struct {
		response   string
		statusCode int
	}

	tests := []struct {
		name   string
		method string
		store  db.URLStorage
		want   want
	}{
		{
			name:   "connected test",
			method: http.MethodGet,
			store:  &connectedStore{},
			want: want{
				statusCode: http.StatusOK,
				response:   "",
			},
		},
		{
			name:   "invalid method: method post",
			method: http.MethodPost,
			store:  db.NewInMemoryStore(),
			want: want{
				statusCode: http.StatusBadRequest,
				response:   fmt.Sprintf("%s: %s", errs.ErrInvalidRequest, http.MethodPost),
			},
		},
		{
			name:   "invalid method: method put",
			method: http.MethodPut,
			store:  db.NewInMemoryStore(),
			want: want{
				statusCode: http.StatusBadRequest,
				response:   fmt.Sprintf("%s: %s", errs.ErrInvalidRequest, http.MethodPut),
			},
		},
		{
			name:   "invalid method: method patch",
			method: http.MethodPatch,
			store:  db.NewInMemoryStore(),
			want: want{
				statusCode: http.StatusBadRequest,
				response:   fmt.Sprintf("%s: %s", errs.ErrInvalidRequest, http.MethodPatch),
			},
		},
		{
			name:   "invalid method: method delete",
			method: http.MethodDelete,
			store:  db.NewInMemoryStore(),
			want: want{
				statusCode: http.StatusBadRequest,
				response:   fmt.Sprintf("%s: %s", errs.ErrInvalidRequest, http.MethodDelete),
			},
		},
		{
			name:   "DB not connected",
			method: http.MethodGet,
			store:  db.NewInMemoryStore(),
			want: want{
				statusCode: http.StatusInternalServerError,
				response:   fmt.Sprintf("%s: DB not connected", errs.ErrDBNotConnected),
			},
		},
		{
			name:   "connection error",
			method: http.MethodGet,
			store:  &brokenStore{},
			want: want{
				statusCode: http.StatusInternalServerError,
				response:   fmt.Sprintf("%s: connection error", errIntentionallyNotWorkingMethod),
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
			handler, err := New(tt.store, logger.Get(), 5)
			require.NoError(t, err, "new handler context error")

			// call the handler
			handler.GetPingDB(w, r)

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
