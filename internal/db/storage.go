package db

import "github.com/pkg/errors"

type (
	ShortURL    string
	OriginalURL string
)

var ErrURLNotFound = errors.New("URL not found")
