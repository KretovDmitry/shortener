package handler

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/KretovDmitry/shortener/internal/config"
	"github.com/KretovDmitry/shortener/internal/errs"
	"github.com/KretovDmitry/shortener/internal/jwt"
	"github.com/KretovDmitry/shortener/internal/models"
	"github.com/KretovDmitry/shortener/internal/models/user"
	"github.com/KretovDmitry/shortener/internal/shorturl"
	"github.com/asaskevich/govalidator"
)

// PostShortenText handles the shortening of a long URL.
func (h *Handler) PostShortenText(w http.ResponseWriter, r *http.Request) {
	// check the request method
	if r.Method != http.MethodPost {
		// Yandex Practicum requires 400 Bad Request instead of 405 Method Not Allowed.
		h.textError(w, r.Method, errs.ErrInvalidRequest, http.StatusBadRequest)
		return
	}

	// Check the content type.
	if r.Header.Get("Content-Encoding") == "" && !h.IsTextPlainContentType(r) {
		h.textError(w, r.Header.Get("Content-Type"), errs.ErrInvalidRequest, http.StatusBadRequest)
		return
	}

	// Read the request body.
	defer func() {
		if err := r.Body.Close(); err != nil {
			h.logger.Errorf("close body: %v", err)
		}
	}()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.textError(w, "failed to read request body", err, http.StatusInternalServerError)
		return
	}

	// Check if the URL is provided.
	if len(body) == 0 {
		h.textError(w, "URL is not provided", errs.ErrInvalidRequest, http.StatusBadRequest)
		return
	}

	// Extract the original URL from the request body.
	originalURL := string(body)

	// Check if the URL is a valid URL.
	if !govalidator.IsURL(originalURL) {
		h.textError(w, "invalid URL", errs.ErrInvalidRequest, http.StatusBadRequest)
		return
	}

	// Generate the shortened URL.
	generatedShortURL := shorturl.Generate(originalURL)

	// Extract the user ID from the request context.
	user, ok := user.FromContext(r.Context())
	if !ok {
		h.textError(w, "no user found", errs.ErrUnauthorized, http.StatusUnauthorized)
		return
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
	if err != nil && !errors.Is(err, errs.ErrConflict) {
		h.textError(w, "failed to save to database", err, http.StatusInternalServerError)
		return
	}

	// Set the response headers and status code.
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
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

	// Write the response body.
	_, err = fmt.Fprintf(w, "http://%s/%s", config.AddrToReturn, generatedShortURL)
	if err != nil {
		h.logger.Errorf("failed to write response: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
