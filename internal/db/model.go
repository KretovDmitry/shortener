package db

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/KretovDmitry/shortener/internal/config"
	"github.com/google/uuid"
)

var (
	ErrURLNotFound    = errors.New("URL not found")
	ErrDBNotConnected = errors.New("database not connected")
	ErrConflict       = errors.New("data conflict")
)

type (
	ShortURL    string
	OriginalURL string
	URL         struct {
		ID          string      `json:"id"`
		ShortURL    ShortURL    `json:"short_url"`
		OriginalURL OriginalURL `json:"original_url"`
	}
)

func (s ShortURL) MarshalJSON() ([]byte, error) {
	result := fmt.Sprintf("http://%s/%s", config.AddrToReturn, s)
	return json.Marshal(result)
}

func NewRecord(shortURL, originalURL string) *URL {
	return &URL{
		ID:          uuid.NewString(),
		ShortURL:    ShortURL(shortURL),
		OriginalURL: OriginalURL(originalURL),
	}
}
