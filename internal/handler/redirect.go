package handler

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"

	"github.com/KretovDmitry/shortener/internal/db"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

var Base58Regexp = regexp.MustCompile(`^[A-HJ-NP-Za-km-z1-9]{8}$`)

// Redirect serves a redirect to the original URL based on the shortened URL.
func (h *handler) Redirect(w http.ResponseWriter, r *http.Request) {
	defer h.logger.Sync()

	// check request method
	if r.Method != http.MethodGet {
		// Yandex Practicum technical specification requires
		// using a status code of 400 Bad Request instead of 405 Method Not Allowed.
		h.logger.Info("got request with bad method", zap.String("method", r.Method))
		http.Error(w, `Only POST method is allowed`, http.StatusBadRequest)
		return
	}

	shortURL := chi.URLParam(r, "shortURL")

	if !Base58Regexp.MatchString(shortURL) {
		h.logger.Info("requested invalid URL", zap.String("url", shortURL))
		http.Error(w, "Invalid URL: "+shortURL, http.StatusBadRequest)
		return
	}

	record, err := h.store.Get(r.Context(), db.ShortURL(shortURL))
	if err != nil {
		if errors.Is(err, db.ErrURLNotFound) {
			h.logger.Info("requested non-existent URL", zap.String("url", shortURL))
			http.Error(w, "No such URL: "+shortURL, http.StatusBadRequest)
			return
		}
		h.logger.Error("failed to retrieve initial URL", zap.Error(err))
		http.Error(w, fmt.Sprintf("Internal server error: %s", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Location", string(record.OriginalURL))
	w.WriteHeader(http.StatusTemporaryRedirect)
}
