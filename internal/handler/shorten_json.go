package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/KretovDmitry/shortener/internal/errs"
	"github.com/KretovDmitry/shortener/internal/jwt"
	"github.com/KretovDmitry/shortener/internal/models"
	"github.com/KretovDmitry/shortener/internal/models/user"
	"github.com/KretovDmitry/shortener/internal/shorturl"
	"github.com/asaskevich/govalidator"
)

type (
	shortenJSONRequestPayload struct {
		URL string `json:"url"`
	}

	shortenJSONResponsePayload struct {
		Result  string `json:"result"`
		Message string `json:"message"`
		Success bool   `json:"success"`
	}
)

// PostShortenJSON handles the shortening of a long URL.
// The message field should be set to an error message if the shortening failed.
// Otherwise, success should be set to true and the result field should contain the shortened URL.
//
// Request:
//
//	POST /api/shorten
//	Content-Type: application/json
//	{ "url": "https://example.com" }
//
// Response:
//
//	HTTP/1.1 201 Created
//	Content-Type: application/json
//	{
//		"result": "http://config.AddrToReturn/Base58"
//		"success": true
//		"message": "OK"
//	}
func (h *Handler) PostShortenJSON(w http.ResponseWriter, r *http.Request) {
	// check request method
	if r.Method != http.MethodPost {
		// Yandex Practicum requires 400 Bad Request instead of 405 Method Not Allowed.
		h.shortenJSONError(w, r.Method, errs.ErrInvalidRequest, http.StatusBadRequest)
		return
	}

	// check content type
	if !h.IsApplicationJSONContentType(r) {
		h.shortenJSONError(w, r.Header.Get("Content-Type"), errs.ErrInvalidRequest, http.StatusBadRequest)
		return
	}

	// decode request body
	var payload shortenJSONRequestPayload
	defer func() {
		if err := r.Body.Close(); err != nil {
			h.logger.Errorf("close body: %v", err)
		}
	}()
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
	shortURL := shorturl.Generate(payload.URL)

	user, ok := user.FromContext(r.Context())
	if !ok {
		h.shortenJSONError(w, "no user found", errs.ErrUnauthorized, http.StatusUnauthorized)
		return
	}

	newRecord := models.NewRecord(shortURL, payload.URL, user.ID)

	// Build the JWT authentication token.
	authToken, err := jwt.BuildJWTString(user.ID,
		h.config.JWT.SigningKey, h.config.JWT.Expiration)
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
		Expires:  time.Now().Add(h.config.JWT.Expiration),
		HttpOnly: true,
	})

	// create response payload
	s := fmt.Sprintf("http://%s/%s", h.config.HTTPServer.ReturnAddress, shortURL)
	result := shortenJSONResponsePayload{Result: s, Success: true, Message: "OK"}

	// encode response body
	if err = json.NewEncoder(w).Encode(result); err != nil {
		h.logger.Errorf("failed to encode response: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// shortenJSONError is a helper function that sets the appropriate response
// headers and status code for errors returned by the ShortenJSON endpoint.
func (h *Handler) shortenJSONError(w http.ResponseWriter, message string, err error, code int) {
	logger := h.logger.SkipCaller(1)
	if code >= http.StatusInternalServerError {
		logger.Errorf("%s: %s", message, err)
	} else {
		logger.Infof("%s: %s", message, err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	err = json.NewEncoder(w).Encode(shortenJSONResponsePayload{
		Success: false,
		Message: fmt.Sprintf("%s: %s", err, message),
	})
	if err != nil {
		h.logger.Errorf("failed to encode response: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
