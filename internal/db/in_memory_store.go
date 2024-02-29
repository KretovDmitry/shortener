package db

import (
	"sync"
)

type inMemoryStore struct {
	mu    sync.RWMutex
	store map[ShortURL]OriginalURL
}

func NewInMemoryStore() *inMemoryStore {
	return &inMemoryStore{store: make(map[ShortURL]OriginalURL)}
}

func (s *inMemoryStore) RetrieveInitialURL(sURL ShortURL) (OriginalURL, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	url, found := s.store[sURL]
	if !found {
		return "", ErrURLNotFound
	}

	return url, nil
}

func (s *inMemoryStore) SaveURL(sURL ShortURL, url OriginalURL) error {
	s.mu.Lock()
	s.store[sURL] = url
	s.mu.Unlock()

	return nil
}
