package db

type Storage interface {
	RetrieveInitialURL(ShortURL) (OriginalURL, error)
	SaveURL(ShortURL, OriginalURL) error
}

type (
	ShortURL    string
	OriginalURL string
)
