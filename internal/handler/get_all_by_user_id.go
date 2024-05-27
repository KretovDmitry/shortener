package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/KretovDmitry/shortener/internal/errs"
	"github.com/KretovDmitry/shortener/internal/models"
	"github.com/KretovDmitry/shortener/internal/models/user"
	"go.uber.org/zap"
)

type getAllByUserIDResponsePayload struct {
	ShortURL    models.ShortURL    `json:"short_url"`
	OriginalURL models.OriginalURL `json:"original_url"`
}

// GetAllByUserID returns shortened and original URLs for a given user ID.
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
//	[
//		{
//		    "short_url": "http://config.AddrToReturn/Base58{8}",
//		    "original_url": "http://..."
//		},
//		...
//	]
func (h *Handler) GetAllByUserID(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	// check request method
	if r.Method != http.MethodGet {
		// Yandex Practicum requires 400 Bad Request instead of 405 Method Not Allowed.
		h.textError(w, r.Method, errs.ErrInvalidRequest, http.StatusBadRequest)
		return
	}

	// Extract the user ID from the request context.
	user, ok := user.FromContext(r.Context())
	if !ok {
		h.textError(w, "no user found", errs.ErrUnauthorized, http.StatusUnauthorized)
		return
	}

	URLs, err := h.store.GetAllByUserID(r.Context(), user.ID)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			h.textError(w, "nothing found", err, http.StatusNoContent)
			return
		}
		h.textError(w, "failed to get URLs", err, http.StatusInternalServerError)
		return
	}

	response := make([]getAllByUserIDResponsePayload, len(URLs))
	for i, u := range URLs {
		response[i].ShortURL = u.ShortURL
		response[i].OriginalURL = u.OriginalURL
	}

	// set the response header content type
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// encode response body
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("failed to encode response", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
