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
		// Yandex Practicum requires 400 Bad Request instead of 405 Method Not Allowed.
		h.shortenTextError(w, "bad method", ErrOnlyPOSTMethodIsAllowed, http.StatusBadRequest)
		return
	}

	// check if content type is valid
	if r.Header.Get("Content-Encoding") == "" && !isTextContentType(r.Header.Get("Content-Type")) {
		h.shortenTextError(w, "bad content-type", ErrOnlyTextContentType, http.StatusBadRequest)
		return
	}

	// read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.shortenTextError(w, "failed to read request body", err, http.StatusInternalServerError)
		return
	}

	// check if URL is provided
	if len(body) == 0 {
		h.shortenTextError(w, "body is empty", ErrURLIsNotProvided, http.StatusBadRequest)
		return
	}

	originalURL := string(body)

	// generate short URL
	generatedShortURL, err := shorturl.Generate(originalURL)
	if err != nil {
		h.shortenTextError(w, "failed to shorten url: "+originalURL, err, http.StatusInternalServerError)
		return
	}

	newRecord := db.NewRecord(generatedShortURL, originalURL)

	// save URL to database
	err = h.store.Save(r.Context(), newRecord)
	if err != nil && !errors.Is(err, db.ErrConflict) {
		h.shortenTextError(w, "failed to save to database", err, http.StatusInternalServerError)
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

func (h *handler) shortenTextError(w http.ResponseWriter, message string, err error, code int) {
	if code >= 500 {
		h.logger.Error(message, zap.Error(err))
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(code)
	_, err = w.Write([]byte(fmt.Sprintf("%s: %s", message, err)))
	if err != nil {
		h.logger.Error("failed to write response", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func isTextContentType(contentType string) bool {
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	if i := strings.Index(contentType, ";"); i > -1 {
		contentType = contentType[0:i]
	}
	return contentType == "text/plain"
}
