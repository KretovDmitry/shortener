package memstore

import (
	"context"
	"fmt"
	"sync"

	"github.com/KretovDmitry/shortener/internal/errs"
	"github.com/KretovDmitry/shortener/internal/models"
	"github.com/KretovDmitry/shortener/internal/repository"
)

var _ repository.URLStorage = (*URLRepository)(nil)

// URLRepository is an in-memory implementation of the URLStorage interface.
// It stores URLs in a map and provides methods to interact with the stored data.
// It is safe for concurrent use.
type URLRepository struct {
	// store is a map that stores the URLs.
	store map[models.ShortURL]models.URL
	// mu is a mutex that protects the store map from concurrent access.
	mu sync.RWMutex
}

// NewInMemoryStore creates a new instance of the InMemoryStore.
// It initializes an empty map to store the URLs.
func NewURLRepository() *URLRepository {
	return &URLRepository{store: make(map[models.ShortURL]models.URL)}
}

// Get retrieves a URL by its short URL.
// If the URL is not found, it returns ErrNotFound.
func (r *URLRepository) Get(_ context.Context, sURL models.ShortURL) (*models.URL, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	record, found := r.store[sURL]
	if !found {
		return nil, fmt.Errorf("%s: %w", record.ShortURL, errs.ErrNotFound)
	}

	return &record, nil
}

// GetAllByUserID retrieves all URLs belonging to a specific user.
// If no URLs are found for the specified user, it returns ErrNotFound.
func (r *URLRepository) GetAllByUserID(_ context.Context, userID string) ([]*models.URL, error) {
	r.mu.RLock()

	all := make([]*models.URL, 0)
	for _, record := range r.store {
		if record.UserID == userID {
			all = append(all, &record)
		}
	}

	r.mu.RUnlock()

	if len(all) == 0 {
		return nil, errs.ErrNotFound
	}

	return all, nil
}

// DeleteURLs deletes the specified URLs from the store.
// It marks the URLs as deleted and does not remove them from the store.
func (r *URLRepository) DeleteURLs(_ context.Context, urls ...*models.URL) error {
	r.mu.Lock()

	for _, url := range urls {
		for shortURL, record := range r.store {
			if record.UserID == url.UserID {
				record.IsDeleted = true
				r.store[shortURL] = record
				break
			}
		}
	}

	r.mu.Unlock()
	return nil
}

// Save saves a URL to the store.
// If a URL with the same short URL already exists in the store, it returns ErrConflict.
func (r *URLRepository) Save(_ context.Context, u *models.URL) error {
	r.mu.Lock()
	if _, ok := r.store[u.ShortURL]; ok {
		return errs.ErrConflict
	}
	r.store[u.ShortURL] = *u
	r.mu.Unlock()

	return nil
}

// SaveAll saves multiple URLs to the store.
// If a URL with the same short URL already exists in the store, it returns ErrConflict.
func (r *URLRepository) SaveAll(_ context.Context, u []*models.URL) error {
	r.mu.Lock()
	for _, u := range u {
		if _, ok := r.store[u.ShortURL]; ok {
			return errs.ErrConflict
		}
		r.store[u.ShortURL] = *u
	}
	r.mu.Unlock()

	return nil
}

// Ping is a placeholder method that returns an error
// indicating that the database is not connected [ErrDBNotConnected].
func (r *URLRepository) Ping(_ context.Context) error {
	return errs.ErrDBNotConnected
}
