package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/KretovDmitry/shortener/internal/models"
	"github.com/KretovDmitry/shortener/internal/models/user"
	"go.uber.org/zap"
)

// GetAllByUserID returns all URLs for a given user ID.
//
// Request:
//
//	GET /api/user/urls
//
// Response:
//
//	HTTP/1.1 200 OK
//	Content-Type: application/json
//
// [
//
//	{
//	    "short_url": "http://config.AddrToReturn/Base58{8}",
//	    "original_url": "http://..."
//	},
//	...
//
// ]
func (h *Handler) GetAllByUserID(w http.ResponseWriter, r *http.Request) {
	defer h.logger.Sync()
	defer r.Body.Close()

	// check request method
	if r.Method != http.MethodGet {
		// Yandex Practicum requires 400 Bad Request instead of 405 Method Not Allowed.
		h.textError(w, "bad method: "+r.Method, ErrOnlyGETMethodIsAllowed, http.StatusBadRequest)
		return
	}

	// Extract the user ID from the request context.
	user, ok := user.FromContext(r.Context())
	if !ok {
		h.textError(w, "failed get user from context",
			models.ErrInvalidDataType, http.StatusInternalServerError)
	}

	URLs, err := h.store.GetAllByUserID(r.Context(), user.ID)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			w.WriteHeader(http.StatusNoContent)
			h.logger.Info("No URLs found for user", zap.String("userID", user.ID))
			return
		}
		h.textError(w, "failed to get URLs", err, http.StatusInternalServerError)
		return
	}

	h.logger.Debug("URLs", zap.Any("URLs", URLs))

	// set the response header content type
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// encode response body
	if err := json.NewEncoder(w).Encode(URLs); err != nil {
		h.logger.Error("failed to encode response", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
