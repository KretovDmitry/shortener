package handler

import (
	"errors"

	"github.com/KretovDmitry/shortener/internal/db"
)

type handlerContext struct {
	store db.Storage
}

// NewHandlerContext constructs a new handlerContext,
// ensuring that the dependencies are valid values
func NewHandlerContext(store db.Storage) (*handlerContext, error) {
	if store == nil {
		return nil, errors.New("nil store")
	}
	return &handlerContext{store: store}, nil
}
