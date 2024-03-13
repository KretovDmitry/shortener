package handler

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/KretovDmitry/shortener/internal/config"
	"github.com/KretovDmitry/shortener/internal/db"
	"github.com/KretovDmitry/shortener/internal/shorturl"
	"go.uber.org/zap"
)

func (h *handler) ShortenText(w http.ResponseWriter, r *http.Request) {
	defer h.logger.Sync()

	if r.Header.Get("Content-Encoding") == "" {
		contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
		if i := strings.Index(contentType, ";"); i > -1 {
			contentType = contentType[0:i]
		}
		if contentType != "text/plain" {
			msg := `Only "text/plain" Content-Type is allowed`
			http.Error(w, msg, http.StatusBadRequest)
			return
		}
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("failed to read request body", zap.Error(err))
		msg := fmt.Sprintf("Internal server error: %s", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
	r.Body.Close()

	if len(body) == 0 {
		http.Error(w, "Empty body, must contain URL", http.StatusBadRequest)
		return
	}

	originalURL := string(body)

	generatedShortURL, err := shorturl.Generate(originalURL)
	if err != nil {
		h.logger.Error("failed to generate short URL", zap.Error(err))
		msg := fmt.Sprintf("Internal server error: %s", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	newRecord := db.NewRecord(generatedShortURL, originalURL)

	if err := h.store.Save(r.Context(), newRecord); err != nil {
		h.logger.Error("failed to save URL", zap.Error(err))
		msg := fmt.Sprintf("Internal server error: %s", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(fmt.Sprintf("http://%s/%s", config.AddrToReturn, generatedShortURL)))
}
