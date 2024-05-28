package db

import (
	"context"
	"fmt"

	"github.com/KretovDmitry/shortener/internal/config"
	"github.com/KretovDmitry/shortener/internal/models"
)

// Interface of the URL storage.
type URLStorage interface {
	Save(context.Context, *models.URL) error
	SaveAll(context.Context, []*models.URL) error
	Get(context.Context, models.ShortURL) (*models.URL, error)
	GetAllByUserID(ctx context.Context, userID string) ([]*models.URL, error)
	DeleteURLs(ctx context.Context, urls ...*models.URL) error
	Ping(context.Context) error
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
