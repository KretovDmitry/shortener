package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/KretovDmitry/shortener/internal/config"
	"github.com/google/uuid"
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

func (p *Producer) WriteRecord(record *URLRecord) error {
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

func (c *Consumer) ReadRecord() (*URLRecord, error) {
	record := &URLRecord{}
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
			fileStore.cache.SaveURL(record)
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

func (fs *fileStore) RetrieveInitialURL(_ context.Context, sURL ShortURL) (OriginalURL, error) {
	return fs.cache.RetrieveInitialURL(sURL)
}

func (fs *fileStore) SaveURL(_ context.Context, record *URLRecord) error {
	savedURL, err := fs.cache.RetrieveInitialURL(record.ShortURL)
	if err != nil && !errors.Is(err, ErrURLNotFound) {
		return err
	}

	if savedURL == record.OriginalURL {
		return nil
	}

	record.ID = uuid.New().String()

	if config.FileStorage.WriteRequired() {
		if err := fs.file.WriteRecord(record); err != nil {
			return fmt.Errorf("write record: %w", err)
		}
	}

	return fs.cache.SaveURL(record)
}

func (fs *fileStore) Ping(_ context.Context) error {
	return ErrDBNotConnected
}
