package db

import (
	"errors"
	"sync"
)

var ErrNotFound = errors.New("URL not found")

type inMemoryStore struct {
	mu    sync.RWMutex
	store map[string]string
}

func NewInMemoryStore() Storage {
	var store Storage = &inMemoryStore{store: make(map[string]string)}
	return store
}

func (s *inMemoryStore) RetrieveInitialURL(key string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	url, found := s.store[key]
	if !found {
		return "", ErrNotFound
	}
	return url, nil
}

func (s *inMemoryStore) SaveURL(shortURL, url string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.store[shortURL] = url
	return nil
}
