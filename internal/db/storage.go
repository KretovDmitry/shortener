package db

import (
	"context"
	"fmt"

	"github.com/KretovDmitry/shortener/internal/config"
	"github.com/pkg/errors"
)

type URLStorage interface {
	Save(context.Context, *URL) error
	SaveAll(context.Context, []*URL) error
	Get(context.Context, ShortURL) (*URL, error)
	Ping(context.Context) error
}

var (
	ErrURLNotFound    = errors.New("URL not found")
	ErrDBNotConnected = errors.New("database not connected")
	ErrConflict       = errors.New("data conflict")
)

// NewStore creates a new instance of URLStorage based on the configuration
func NewStore(ctx context.Context) (URLStorage, error) {
	if config.DSN != "" {
		// create a new postgres store
		store, err := NewPostgresStore(ctx, config.DSN)
		if err != nil {
			return nil, fmt.Errorf("new postgres store: %w", err)
		}

		return store, nil
	}

	// create a new file storage combined with in memory storage
	store, err := NewFileStore(config.FileStorage.Path())
	if err != nil {
		return nil, fmt.Errorf("new file store: %w", err)
	}

	return store, nil
}
