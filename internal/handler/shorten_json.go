package handler

import (
	"encoding/json"
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
		Result db.ShortURL `json:"result"`
	}
)

// ShortenJSON handles the shortening of a long URL.
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
//	}
func (h *handler) ShortenJSON(w http.ResponseWriter, r *http.Request) {
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

	// check content type
	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if contentType != "application/json" {
		h.logger.Info("got request with bad content-type",
			zap.String("content-type", r.Header.Get("Content-Type")))
		msg := `Only "application/json" Content-Type is allowed`
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	// decode request body
	var payload shortenJSONRequestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		msg := "failed decode request JSON body"
		h.logger.Error(msg, zap.Error(err))
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	// check if URL is provided
	if len(payload.URL) == 0 {
		msg := "The URL field in the JSON body of the request is empty"
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	// check if URL is a valid URL
	if !govalidator.IsURL(payload.URL) {
		msg := fmt.Sprintf("The provided string is not a URL: %s", payload.URL)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	// generate short URL
	generatedShortURL, err := shorturl.Generate(payload.URL)
	if err != nil {
		h.logger.Error("failed to generate short URL", zap.Error(err))
		msg := fmt.Sprintf("Internal server error: %s", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	// save URL to database
	newRecord := db.NewRecord(generatedShortURL, payload.URL)
	if err := h.store.Save(r.Context(), newRecord); err != nil {
		h.logger.Error("failed to save URLs", zap.Error(err))
		msg := fmt.Sprintf("Internal server error: %s", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	// set response headers
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	// create response payload
	result := shortenJSONResponsePayload{Result: db.ShortURL(generatedShortURL)}
	// encode response body
	if err := json.NewEncoder(w).Encode(result); err != nil {
		msg := "failed to encode the body of the JSON response"
		h.logger.Error(msg, zap.Error(err))
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
}
