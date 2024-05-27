package handler

import (
	"encoding/json"
	"net/http"

	"github.com/KretovDmitry/shortener/internal/errs"
	"github.com/KretovDmitry/shortener/internal/models"
	"github.com/KretovDmitry/shortener/internal/models/user"
	"go.uber.org/zap"
)

// DeleteByUserID deletes a list of shortened URLs owned by a specific user.
//
// Request:
//
//	DELETE /api/user/urls
//
//	{ urls: [ "6qxTVvsy", "RTfd56hn", "Jlfd67ds", ... ] }
//
// Response:
//
//	HTTP/1.1 202 Accepted
func (h *Handler) DeleteURLs(w http.ResponseWriter, r *http.Request) {
	// Check the request method.
	if r.Method != http.MethodDelete {
		// Return a "Bad Request" error if the request method is not "DELETE".
		h.textError(w, r.Method, errs.ErrInvalidRequest, http.StatusBadRequest)
		return
	}

	// Check content type.
	if !h.IsApplicationJSONContentType(r) {
		h.textError(w, r.Header.Get("Content-Type"), errs.ErrInvalidRequest, http.StatusBadRequest)
		return
	}

	// Extract the user from the request context.
	user, ok := user.FromContext(r.Context())
	if !ok {
		h.textError(w, "no user found", errs.ErrUnauthorized, http.StatusUnauthorized)
		return
	}

	// Decode the request body.
	var payload []models.ShortURL
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		// Return an internal server error if the request body cannot be decoded.
		h.textError(w, "failed to decode request",
			err, http.StatusInternalServerError)
		return
	}

	h.logger.Info("got delete request", zap.Any("urls", payload))

	// Schedule deletion of the URLs.
	for _, shortURL := range payload {
		h.deleteURLsChan <- &models.URL{
			ShortURL: shortURL,
			UserID:   user.ID,
		}
	}

	// Return an "Accepted" status code.
	w.WriteHeader(http.StatusAccepted)
}
