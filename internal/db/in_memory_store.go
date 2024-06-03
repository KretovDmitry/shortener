package db

import (
	"context"
	"fmt"
	"sync"

	"github.com/KretovDmitry/shortener/internal/errs"
	"github.com/KretovDmitry/shortener/internal/models"
)

var _ URLStorage = (*InMemoryStore)(nil)

// InMemoryStore is an in-memory implementation of the URLStorage interface.
// It stores URLs in a map and provides methods to interact with the stored data.
// It is safe for concurrent use.
type InMemoryStore struct {
	// store is a map that stores the URLs.
	store map[models.ShortURL]models.URL
	// mu is a mutex that protects the store map from concurrent access.
	mu sync.RWMutex
}

// NewInMemoryStore creates a new instance of the InMemoryStore.
// It initializes an empty map to store the URLs.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{store: make(map[models.ShortURL]models.URL)}
}

// Get retrieves a URL by its short URL.
// If the URL is not found, it returns ErrNotFound.
func (s *InMemoryStore) Get(_ context.Context, sURL models.ShortURL) (*models.URL, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	record, found := s.store[sURL]
	if !found {
		return nil, fmt.Errorf("%s: %w", record.ShortURL, errs.ErrNotFound)
	}

	return &record, nil
}

// GetAllByUserID retrieves all URLs belonging to a specific user.
// If no URLs are found for the specified user, it returns ErrNotFound.
func (s *InMemoryStore) GetAllByUserID(_ context.Context, userID string) ([]*models.URL, error) {
	s.mu.RLock()

	all := make([]*models.URL, 0)
	for _, record := range s.store {
		if record.UserID == userID {
			all = append(all, &record)
		}
	}

	s.mu.RUnlock()

	if len(all) == 0 {
		return nil, errs.ErrNotFound
	}

	return all, nil
}

// DeleteURLs deletes the specified URLs from the store.
// It marks the URLs as deleted and does not remove them from the store.
func (s *InMemoryStore) DeleteURLs(_ context.Context, urls ...*models.URL) error {
	s.mu.Lock()

	for _, url := range urls {
		for shortURL, record := range s.store {
			if record.UserID == url.UserID {
				record.IsDeleted = true
				s.store[shortURL] = record
				break
			}
		}
	}

	s.mu.Unlock()
	return nil
}

// Save saves a URL to the store.
// If a URL with the same short URL already exists in the store, it returns ErrConflict.
func (s *InMemoryStore) Save(_ context.Context, u *models.URL) error {
	s.mu.Lock()
	if _, ok := s.store[u.ShortURL]; ok {
		return errs.ErrConflict
	}
	s.store[u.ShortURL] = *u
	s.mu.Unlock()

	return nil
}

// SaveAll saves multiple URLs to the store.
// If a URL with the same short URL already exists in the store, it returns ErrConflict.
func (s *InMemoryStore) SaveAll(_ context.Context, u []*models.URL) error {
	s.mu.Lock()
	for _, u := range u {
		if _, ok := s.store[u.ShortURL]; ok {
			return errs.ErrConflict
		}
		s.store[u.ShortURL] = *u
	}
	s.mu.Unlock()

	return nil
}

// Ping is a placeholder method that returns an error
// indicating that the database is not connected [ErrDBNotConnected].
func (s *InMemoryStore) Ping(_ context.Context) error {
	return errs.ErrDBNotConnected
}
