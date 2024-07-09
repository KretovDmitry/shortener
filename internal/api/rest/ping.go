package rest

import (
	"errors"
	"net/http"

	"github.com/KretovDmitry/shortener/internal/errs"
)

// GetPingDB checks the status of the database connection.
//
// Request:
//
//	GET /ping
func (h *Handler) GetPingDB(w http.ResponseWriter, r *http.Request) {
	// check request method
	if r.Method != http.MethodGet {
		// Yandex Practicum requires 400 Bad Request instead of 405 Method Not Allowed.
		h.textError(w, r.Method, errs.ErrInvalidRequest, http.StatusBadRequest)
		return
	}

	if err := h.store.Ping(r.Context()); err != nil {
		if errors.Is(err, errs.ErrDBNotConnected) {
			h.textError(w, "DB not connected", err, http.StatusInternalServerError)
			return
		}
		h.textError(w, "connection error", err, http.StatusInternalServerError)
		return
	}
}
