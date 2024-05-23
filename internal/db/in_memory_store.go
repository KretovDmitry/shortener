package db

import (
	"context"
	"fmt"
	"sync"

	"github.com/KretovDmitry/shortener/internal/errs"
	"github.com/KretovDmitry/shortener/internal/models"
)

type inMemoryStore struct {
	mu    sync.RWMutex
	store map[models.ShortURL]models.URL
}

func NewInMemoryStore() *inMemoryStore {
	return &inMemoryStore{store: make(map[models.ShortURL]models.URL)}
}

func (s *inMemoryStore) Get(_ context.Context, sURL models.ShortURL) (*models.URL, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	record, found := s.store[sURL]
	if !found {
		return nil, fmt.Errorf("%s: %w", record.ShortURL, errs.ErrNotFound)
	}

	return &record, nil
}

func (s *inMemoryStore) GetAllByUserID(_ context.Context, userID string) ([]*models.URL, error) {
	s.mu.RLock()

	all := make([]*models.URL, 0)
	for _, record := range s.store {
		if record.UserID == userID {
			all = append(all, &record)
		}
	}

	s.mu.RUnlock()
	return all, nil
}

func (s *inMemoryStore) DeleteURLs(_ context.Context, urls ...*models.URL) error {
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

func (s *inMemoryStore) Save(u *models.URL) error {
	s.mu.Lock()
	s.store[u.ShortURL] = *u
	s.mu.Unlock()

	return nil
}

func (s *inMemoryStore) SaveAll(_ context.Context, u []*models.URL) error {
	s.mu.Lock()
	for _, u := range u {
		s.store[u.ShortURL] = *u
	}
	s.mu.Unlock()

	return nil
}
