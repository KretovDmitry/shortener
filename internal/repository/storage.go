// Package repository provides the interfaces of storage.
package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/KretovDmitry/shortener/internal/config"
	"github.com/KretovDmitry/shortener/internal/errs"
	"github.com/KretovDmitry/shortener/internal/logger"
	"github.com/KretovDmitry/shortener/internal/models"
	"github.com/KretovDmitry/shortener/internal/repository/filestore"
	"github.com/KretovDmitry/shortener/internal/repository/postgres"
	"github.com/KretovDmitry/shortener/migrations"
	sqldblogger "github.com/simukti/sqldb-logger"
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

// NewURLStore returns one of the URLStorage implementations based on
// the configuration. Could be in memory, file storage or postgres.
func NewURLStore(config *config.Config, logger logger.Logger) (URLStorage, error) {
	// Check for dependencies that can lead to panic.
	if config == nil {
		return nil, fmt.Errorf("%w: config", errs.ErrNilDependency)
	}

	// Init postgres URL repository if DSN is provided.
	if config.DSN != "" {
		// Connect to the postgres.
		db, err := sql.Open("pgx", config.DSN)
		if err != nil {
			return nil, fmt.Errorf("failed to open the database: %w", err)
		}

		// Log every query to the database.
		db = sqldblogger.OpenDriver(config.DSN, db.Driver(), logger)

		// Check connectivity and DSN correctness.
		if err = db.Ping(); err != nil {
			return nil, fmt.Errorf("failed to connect to the database: %w", err)
		}

		// Up all migrations for github tests.
		err = migrations.Up(db)
		if err != nil {
			return nil, fmt.Errorf("failed to migrate DB: %w", err)
		}

		return postgres.NewURLRepository(db, logger)
	}

	logger.Info("DSN is not provided, initializing file storage")

	store, err := filestore.NewFileStore(config)
	if err != nil {
		return nil, fmt.Errorf("new file repository: %w", err)
	}

	if config.FileStoragePath != "" {
		logger.Infof("file storage initialaized at: %q",
			config.FileStoragePath)
	} else {
		logger.Info("file storage path isn't set, using in memory storage")
	}

	return store, nil
}
