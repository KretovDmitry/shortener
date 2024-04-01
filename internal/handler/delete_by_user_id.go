package handler

import (
	"encoding/json"
	"net/http"

	"github.com/KretovDmitry/shortener/internal/models"
	"github.com/KretovDmitry/shortener/internal/models/user"
	"go.uber.org/zap"
)

type deleteRequestPayload struct {
	Urls []models.ShortURL `json:"urls,omitempty"`
}

// DeleteByUserID deletes a list of shortened URLs owned by a specific user.
//
//	DELETE /api/user/urls
//
// ["6qxTVvsy", "RTfd56hn", "Jlfd67ds"]
//
//	HTTP/1.1 202 Accepted
//
// This endpoint requires the user to be authenticated.
func (h *handler) DeleteByUserID(w http.ResponseWriter, r *http.Request) {
	// Check the request method.
	if r.Method != http.MethodDelete {
		// Return a "Bad Request" error if the request method is not "DELETE".
		h.textError(w, "bad method: "+r.Method,
			ErrOnlyDeleteMethodIsAllowed, http.StatusBadRequest)
		return
	}

	// Extract the user from the request context.
	user, ok := user.FromContext(r.Context())
	if !ok {
		// Return an internal server error
		// if the user cannot be retrieved from the context.
		h.textError(w, "failed to get user from context",
			models.ErrInvalidDataType, http.StatusInternalServerError)
		return
	}

	// Decode the request body.
	var payload deleteRequestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		// Return an internal server error if the request body cannot be decoded.
		h.textError(w, "failed to decode request",
			err, http.StatusInternalServerError)
		return
	}

	h.logger.Info("got delete request", zap.Any("urls", payload))

	// Schedule deletion of the URLs.
	for _, shortURL := range payload.Urls {
		h.deleteURLsChan <- &models.URL{
			ShortURL: shortURL,
			UserID:   user.ID,
		}
	}

	// Return an "Accepted" status code.
	w.WriteHeader(http.StatusAccepted)
}
