package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/KretovDmitry/shortener/internal/config"
)

type Producer struct {
	file    *os.File
	encoder *json.Encoder
}

func NewProducer(fileName string) (*Producer, error) {
	file, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	return &Producer{
		file:    file,
		encoder: json.NewEncoder(file),
	}, nil
}

func (p *Producer) WriteRecord(record *URL) error {
	return p.encoder.Encode(&record)
}

type Consumer struct {
	file    *os.File
	decoder *json.Decoder
}

func NewConsumer(fileName string) (*Consumer, error) {
	file, err := os.OpenFile(fileName, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}

	return &Consumer{
		file:    file,
		decoder: json.NewDecoder(file),
	}, nil
}

func (c *Consumer) ReadRecord() (*URL, error) {
	record := new(URL)
	if err := c.decoder.Decode(&record); err != nil {
		return nil, err
	}

	return record, nil
}

type fileStore struct {
	cache *inMemoryStore
	file  *Producer
}

func NewFileStore(filepath string) (*fileStore, error) {
	fileStore := &fileStore{
		cache: NewInMemoryStore(),
		file:  nil,
	}

	consumer, err := NewConsumer(filepath)
	if err != nil {
		return nil, fmt.Errorf("new consumer: %w", err)
	}

	for {
		record, err := consumer.ReadRecord()
		if record != nil {
			if err = fileStore.cache.Save(record); err != nil {
				return nil, fmt.Errorf("save record: %w", err)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read record: %w", err)
		}
	}

	if !config.FileStorage.WriteRequired() {
		return fileStore, nil
	}

	producer, err := NewProducer(filepath)
	if err != nil {
		return nil, fmt.Errorf("new producer: %w", err)
	}

	fileStore.file = producer

	return fileStore, nil
}

func (fs *fileStore) Get(ctx context.Context, sURL ShortURL) (*URL, error) {
	return fs.cache.Get(ctx, sURL)
}

func (fs *fileStore) Save(ctx context.Context, url *URL) error {
	// check if the record already exists in the cache
	record, err := fs.cache.Get(ctx, url.ShortURL)
	if err != nil && !errors.Is(err, ErrURLNotFound) {
		return err
	}

	// if the record already exists return ErrConflict
	if record != nil && record.OriginalURL == url.OriginalURL {
		return ErrConflict
	}

	// write the record to the file if required
	if config.FileStorage.WriteRequired() {
		if err := fs.file.WriteRecord(url); err != nil {
			return fmt.Errorf("write record: %w", err)
		}
	}

	// save the record to the cache if writing to the file was successful if required
	return fs.cache.Save(url)
}

// SaveAll saves multiple URL records to the file and cache.
func (fs *fileStore) SaveAll(ctx context.Context, urls []*URL) error {
	for _, url := range urls {
		// check if the record already exists in the cache
		record, err := fs.cache.Get(ctx, url.ShortURL)
		if err != nil && !errors.Is(err, ErrURLNotFound) {
			return err
		}

		// if the record already exists skip the record
		if record != nil && record.OriginalURL == url.OriginalURL {
			continue
		}

		// write the record to the file if required
		if config.FileStorage.WriteRequired() {
			if err = fs.file.WriteRecord(url); err != nil {
				return fmt.Errorf("write file record: %w", err)
			}
		}

		// save the record to the cache if writing to the file was successful if required
		if err = fs.cache.Save(url); err != nil {
			return fmt.Errorf("save record: %w", err)
		}
	}

	return nil
}

// fileStore Ping method tells that the real database is not connected.
func (fs *fileStore) Ping(context.Context) error {
	return ErrDBNotConnected
}
