package handler

import (
	"encoding/json"
	"net/http"

	"github.com/KretovDmitry/shortener/internal/errs"
	"github.com/KretovDmitry/shortener/internal/models"
	"github.com/KretovDmitry/shortener/internal/models/user"
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

// PostShortenBatch handles requests to shorten multiple URLs in a single request.
//
// Request:
//
//	POST /api/shorten/batch
//	Content-Type: application/json
//
//	 [
//		{
//			"correlation_id": "42b4cb1b-abf0-44e7-89f9-72ad3a277e0a",
//			"original_url": "http://..."
//		},
//		{
//			"correlation_id": "229d9603-8540-4925-83f6-5cb1f239a72b",
//			"original_url": "http://..."
//		},
//		...
//	 ]
//
// Response:
//
//	HTTP/1.1 201 Created
//	Content-Type: application/json
//
//	[
//
//		{
//			"correlation_id": "42b4cb1b-abf0-44e7-89f9-72ad3a277e0a",
//			"short_url": "http://config.AddrToReturn/Base58{8}"
//		},
//		{
//			"correlation_id": "229d9603-8540-4925-83f6-5cb1f239a72b",
//			"short_url": "http://config.AddrToReturn/Base58{8}"
//		},
//		...
//	 ]
func (h *Handler) PostShortenBatch(w http.ResponseWriter, r *http.Request) {
	// check the request method
	if r.Method != http.MethodPost {
		// Yandex Practicum requires 400 Bad Request instead of 405 Method Not Allowed.
		h.textError(w, r.Method, errs.ErrInvalidRequest, http.StatusBadRequest)
		return
	}

	// check content type
	if !h.IsApplicationJSONContentType(r) {
		h.textError(w, r.Header.Get("Content-Type"), errs.ErrInvalidRequest, http.StatusBadRequest)
		return
	}

	// decode the request body
	var payload []shortenBatchRequestPayload
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.textError(w, err.Error(), errs.ErrInvalidRequest, http.StatusInternalServerError)
		return
	}

	// prepare the records to save and send
	recordsToSave := make([]*models.URL, len(payload))
	result := make([]shortenBatchResponsePayload, len(payload))

	user, ok := user.FromContext(r.Context())
	if !ok {
		h.textError(w, "no user found", errs.ErrUnauthorized, http.StatusUnauthorized)
		return
	}

	for i, p := range payload {

		// check if URL is provided
		if len(p.OriginalURL) == 0 {
			h.textError(w, "URL is not provided", errs.ErrInvalidRequest, http.StatusBadRequest)
			return
		}

		// check if URL is a valid URL
		if !govalidator.IsURL(p.OriginalURL) {
			h.textError(w, "invalid URL", errs.ErrInvalidRequest, http.StatusBadRequest)
			return
		}

		// generate short URL
		shortURL, err := shorturl.Generate(p.OriginalURL)
		if err != nil {
			h.textError(w, "failed to shorten url", err, http.StatusInternalServerError)
			return
		}

		recordsToSave[i] = models.NewRecord(shortURL, p.OriginalURL, user.ID)
		result[i] = shortenBatchResponsePayload{p.CorrelationID, models.ShortURL(shortURL)}
	}

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
