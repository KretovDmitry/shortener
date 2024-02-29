package db

import (
	"encoding/json"
	"io"
	"os"

	"github.com/KretovDmitry/shortener/internal/cfg"
	"github.com/pkg/errors"
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
		return nil, errors.Wrap(err, "new consumer")
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
			return nil, errors.Wrap(err, "read record")
		}
	}

	if !cfg.FileStorage.WriteRequired() {
		return fileStore, nil
	}

	producer, err := NewProducer(filepath)
	if err != nil {
		return nil, errors.Wrap(err, "new producer")
	}

	fileStore.file = producer

	return fileStore, nil
}

func (fs *fileStore) RetrieveInitialURL(sURL ShortURL) (OriginalURL, error) {
	return fs.cache.RetrieveInitialURL(sURL)
}

func (fs *fileStore) SaveURL(sURL ShortURL, url OriginalURL) error {
	_, err := fs.cache.RetrieveInitialURL(sURL)
	if err != nil && err != ErrURLNotFound {
		return err
	}

	fs.cache.SaveURL(sURL, url)

	if !cfg.FileStorage.WriteRequired() {
		return nil
	}

	record := &FileRecord{
		ShortURL:    sURL,
		OriginalURL: url,
	}

	return fs.file.WriteRecord(record)
}
