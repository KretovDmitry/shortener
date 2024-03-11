package db

type (
	ShortURL    string
	OriginalURL string
	URLRecord   struct {
		ID          string      `json:"id"`
		ShortURL    ShortURL    `json:"short_url"`
		OriginalURL OriginalURL `json:"original_url"`
	}
)

func NewRecord(shortURL, originalURL string) *URLRecord {
	return &URLRecord{
		ID:          "",
		ShortURL:    ShortURL(shortURL),
		OriginalURL: OriginalURL(originalURL),
	}
}
