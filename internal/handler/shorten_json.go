package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/KretovDmitry/shortener/internal/config"
	"github.com/KretovDmitry/shortener/internal/errs"
	"github.com/KretovDmitry/shortener/internal/jwt"
	"github.com/KretovDmitry/shortener/internal/models"
	"github.com/KretovDmitry/shortener/internal/models/user"
	"github.com/KretovDmitry/shortener/internal/shorturl"
	"github.com/asaskevich/govalidator"
	"go.uber.org/zap"
)

type (
	shortenJSONRequestPayload struct {
		URL string `json:"url"`
	}

	shortenJSONResponsePayload struct {
		Result  models.ShortURL `json:"result"`
		Success bool            `json:"success"`
		Message string          `json:"message"`
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
func (h *Handler) ShortenJSON(w http.ResponseWriter, r *http.Request) {
	// check request method
	if r.Method != http.MethodPost {
		// Yandex Practicum requires 400 Bad Request instead of 405 Method Not Allowed.
		h.textError(w, r.Method, errs.ErrInvalidRequest, http.StatusBadRequest)
		return
	}

	// check content type
	if h.IsApplicationJSONContentType(r) {
		h.shortenJSONError(w, r.Header.Get("Content-Type"), errs.ErrInvalidRequest, http.StatusBadRequest)
		return
	}

	// decode request body
	var payload shortenJSONRequestPayload
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.shortenJSONError(w, "failed to decode request", err, http.StatusInternalServerError)
		return
	}

	// check if URL is provided
	if len(payload.URL) == 0 {
		h.shortenJSONError(w, "URL is not provided", errs.ErrInvalidRequest, http.StatusBadRequest)
		return
	}

	// check if URL is a valid URL
	if !govalidator.IsURL(payload.URL) {
		h.shortenJSONError(w, "invalid URL", errs.ErrInvalidRequest, http.StatusBadRequest)
		return
	}

	// generate short URL
	generatedShortURL, err := shorturl.Generate(payload.URL)
	if err != nil {
		h.shortenJSONError(w, "failed to shorten url", err, http.StatusInternalServerError)
		return
	}

	user, ok := user.FromContext(r.Context())
	if !ok {
		h.shortenJSONError(w, "failed get user from context",
			errs.ErrUnauthorized, http.StatusInternalServerError)
	}

	newRecord := models.NewRecord(generatedShortURL, payload.URL, user.ID)

	// Build the JWT authentication token.
	authToken, err := jwt.BuildJWTString(user.ID, config.Secret, time.Duration(config.JWT))
	if err != nil {
		h.shortenJSONError(w, "failed to build JWT token", err, http.StatusInternalServerError)
		return
	}

	// save URL to database
	err = h.store.Save(r.Context(), newRecord)
	if err != nil && !errors.Is(err, errs.ErrConflict) {
		h.shortenJSONError(w, "failed to save to database", err, http.StatusInternalServerError)
		return
	}

	// Set the response headers and status code
	w.Header().Set("Content-Type", "application/json")
	switch {
	case errors.Is(err, errs.ErrConflict):
		w.WriteHeader(http.StatusConflict)
	default:
		w.WriteHeader(http.StatusCreated)
	}

	// Set the "Authorization" cookie with the JWT authentication token.
	http.SetCookie(w, &http.Cookie{
		Name:     "Authorization",
		Value:    authToken,
		Expires:  time.Now().Add(time.Duration(config.JWT)),
		HttpOnly: true,
	})

	// create response payload
	result := shortenJSONResponsePayload{Result: models.ShortURL(generatedShortURL), Success: true, Message: "OK"}

	// encode response body
	if err := json.NewEncoder(w).Encode(result); err != nil {
		h.logger.Error("failed to encode response", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// shortenJSONError is a helper function that sets the appropriate response
// headers and status code for errors returned by the ShortenJSON endpoint.
func (h *Handler) shortenJSONError(w http.ResponseWriter, message string, err error, code int) {
	if code >= 500 {
		h.logger.Error(message, zap.Error(err), zap.String("loc", caller(2)))
	} else {
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
