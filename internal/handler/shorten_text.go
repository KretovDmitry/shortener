package handler

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/KretovDmitry/shortener/internal/config"
	"github.com/KretovDmitry/shortener/internal/jwt"
	"github.com/KretovDmitry/shortener/internal/models"
	"github.com/KretovDmitry/shortener/internal/models/user"
	"github.com/KretovDmitry/shortener/internal/shorturl"
	"github.com/asaskevich/govalidator"
	"go.uber.org/zap"
)

// ShortenText handles the shortening of a long URL.
func (h *Handler) ShortenText(w http.ResponseWriter, r *http.Request) {
	// Check the request method.
	if r.Method != http.MethodPost {
		// Yandex Practicum requires 400 Bad Request instead of 405 Method Not Allowed.
		h.textError(w, "bad method: "+r.Method, ErrOnlyPOSTMethodIsAllowed, http.StatusBadRequest)
		return
	}

	// Check the content type.
	contentType := r.Header.Get("Content-Type")
	if r.Header.Get("Content-Encoding") == "" && !isTextPlainContentType(contentType) {
		h.textError(w, "bad content-type: "+contentType, ErrOnlyTextPlainContentType, http.StatusBadRequest)
		return
	}

	// Read the request body.
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.textError(w, "failed to read request body", err, http.StatusInternalServerError)
		return
	}

	// Check if the URL is provided.
	if len(body) == 0 {
		h.textError(w, "body is empty", ErrURLIsNotProvided, http.StatusBadRequest)
		return
	}

	// Extract the original URL from the request body.
	originalURL := string(body)

	// Check if the URL is a valid URL.
	if !govalidator.IsURL(originalURL) {
		h.textError(w, "shorten url: "+originalURL, ErrNotValidURL, http.StatusBadRequest)
		return
	}

	// Generate the shortened URL.
	generatedShortURL, err := shorturl.Generate(originalURL)
	if err != nil {
		h.textError(w, "failed to shorten url: "+originalURL, err, http.StatusInternalServerError)
		return
	}

	// Extract the user ID from the request context.
	user, ok := user.FromContext(r.Context())
	if !ok {
		h.textError(w, "failed get user from context",
			models.ErrInvalidDataType, http.StatusInternalServerError)
	}

	// Create a new record with the generated short URL, original URL, and user ID.
	newRecord := models.NewRecord(generatedShortURL, originalURL, user.ID)

	// Build the JWT authentication token.
	authToken, err := jwt.BuildJWTString(user.ID, config.Secret, time.Duration(config.JWT))
	if err != nil {
		h.textError(w, "failed to build JWT token", err, http.StatusInternalServerError)
		return
	}

	// Save the record to the database.
	err = h.store.Save(r.Context(), newRecord)
	if err != nil && !errors.Is(err, models.ErrConflict) {
		h.textError(w, "failed to save to database: "+originalURL, err, http.StatusInternalServerError)
		return
	}

	// Set the response headers and status code.
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	switch {
	case errors.Is(err, models.ErrConflict):
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

	// Write the response body.
	_, err = fmt.Fprintf(w, "http://%s/%s", config.AddrToReturn, generatedShortURL)
	if err != nil {
		h.logger.Error("failed to write response", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// isTextPlainContentType returns true if the content type is text/plain.
func isTextPlainContentType(contentType string) bool {
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	if i := strings.Index(contentType, ";"); i > -1 {
		contentType = contentType[0:i]
	}
	return contentType == "text/plain"
}
