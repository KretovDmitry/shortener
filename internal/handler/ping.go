package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/KretovDmitry/shortener/internal/db"
	"go.uber.org/zap"
)

// PingDB checks the status of the database connection.
//
// Method: GET
func (h *handler) PingDB(w http.ResponseWriter, r *http.Request) {
	defer h.logger.Sync()

	// check request method
	if r.Method != http.MethodGet {
		// Yandex Practicum technical specification requires
		// using a status code of 400 Bad Request instead of 405 Method Not Allowed.
		h.logger.Info("got request with bad method", zap.String("method", r.Method))
		http.Error(w, `Only GET method is allowed`, http.StatusBadRequest)
		return
	}

	if err := h.store.Ping(r.Context()); err != nil {
		if !errors.Is(err, db.ErrDBNotConnected) {
			h.logger.Error("ping postgres", zap.Error(err))
		}
		msg := fmt.Sprintf("Internal server error: %v", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
}
