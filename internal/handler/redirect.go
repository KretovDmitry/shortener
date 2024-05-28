package handler

import (
	"errors"
	"net/http"
	"regexp"

	"github.com/KretovDmitry/shortener/internal/errs"
	"github.com/KretovDmitry/shortener/internal/models"
	"github.com/go-chi/chi/v5"
)

// Base58Regexp is a regular expression that matches a valid Base58-encoded string.
// It is used to validate the format of shortened URLs.
var Base58Regexp = regexp.MustCompile(`^[A-HJ-NP-Za-km-z1-9]{8}$`)

// Redirect serves a redirect to the original URL based on the shortened URL.
//
// Request:
//
//	GET /{shortURL}
//
// Response:
//
//	HTTP/1.1 307 Temporary Redirect
//	Header "Location" contains original url
func (h *Handler) Redirect(w http.ResponseWriter, r *http.Request) {
	// check request method
	if r.Method != http.MethodGet {
		// Yandex Practicum requires 400 Bad Request instead of 405 Method Not Allowed.
		h.textError(w, r.Method, errs.ErrInvalidRequest, http.StatusBadRequest)
		return
	}

	shortURL := chi.URLParam(r, "shortURL")

	// check if shortened URL is valid
	if !Base58Regexp.MatchString(shortURL) {
		h.textError(w, "invalid URL", errs.ErrInvalidRequest, http.StatusBadRequest)
		return
	}

	// get original URL
	record, err := h.store.Get(r.Context(), models.ShortURL(shortURL))
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			h.textError(w, "no such URL", errs.ErrNotFound, http.StatusBadRequest)
			return
		}
		h.textError(w, "failed to retrieve url", err, http.StatusInternalServerError)
		return
	}

	if record.IsDeleted {
		w.WriteHeader(http.StatusGone)
		return
	}

	// set redirect header
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Location", string(record.OriginalURL))
	w.WriteHeader(http.StatusTemporaryRedirect)
}
