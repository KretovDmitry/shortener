package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/KretovDmitry/shortener/internal/models"
	"github.com/KretovDmitry/shortener/internal/shorturl"
	"github.com/asaskevich/govalidator"
	"go.uber.org/zap"
)

type (
	shortenBatchRequestPayload struct {
		CorrelationID string `json:"correlation_id"`
		OriginalURL   string `json:"original_url"`
	}

	shortenBatchResponsePayload struct {
		CorrelationID string          `json:"correlation_id"`
		ShortURL      models.ShortURL `json:"short_url"`
	}
)

// ShortenBatch handles requests to shorten multiple URLs in a single request.
//
// Request
//
//	POST /api/shorten/batch
//	Content-Type: application/json
//
// [
//
//	{
//		"correlation_id": "42b4cb1b-abf0-44e7-89f9-72ad3a277e0a",
//		"original_url": "http://..."
//	},
//	{
//		"correlation_id": "229d9603-8540-4925-83f6-5cb1f239a72b",
//		"original_url": "http://..."
//	}
//
// ]
//
// Response:
//
//	HTTP/1.1 201 Created
//	Content-Type: application/json
//
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

	// check the request method
	if r.Method != http.MethodPost {
		// Yandex Practicum requires 400 Bad Request instead of 405 Method Not Allowed.
		h.textError(w, "bad method: "+r.Method, ErrOnlyPOSTMethodIsAllowed, http.StatusBadRequest)
		return
	}

	// check content type
	contentType := r.Header.Get("Content-Type")
	if strings.ToLower(strings.TrimSpace(contentType)) != "application/json" {
		h.textError(w, "bad content-type: "+contentType, ErrOnlyApplicationJSONContentType, http.StatusBadRequest)
		return
	}

	// decode the request body
	var payload []shortenBatchRequestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.textError(w, "failed to decode request", err, http.StatusInternalServerError)
		return
	}

	h.logger.Debug("received batch request", zap.Int("count", len(payload)))

	// prepare the records to save and send
	recordsToSave := make([]*models.URL, len(payload))
	result := make([]shortenBatchResponsePayload, len(payload))
	userID, ok := r.Context().Value(models.UserIDCtxKey{}).(string)
	if !ok {
		h.textError(w, "could't assert user ID to string", models.ErrInvalidDataType, http.StatusInternalServerError)
	}

	for i, p := range payload {

		// check if URL is provided
		if len(p.OriginalURL) == 0 {
			h.textError(w, "url field is empty", ErrURLIsNotProvided, http.StatusBadRequest)
			return
		}

		// check if URL is a valid URL
		if !govalidator.IsURL(p.OriginalURL) {
			h.textError(w, "shorten url: "+p.OriginalURL, ErrNotValidURL, http.StatusBadRequest)
			return
		}

		// generate short URL
		shortURL, err := shorturl.Generate(p.OriginalURL)
		if err != nil {
			h.textError(w, "failed to shorten url: "+p.OriginalURL, err, http.StatusInternalServerError)
			return
		}

		recordsToSave[i] = models.NewRecord(shortURL, p.OriginalURL, userID)
		result[i] = shortenBatchResponsePayload{p.CorrelationID, models.ShortURL(shortURL)}
	}

	h.logger.Debug("saving batch of records", zap.Int("count", len(recordsToSave)))

	// save the records
	if err := h.store.SaveAll(r.Context(), recordsToSave); err != nil {
		h.textError(w, "failed to save to database", err, http.StatusInternalServerError)
		return
	}

	// set the response headers and status code
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	// encode the response body
	if err := json.NewEncoder(w).Encode(result); err != nil {
		h.logger.Error("failed to encode response", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
