package handler

import (
	"errors"
	"net/http"

	"github.com/KretovDmitry/shortener/internal/models"
)

// PingDB checks the status of the database connection.
//
// Method: GET
func (h *Handler) PingDB(w http.ResponseWriter, r *http.Request) {
	defer h.logger.Sync()
	defer r.Body.Close()

	// check request method
	if r.Method != http.MethodGet {
		// Yandex Practicum requires 400 Bad Request instead of 405 Method Not Allowed.
		h.textError(w, "bad method: "+r.Method, ErrOnlyGETMethodIsAllowed, http.StatusBadRequest)
		return
	}

	if err := h.store.Ping(r.Context()); err != nil {
		if errors.Is(err, models.ErrDBNotConnected) {
			h.textError(w, "DB not connected", models.ErrDBNotConnected, http.StatusInternalServerError)
			return
		}
		h.textError(w, "connection error", err, http.StatusInternalServerError)
		return
	}
}
