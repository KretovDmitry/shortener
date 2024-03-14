package handler

import (
	"errors"
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
	defer r.Body.Close()

	// check request method
	if r.Method != http.MethodPost {
		// Yandex Practicum technical specification requires
		// using a status code of 400 Bad Request instead of 405 Method Not Allowed.
		h.logger.Info("got request with bad method", zap.String("method", r.Method))
		http.Error(w, `Only POST method is allowed`, http.StatusBadRequest)
		return
	}

	// check if content type is valid
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

	// read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("failed to read request body", zap.Error(err))
		msg := fmt.Sprintf("Internal server error: %s", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	// check if URL is provided
	if len(body) == 0 {
		http.Error(w, "Empty body, must contain URL", http.StatusBadRequest)
		return
	}

	originalURL := string(body)

	// generate short URL
	generatedShortURL, err := shorturl.Generate(originalURL)
	if err != nil {
		msg := fmt.Sprintf("failed to generate short URL: %s", originalURL)
		h.logger.Error(msg, zap.Error(err))
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	newRecord := db.NewRecord(generatedShortURL, originalURL)

	// save URL to database
	err = h.store.Save(r.Context(), newRecord)
	if err != nil && !errors.Is(err, db.ErrConflict) {
		h.logger.Error("failed to save URLs", zap.Error(err))
		msg := fmt.Sprintf("Internal server error: %s", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	// Set the response headers and status code
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	switch {
	case errors.Is(err, db.ErrConflict):
		w.WriteHeader(http.StatusConflict)
	default:
		w.WriteHeader(http.StatusCreated)
	}

	// write response body
	w.Write([]byte(fmt.Sprintf("http://%s/%s", config.AddrToReturn, generatedShortURL)))
}
