package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/KretovDmitry/shortener/internal/db"
	"github.com/KretovDmitry/shortener/internal/shorturl"
	"github.com/asaskevich/govalidator"
	"go.uber.org/zap"
)

type (
	shortenJSONRequestPayload struct {
		URL string `json:"url"`
	}

	shortenJSONResponsePayload struct {
		Result  db.ShortURL `json:"result"`
		Success bool        `json:"success"`
		Message string      `json:"message"`
	}
)

// ShortenJSON handles the shortening of a long URL.
// The message field should be set to an error message if the shortening failed.
// Otherwise, success should be set to true and the result field should contain the shortened URL.
//
// Request:
//
//	POST /api/shorten
//	Content-Type: application/json
//	{
//	    "url": "https://example.com"
//	}
//
// Response:
//
//	HTTP/1.1 201 Created
//	Content-Type: application/json
//	{
//	    "result": "http://config.AddrToReturn/Base58{8}"
//		"success": true
//		"message": "OK"
//	}
func (h *handler) ShortenJSON(w http.ResponseWriter, r *http.Request) {
	defer h.logger.Sync()
	defer r.Body.Close()

	// check request method
	if r.Method != http.MethodPost {
		// Yandex Practicum requires 400 Bad Request instead of 405 Method Not Allowed.
		h.shortenJSONError(w, "bad method: "+r.Method, ErrOnlyPOSTMethodIsAllowed, http.StatusBadRequest)
		return
	}

	// check content type
	contentType := r.Header.Get("Content-Type")
	if strings.ToLower(strings.TrimSpace(contentType)) != "application/json" {
		h.shortenJSONError(w, "bad content-type: "+contentType,
			ErrOnlyApplicationJSONContentType, http.StatusBadRequest)
		return
	}

	// decode request body
	var payload shortenJSONRequestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.shortenJSONError(w, "failed to decode request", err, http.StatusInternalServerError)
		return
	}

	// check if URL is provided
	if len(payload.URL) == 0 {
		h.shortenJSONError(w, "url field is empty", ErrURLIsNotProvided, http.StatusBadRequest)
		return
	}

	// check if URL is a valid URL
	if !govalidator.IsURL(payload.URL) {
		h.shortenJSONError(w, "shorten url: "+payload.URL, ErrNotValidURL, http.StatusBadRequest)
		return
	}

	// generate short URL
	generatedShortURL, err := shorturl.Generate(payload.URL)
	if err != nil {
		h.shortenJSONError(w, "failed to shorten url: "+payload.URL, err, http.StatusInternalServerError)
		return
	}

	newRecord := db.NewRecord(generatedShortURL, payload.URL)

	// save URL to database
	err = h.store.Save(r.Context(), newRecord)
	if err != nil && !errors.Is(err, db.ErrConflict) {
		h.shortenJSONError(w, "failed to save to database: "+payload.URL, err, http.StatusInternalServerError)
		return
	}

	// Set the response headers and status code
	w.Header().Set("Content-Type", "application/json")
	switch {
	case errors.Is(err, db.ErrConflict):
		w.WriteHeader(http.StatusConflict)
	default:
		w.WriteHeader(http.StatusCreated)
	}

	// create response payload
	result := shortenJSONResponsePayload{Result: db.ShortURL(generatedShortURL), Success: true, Message: "OK"}

	// encode response body
	if err := json.NewEncoder(w).Encode(result); err != nil {
		h.logger.Error("failed to encode response", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// shortenJSONError is a helper function that sets the appropriate response
// headers and status code for errors returned by the ShortenJSON endpoint.
func (h *handler) shortenJSONError(w http.ResponseWriter, message string, err error, code int) {
	if code >= 500 {
		h.logger.Error(message, zap.Error(err), zap.String("loc", caller(2)))
	}
	if code >= 400 && code < 500 {
		h.logger.Info(message, zap.Error(err), zap.String("loc", caller(2)))
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	err = json.NewEncoder(w).Encode(shortenJSONResponsePayload{
		Success: false,
		Message: fmt.Sprintf("%s: %s", message, err),
	})
	if err != nil {
		h.logger.Error("failed to encode response", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
