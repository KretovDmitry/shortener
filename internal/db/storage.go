package db

import (
	"github.com/pkg/errors"
)

type Store interface {
	SaveURL(ShortURL, OriginalURL) error
	RetrieveInitialURL(ShortURL) (OriginalURL, error)
}

var ErrURLNotFound = errors.New("URL not found")
