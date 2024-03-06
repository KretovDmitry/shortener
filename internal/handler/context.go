package handler

import (
	"errors"

	"github.com/KretovDmitry/shortener/internal/db"
)

type URLStore interface {
	SaveURL(db.ShortURL, db.OriginalURL) error
	RetrieveInitialURL(db.ShortURL) (db.OriginalURL, error)
}

type handlerContext struct {
	store URLStore
}

// NewHandlerContext constructs a new handlerContext,
// ensuring that the dependencies are valid values
func NewHandlerContext(store URLStore) (*handlerContext, error) {
	if store == nil {
		return nil, errors.New("nil store")
	}
	return &handlerContext{store: store}, nil
}
