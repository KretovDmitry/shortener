package db

import (
	"encoding/json"
	"fmt"

	"github.com/KretovDmitry/shortener/internal/config"
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
		ID:          "",
		ShortURL:    ShortURL(shortURL),
		OriginalURL: OriginalURL(originalURL),
	}
}
