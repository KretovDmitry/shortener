package handler

import (
	"encoding/json"
	"net/http"

	"github.com/KretovDmitry/shortener/internal/errs"
	"go.uber.org/zap"
)

type getStatsResponse struct {
	URLs  int `json:"urls"`  // number of all shortened urls
	Users int `json:"users"` // number of all users
}

// GetStats reveals total number of users and shortened urls in JSON format.
func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	// check request method
	if r.Method != http.MethodGet {
		// Yandex Practicum requires 400 Bad Request instead of 405 Method Not Allowed.
		h.textError(w, r.Method, errs.ErrInvalidRequest, http.StatusBadRequest)
		return
	}

	var response getStatsResponse

	count, err := h.store.CountShortURLs(r.Context())
	if err != nil {
		h.textError(w, "count urls", err, http.StatusInternalServerError)
		return
	}

	response.URLs = count

	count, err = h.store.CountUsers(r.Context())
	if err != nil {
		h.textError(w, "count users", err, http.StatusInternalServerError)
		return
	}

	response.Users = count

	// set the response headers and status code.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// encode the response body.
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("failed to encode response", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
