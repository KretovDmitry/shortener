package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/KretovDmitry/shortener/internal/config"
)

type FileRecord struct {
	ShortURL    ShortURL    `json:"short_url"`
	OriginalURL OriginalURL `json:"original_url"`
}

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

func (p *Producer) WriteRecord(record *FileRecord) error {
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

func (c *Consumer) ReadRecord() (*FileRecord, error) {
	record := &FileRecord{}
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
			fileStore.cache.SaveURL(record.ShortURL, record.OriginalURL)
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

func (fs *fileStore) RetrieveInitialURL(sURL ShortURL) (OriginalURL, error) {
	return fs.cache.RetrieveInitialURL(sURL)
}

func (fs *fileStore) SaveURL(sURL ShortURL, url OriginalURL) error {
	savedURL, err := fs.cache.RetrieveInitialURL(sURL)
	if err != nil && !errors.Is(err, ErrURLNotFound) {
		return err
	}

	if savedURL == url {
		return nil
	}

	if !config.FileStorage.WriteRequired() {
		return fs.cache.SaveURL(sURL, url)
	}

	record := &FileRecord{
		ShortURL:    sURL,
		OriginalURL: url,
	}

	if err := fs.file.WriteRecord(record); err != nil {
		return fmt.Errorf("write record: %w", err)
	}

	return fs.cache.SaveURL(sURL, url)
}
