// Package db provides the interface and implementation of URL storage.
package db

import (
	"context"
	"fmt"

	"github.com/KretovDmitry/shortener/internal/config"
	"github.com/KretovDmitry/shortener/internal/models"
)

// Interface of the URL storage.
type URLStorage interface {
	// Save saves a single URL to the storage.
	Save(ctx context.Context, url *models.URL) error

	// SaveAll saves a slice of URLs to the storage.
	SaveAll(ctx context.Context, urls []*models.URL) error

	// Get retrieves a URL from the storage by its short URL.
	Get(ctx context.Context, shortURL models.ShortURL) (*models.URL, error)

	// GetAllByUserID retrieves all URLs for a specific user from the storage.
	GetAllByUserID(ctx context.Context, userID string) ([]*models.URL, error)

	// DeleteURLs deletes one or more URLs from the storage.
	DeleteURLs(ctx context.Context, urls ...*models.URL) error

	// Ping checks the health of the storage.
	Ping(ctx context.Context) error
}

// NewStore creates a new instance of URLStorage based on the configuration.
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
	store, err := NewFileStore(config.FS.Path())
	if err != nil {
		return nil, fmt.Errorf("new file store: %w", err)
	}

	return store, nil
}
