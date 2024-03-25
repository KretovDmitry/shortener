package handler

import (
	"errors"
	"net/http"
	"regexp"

	"github.com/KretovDmitry/shortener/internal/models"
	"github.com/go-chi/chi/v5"
)

var Base58Regexp = regexp.MustCompile(`^[A-HJ-NP-Za-km-z1-9]{8}$`)

// Redirect serves a redirect to the original URL based on the shortened URL.
func (h *handler) Redirect(w http.ResponseWriter, r *http.Request) {
	defer h.logger.Sync()
	defer r.Body.Close()

	// check request method
	if r.Method != http.MethodGet {
		// Yandex Practicum requires 400 Bad Request instead of 405 Method Not Allowed.
		h.textError(w, "bad method: "+r.Method, ErrOnlyGETMethodIsAllowed, http.StatusBadRequest)
		return
	}

	shortURL := chi.URLParam(r, "shortURL")

	// check if shortened URL is valid
	if !Base58Regexp.MatchString(shortURL) {
		h.textError(w, "redirect with url: "+shortURL, ErrNotValidURL, http.StatusBadRequest)
		return
	}

	// get original URL
	record, err := h.store.Get(r.Context(), models.ShortURL(shortURL))
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			h.textError(w, "redirect with url: "+shortURL, models.ErrNotFound, http.StatusBadRequest)
			return
		}
		h.textError(w, "failed to retrieve url: "+shortURL, err, http.StatusInternalServerError)
		return
	}

	// set redirect header
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Location", string(record.OriginalURL))
	w.WriteHeader(http.StatusTemporaryRedirect)
}
