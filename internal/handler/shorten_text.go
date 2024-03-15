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
	"github.com/asaskevich/govalidator"
	"go.uber.org/zap"
)

func (h *handler) ShortenText(w http.ResponseWriter, r *http.Request) {
	defer h.logger.Sync()
	defer r.Body.Close()

	// check request method
	if r.Method != http.MethodPost {
		// Yandex Practicum requires 400 Bad Request instead of 405 Method Not Allowed.
		h.textError(w, "bad method: "+r.Method, ErrOnlyPOSTMethodIsAllowed, http.StatusBadRequest)
		return
	}

	// check if content type is valid
	contentType := r.Header.Get("Content-Type")
	if r.Header.Get("Content-Encoding") == "" && !isTextPlainContentType(contentType) {
		h.textError(w, "bad content-type: "+contentType, ErrOnlyTextPlainContentType, http.StatusBadRequest)
		return
	}

	// read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.textError(w, "failed to read request body", err, http.StatusInternalServerError)
		return
	}

	// check if URL is provided
	if len(body) == 0 {
		h.textError(w, "body is empty", ErrURLIsNotProvided, http.StatusBadRequest)
		return
	}

	originalURL := string(body)

	// check if URL is a valid URL
	if !govalidator.IsURL(originalURL) {
		h.textError(w, "shorten url: "+originalURL, ErrNotValidURL, http.StatusBadRequest)
		return
	}

	// generate short URL
	generatedShortURL, err := shorturl.Generate(originalURL)
	if err != nil {
		h.textError(w, "failed to shorten url: "+originalURL, err, http.StatusInternalServerError)
		return
	}

	newRecord := db.NewRecord(generatedShortURL, originalURL)

	// save URL to database
	err = h.store.Save(r.Context(), newRecord)
	if err != nil && !errors.Is(err, db.ErrConflict) {
		h.textError(w, "failed to save to database: "+originalURL, err, http.StatusInternalServerError)
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
	_, err = w.Write([]byte(fmt.Sprintf("http://%s/%s", config.AddrToReturn, generatedShortURL)))
	if err != nil {
		h.logger.Error("failed to write response", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// isTextPlainContentType returns true if content type is text/plain
func isTextPlainContentType(contentType string) bool {
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	if i := strings.Index(contentType, ";"); i > -1 {
		contentType = contentType[0:i]
	}
	return contentType == "text/plain"
}
