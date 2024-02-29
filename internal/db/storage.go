package db

import "github.com/pkg/errors"

type Storage interface {
	RetrieveInitialURL(ShortURL) (OriginalURL, error)
	SaveURL(ShortURL, OriginalURL) error
}

type (
	ShortURL    string
	OriginalURL string
)

var ErrURLNotFound = errors.New("URL not found")
