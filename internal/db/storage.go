package db

import (
	"context"
	"fmt"

	"github.com/KretovDmitry/shortener/internal/config"
	"github.com/pkg/errors"
)

type Store interface {
	SaveURL(context.Context, *URLRecord) error
	RetrieveInitialURL(context.Context, ShortURL) (OriginalURL, error)
	Ping(context.Context) error
}

var (
	ErrURLNotFound    = errors.New("URL not found")
	ErrDBNotConnected = errors.New("database not connected")
)

func NewStore(ctx context.Context) (Store, error) {
	if config.DSN != "" {
		return NewPostgresStore(ctx, config.DSN)
	}

	store, err := NewFileStore(config.FileStorage.Path())
	if err != nil {
		return nil, fmt.Errorf("new file store: %w", err)
	}

	return store, nil
}
