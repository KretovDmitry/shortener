// This package contains the data models for the shortener application.
package models

import (
	"github.com/google/uuid"
)

// ShortURL is a string that represents a shortened URL.
type ShortURL string

// OriginalURL is a string that represents the original URL.
type OriginalURL string

// URL is a struct that represents a URL record in the database.
// It contains the following fields:
//   - ID: a unique identifier for the URL record.
//   - ShortURL: the shortened URL.
//   - OriginalURL: the original URL.
//   - UserID: the ID of the user who created the URL record.
//   - IsDeleted: a boolean flag that indicates whether the URL record has been deleted.
type URL struct {
	ID          string      `json:"id"`
	ShortURL    ShortURL    `json:"short_url"`
	OriginalURL OriginalURL `json:"original_url"`
	UserID      string      `json:"user_id"`
	IsDeleted   bool        `json:"is_deleted" db:"is_deleted"`
}

// NewRecord is a function that creates a new URL record.
func NewRecord(shortURL, originalURL, userID string) *URL {
	return &URL{
		ID:          uuid.NewString(),
		ShortURL:    ShortURL(shortURL),
		OriginalURL: OriginalURL(originalURL),
		UserID:      userID,
	}
}
