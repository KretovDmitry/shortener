package handler

import (
	"context"
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

type connectedStore struct {
	*db.InMemoryStore
}

func (s *connectedStore) Ping(context.Context) error {
	return nil
}

func TestGetPingDB(t *testing.T) {
	path := "/ping"

	type want struct {
		response   string
		statusCode int
	}

	tests := []struct {
		name  string
		store db.URLStorage
		want  want
	}{
		{
			name:  "connected test",
			store: &connectedStore{},
			want: want{
				statusCode: http.StatusOK,
				response:   "",
			},
		},
		{
			name:  "DB not connected",
			store: db.NewInMemoryStore(),
			want: want{
				statusCode: http.StatusInternalServerError,
				response: fmt.Sprintf(
					"%s: DB not connected", errs.ErrDBNotConnected,
				),
			},
		},
		{
			name:  "connection error",
			store: &brokenStore{},
			want: want{
				statusCode: http.StatusInternalServerError,
				response: fmt.Sprintf(
					"%s: connection error", errIntentionallyNotWorkingMethod,
				),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, path, http.NoBody)

			w := httptest.NewRecorder()

			handler, err := New(tt.store, logger.Get(), 5)
			require.NoError(t, err, "failed to init new handler")

			handler.GetPingDB(w, r)

			res := w.Result()

			response := getResponseTextPayload(t, res)
			require.NoError(t, res.Body.Close(), "failed close body")

			assert.Equal(t, tt.want.statusCode, res.StatusCode)
			assert.Equal(t, tt.want.response, response)
		})
	}
}

func TestGetPing_Method(t *testing.T) {
	path := "/ping"

	tests := []struct {
		name   string
		method string
	}{
		{"invalid method: put", http.MethodPut},
		{"invalid method: head", http.MethodHead},
		{"invalid method: post", http.MethodPost},
		{"invalid method: patch", http.MethodPatch},
		{"invalid method: trace", http.MethodTrace},
		{"invalid method: delete", http.MethodDelete},
		{"invalid method: connect", http.MethodConnect},
		{"invalid method: options", http.MethodOptions},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, path, http.NoBody)

			w := httptest.NewRecorder()

			handler, err := New(&connectedStore{}, logger.Get(), 5)
			require.NoError(t, err, "failed to init new handler")

			handler.GetPingDB(w, r)

			res := w.Result()

			response := getResponseTextPayload(t, res)
			require.NoError(t, res.Body.Close(), "failed close body")

			assert.Equal(t, http.StatusBadRequest, res.StatusCode)
			assert.Equal(t, textPlain, res.Header.Get(contentType))
			assert.Equal(t, fmt.Sprintf("%s: %s",
				errs.ErrInvalidRequest, tt.method), response)
		})
	}
}
