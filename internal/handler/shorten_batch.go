package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/KretovDmitry/shortener/internal/db"
	"github.com/KretovDmitry/shortener/internal/shorturl"
	"go.uber.org/zap"
)

type (
	shortenBatchRequestPayload struct {
		CorrelationID string `json:"correlation_id"`
		OriginalURL   string `json:"original_url"`
	}

	shortenBatchResponsePayload struct {
		CorrelationID string      `json:"correlation_id"`
		ShortURL      db.ShortURL `json:"short_url"`
	}
)

// ShortenBatch handles requests to shorten multiple URLs in a single request.
//
// Request body:
// [
//
//	{
//		"correlation_id": "42b4cb1b-abf0-44e7-89f9-72ad3a277e0a",
//		"original_url": "http://nywha1.yandex/ovuaqasue6jd4"
//	},
//	{
//		"correlation_id": "229d9603-8540-4925-83f6-5cb1f239a72b",
//		"original_url": "http://wakkz7fjeuj.yandex/uwijfp9cbpn/a3hfjaww/x0mhpeq"
//	}
//
// ]
//
// Response body:
// [
//
//	{
//		"correlation_id": "42b4cb1b-abf0-44e7-89f9-72ad3a277e0a",
//		"short_url": "http://config.AddrToReturn/Base58{8}"
//	},
//	{
//		"correlation_id": "229d9603-8540-4925-83f6-5cb1f239a72b",
//		"short_url": "http://config.AddrToReturn/Base58{8}"
//	},
//
// ]
func (h *handler) ShortenBatch(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	defer h.logger.Sync()

	// Check the request method.
	if r.Method != http.MethodPost {
		// Yandex Practicum technical specification requires
		// using a status code of 400 Bad Request instead of 405 Method Not Allowed.
		h.logger.Info("got request with bad method", zap.String("method", r.Method))
		http.Error(w, `Only POST method is allowed`, http.StatusBadRequest)
		return
	}

	// Check the content type.
	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if contentType != "application/json" {
		h.logger.Info("got request with bad content-type",
			zap.String("content-type", r.Header.Get("Content-Type")))
		msg := `Only "application/json" Content-Type is allowed`
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	// Decode the request body.
	var payload []shortenBatchRequestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		msg := "failed to decode request JSON body"
		h.logger.Error(msg, zap.Error(err))
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	h.logger.Debug("received batch request", zap.Int("count", len(payload)))

	// Prepare the records to save and send.
	recordsToSave := make([]*db.URL, len(payload))
	result := make([]shortenBatchResponsePayload, len(payload))
	for i, p := range payload {
		// generate short URL
		shortURL, err := shorturl.Generate(p.OriginalURL)
		if err != nil {
			msg := fmt.Sprintf("failed to generate short URL: %s", p.OriginalURL)
			h.logger.Error(msg, zap.Error(err))
			http.Error(w, msg, http.StatusInternalServerError)
			return
		}

		recordsToSave[i] = db.NewRecord(shortURL, p.OriginalURL)
		result[i] = shortenBatchResponsePayload{p.CorrelationID, db.ShortURL(shortURL)}
	}

	h.logger.Debug("saving batch of records", zap.Int("count", len(recordsToSave)))

	// Save the records.
	if err := h.store.SaveAll(r.Context(), recordsToSave); err != nil {
		msg := "failed to save records"
		h.logger.Error(msg, zap.Error(err))
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	// Set the response headers and status code.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	// Encode the response body.
	if err := json.NewEncoder(w).Encode(result); err != nil {
		msg := "failed to encode the body of the JSON response"
		h.logger.Error(msg, zap.Error(err))
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
}
