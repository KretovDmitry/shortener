package db

import (
	"sync"

	"github.com/KretovDmitry/shortener/internal/cfg"
	"github.com/pkg/errors"
)

var ErrURLNotFound = errors.New("URL not found")

type inMemoryStore struct {
	mu         sync.RWMutex
	store      map[ShortURL]OriginalURL
	fileWriter *Producer
}

func NewInMemoryStore() (*inMemoryStore, error) {
	store := make(map[ShortURL]OriginalURL)
	var producer *Producer

	consumer, err := NewConsumer(cfg.FileStorage.Path())
	if err != nil {
		return nil, errors.Wrap(err, "new consumer")
	}

	err = consumer.ReadAll(store)
	if err != nil {
		return nil, errors.Wrap(err, "read all")
	}

	if cfg.FileStorage.Required() {
		producer, err = NewProducer(cfg.FileStorage.Path())
		if err != nil {
			return nil, errors.Wrap(err, "new producer")
		}
	}

	return &inMemoryStore{
		store:      store,
		fileWriter: producer,
	}, nil
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
	_, found := s.store[sURL]
	if !found {
		s.store[sURL] = url
	}
	s.mu.Unlock()

	if !found && cfg.FileStorage.Required() {
		record := &Record{
			ShortURL:    sURL,
			OriginalURL: url,
		}
		err := s.fileWriter.WriteRecord(record)
		if err != nil {
			return errors.Wrap(err, "write record")
		}
	}

	return nil
}
