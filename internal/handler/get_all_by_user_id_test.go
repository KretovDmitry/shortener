package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KretovDmitry/shortener/internal/db"
	"github.com/KretovDmitry/shortener/internal/errs"
	"github.com/KretovDmitry/shortener/internal/logger"
	"github.com/KretovDmitry/shortener/internal/models"
	"github.com/KretovDmitry/shortener/internal/models/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAllByUserID_Method(t *testing.T) {
	path := "/api/user/urls"

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

			r = r.WithContext(user.NewContext(r.Context(), &user.User{ID: "test"}))

			w := httptest.NewRecorder()

			handler, err := New(db.NewInMemoryStore(), logger.Get(), 5)
			require.NoError(t, err, "new handler error")

			handler.GetAllByUserID(w, r)

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

func TestGetAllByUserID_WithoutUserInContext(t *testing.T) {
	path := "/api/user/urls"

	r := httptest.NewRequest(http.MethodGet, path, http.NoBody)

	w := httptest.NewRecorder()

	handler, err := New(db.NewInMemoryStore(), logger.Get(), 5)
	require.NoError(t, err, "new handler error")

	handler.GetAllByUserID(w, r)

	res := w.Result()

	response := getResponseTextPayload(t, res)
	require.NoError(t, res.Body.Close(), "failed close body")

	assert.Equal(t, http.StatusUnauthorized, res.StatusCode,
		"status code mismatch")
	assert.Equal(t, fmt.Sprintf("%s: no user found", errs.ErrUnauthorized),
		response, "response message mismatch")
}

func TestGetAllByUserID_NoData(t *testing.T) {
	path := "/api/user/urls"

	r := httptest.NewRequest(http.MethodGet, path, http.NoBody)

	r = r.WithContext(user.NewContext(r.Context(), &user.User{ID: "test"}))

	w := httptest.NewRecorder()

	handler, err := New(db.NewInMemoryStore(), logger.Get(), 5)
	require.NoError(t, err, "new handler error")

	handler.GetAllByUserID(w, r)

	res := w.Result()

	response := getResponseTextPayload(t, res)
	require.NoError(t, res.Body.Close(), "failed close body")

	assert.Equal(t, http.StatusNoContent, res.StatusCode)
	assert.Equal(t, textPlain, res.Header.Get(contentType))
	assert.Equal(t, fmt.Sprintf("%s: nothing found", errs.ErrNotFound), response)
}

func TestGetAllByUserID_Data(t *testing.T) {
	path := "/api/user/urls"
	userID := "test"
	data := []*models.URL{
		{
			ID:          "some id 1",
			OriginalURL: "https://practicum.yandex.ru",
			ShortURL:    "TZqSKV4tcyE",
			UserID:      userID,
		},
		{
			ID:          "some id 2",
			OriginalURL: "https://go.dev",
			ShortURL:    "YBbxJEcQ9vq",
			UserID:      userID,
		},
	}

	r := httptest.NewRequest(http.MethodGet, path, http.NoBody)
	r.Header.Set(contentType, applicationJSON)

	r = r.WithContext(user.NewContext(r.Context(), &user.User{ID: userID}))

	w := httptest.NewRecorder()

	mocks := db.NewInMemoryStore()

	err := mocks.SaveAll(context.TODO(), data)
	require.NoError(t, err, "save failed")

	handler, err := New(mocks, logger.Get(), 5)
	require.NoError(t, err, "new handler error")

	handler.GetAllByUserID(w, r)

	res := w.Result()

	response := decodeAllByUserIDResponsePayload(t, res)
	require.NoError(t, res.Body.Close(), "failed close body")

	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, applicationJSON, res.Header.Get(contentType))

	all, err := mocks.GetAllByUserID(context.TODO(), "test")
	require.NoError(t, err, "in memory store: get all failed")

	assert.Equal(t, len(all), len(response), "response mismatch")
}

func decodeAllByUserIDResponsePayload(
	t *testing.T, r *http.Response,
) []getAllByUserIDResponsePayload {
	res := make([]getAllByUserIDResponsePayload, 0)
	err := json.NewDecoder(r.Body).Decode(&res)
	require.NoError(t, err, "failed to decode response JSON")
	require.NoError(t, r.Body.Close(), "failed to close body")
	return res
}
