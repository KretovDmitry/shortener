package db

import (
	"fmt"
	"sync"
)

type inMemoryStore struct {
	mu    sync.RWMutex
	store map[ShortURL]URLRecord
}

func NewInMemoryStore() *inMemoryStore {
	return &inMemoryStore{store: make(map[ShortURL]URLRecord)}
}

func (s *inMemoryStore) RetrieveInitialURL(sURL ShortURL) (OriginalURL, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	record, found := s.store[sURL]
	if !found {
		return "", fmt.Errorf("%w: %s", ErrURLNotFound, record.ShortURL)
	}

	return record.OriginalURL, nil
}

func (s *inMemoryStore) SaveURL(r *URLRecord) error {
	s.mu.Lock()
	s.store[r.ShortURL] = *r
	s.mu.Unlock()

	return nil
}
