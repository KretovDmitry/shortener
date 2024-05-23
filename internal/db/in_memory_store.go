package db

import (
	"context"
	"fmt"
	"sync"

	"github.com/KretovDmitry/shortener/internal/errs"
	"github.com/KretovDmitry/shortener/internal/models"
)

var _ URLStorage = (*InMemoryStore)(nil)

type InMemoryStore struct {
	store map[models.ShortURL]models.URL
	mu    sync.RWMutex
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{store: make(map[models.ShortURL]models.URL)}
}

func (s *InMemoryStore) Get(_ context.Context, sURL models.ShortURL) (*models.URL, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	record, found := s.store[sURL]
	if !found {
		return nil, fmt.Errorf("%s: %w", record.ShortURL, errs.ErrNotFound)
	}

	return &record, nil
}

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

func (s *InMemoryStore) Save(_ context.Context, u *models.URL) error {
	s.mu.Lock()
	if _, ok := s.store[u.ShortURL]; ok {
		return errs.ErrConflict
	}
	s.store[u.ShortURL] = *u
	s.mu.Unlock()

	return nil
}

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

func (s *InMemoryStore) Ping(_ context.Context) error {
	return errs.ErrDBNotConnected
}
