package handler

import (
	"context"
	"fmt"
	"net/http"

	"go.uber.org/zap"
)

func (h *handler) PingDB(w http.ResponseWriter, r *http.Request) {
	defer h.logger.Sync()

	// guard in case of future router switching
	if r.Method != http.MethodGet {
		h.logger.Info("got request with bad method", zap.String("method", r.Method))
		msg := `Only GET method is allowed`
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	if err := h.sqlStore.Ping(context.TODO()); err != nil {
		h.logger.Error("ping postgres", zap.Error(err))
		msg := fmt.Sprintf("Internal server error: %v", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
}
