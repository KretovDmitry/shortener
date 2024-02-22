package db

import (
	"errors"
	"sync"
)

var ErrURLNotFound = errors.New("URL not found")

type inMemoryStore struct {
	mu    sync.RWMutex
	store map[string]string
}

func NewInMemoryStore() *inMemoryStore {
	return &inMemoryStore{store: make(map[string]string)}
}

func (s *inMemoryStore) RetrieveInitialURL(key string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	url, found := s.store[key]
	if !found {
		return "", ErrURLNotFound
	}
	return url, nil
}

func (s *inMemoryStore) SaveURL(shortURL, url string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.store[shortURL] = url
	return nil
}
