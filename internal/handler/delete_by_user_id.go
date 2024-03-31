package handler

import (
	"encoding/json"
	"net/http"

	"github.com/KretovDmitry/shortener/internal/models"
	"github.com/KretovDmitry/shortener/internal/models/user"
)

func (h *handler) DeleteByUserID(w http.ResponseWriter, r *http.Request) {
	defer h.logger.Sync()
	defer r.Body.Close()

	// check request method
	if r.Method != http.MethodDelete {
		// Yandex Practicum requires 400 Bad Request instead of 405 Method Not Allowed.
		h.textError(w, "bad method: "+r.Method, ErrOnlyGETMethodIsAllowed, http.StatusBadRequest)
		return
	}

	// Extract the user ID from the request context.
	user, ok := user.FromContext(r.Context())
	if !ok {
		h.textError(w, "could't assert user ID to string",
			models.ErrInvalidDataType, http.StatusInternalServerError)
	}

	// decode the request body
	var payload []models.ShortURL
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.textError(w, "failed to decode request", err, http.StatusInternalServerError)
		return
	}

	h.deleteURLsChan <- &models.URL{
		UserID: user.ID,
	}
}
