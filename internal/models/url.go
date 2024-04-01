package models

import (
	"encoding/json"
	"fmt"

	"github.com/KretovDmitry/shortener/internal/config"
	"github.com/google/uuid"
)

type (
	ShortURL    string
	OriginalURL string
	URL         struct {
		ID          string      `json:"id,omitempty"`
		ShortURL    ShortURL    `json:"short_url"`
		OriginalURL OriginalURL `json:"original_url"`
		UserID      string      `json:"user_id,omitempty"`
		IsDeleted   bool        `db:"is_deleted"`
	}
)

func (s ShortURL) MarshalJSON() ([]byte, error) {
	result := fmt.Sprintf("http://%s/%s", config.AddrToReturn, s)
	return json.Marshal(result)
}

func NewRecord(shortURL, originalURL, userID string) *URL {
	return &URL{
		ID:          uuid.NewString(),
		ShortURL:    ShortURL(shortURL),
		OriginalURL: OriginalURL(originalURL),
		UserID:      userID,
	}
}
