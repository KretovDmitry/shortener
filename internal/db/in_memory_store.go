package db

import (
	"context"
	"fmt"
	"sync"
)

type inMemoryStore struct {
	mu    sync.RWMutex
	store map[ShortURL]URL
}

func NewInMemoryStore() *inMemoryStore {
	return &inMemoryStore{store: make(map[ShortURL]URL)}
}

func (s *inMemoryStore) Get(_ context.Context, sURL ShortURL) (*URL, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	record, found := s.store[sURL]
	if !found {
		return nil, fmt.Errorf("%s: %w", record.ShortURL, ErrURLNotFound)
	}

	return &record, nil
}

func (s *inMemoryStore) Save(u *URL) error {
	s.mu.Lock()
	s.store[u.ShortURL] = *u
	s.mu.Unlock()

	return nil
}

func (s *inMemoryStore) SaveAll(_ context.Context, u []*URL) error {
	s.mu.Lock()
	for _, u := range u {
		s.store[u.ShortURL] = *u
	}
	s.mu.Unlock()

	return nil
}
