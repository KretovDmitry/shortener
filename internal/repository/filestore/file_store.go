package filestore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/KretovDmitry/shortener/internal/config"
	"github.com/KretovDmitry/shortener/internal/errs"
	"github.com/KretovDmitry/shortener/internal/models"
	"github.com/KretovDmitry/shortener/internal/repository/memstore"
)

// Producer is a struct that represents a producer for writing URL records to a file.
type Producer struct {
	// file is the underlying file handle for writing records.
	file *os.File
	// encoder is the JSON encoder used to write records to the file.
	encoder *json.Encoder
}

// NewProducer creates a new Producer instance for writing URL records to a file.
// It takes a filepath as input and returns a Producer instance
// along with any encountered errors.
func NewProducer(fileName string) (*Producer, error) {
	file, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o666)
	if err != nil {
		return nil, err
	}
	return &Producer{
		file:    file,
		encoder: json.NewEncoder(file),
	}, nil
}

// WriteRecord writes a URL record to the file using the JSON encoder.
func (p *Producer) WriteRecord(record *models.URL) error {
	return p.encoder.Encode(record)
}

// Consumer is a struct that represents a consumer for reading URL records from a file.
type Consumer struct {
	// file is the underlying file handle for reading records.
	file *os.File
	// decoder is the JSON decoder used to read records from the file.
	decoder *json.Decoder
}

// NewConsumer creates a new Consumer instance for reading URL records from a file.
// It takes a filepath as input and returns a Consumer instance
// along with any encountered errors.
func NewConsumer(fileName string) (*Consumer, error) {
	file, err := os.OpenFile(fileName, os.O_RDONLY|os.O_CREATE, 0o644)
	if err != nil {
		return nil, err
	}

	return &Consumer{
		file:    file,
		decoder: json.NewDecoder(file),
	}, nil
}

// ReadRecord reads a URL record from the file using the JSON decoder.
func (c *Consumer) ReadRecord() (*models.URL, error) {
	record := new(models.URL)
	if err := c.decoder.Decode(record); err != nil {
		return nil, err
	}

	return record, nil
}

// FileStore is a struct that represents a file-based storage system for URL records.
type FileStore struct {
	// cache is an in memory instance of URL repository
	// used for caching URL records.
	cache *memstore.URLRepository
	// file is a Producer instance used for writing URL records to the file.
	file *Producer
	// application configuration.
	config *config.Config
}

// NewFileStore creates a new fileStore instance for managing URL records in a file.
func NewFileStore(config *config.Config) (*FileStore, error) {
	if config == nil {
		return nil, fmt.Errorf("%w: config", errs.ErrNilDependency)
	}

	fileStore := &FileStore{
		cache:  memstore.NewURLRepository(),
		file:   nil,
		config: config,
	}

	consumer, err := NewConsumer(config.FileStoragePath)
	if err != nil {
		return nil, fmt.Errorf("new consumer: %w", err)
	}

	var record *models.URL

	for {
		record, err = consumer.ReadRecord()
		if record != nil {
			if err = fileStore.cache.Save(context.TODO(), record); err != nil {
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

	if fileStore.writeToFileRequired() {
		return fileStore, nil
	}

	producer, err := NewProducer(config.FileStoragePath)
	if err != nil {
		return nil, fmt.Errorf("new producer: %w", err)
	}

	fileStore.file = producer

	return fileStore, nil
}

// Get retrieves a URL record from the cache by its short URL.
func (fs *FileStore) Get(ctx context.Context, sURL models.ShortURL) (*models.URL, error) {
	return fs.cache.Get(ctx, sURL)
}

// GetAllByUserID retrieves all URL records belonging to a specific user from the cache.
func (fs *FileStore) GetAllByUserID(ctx context.Context, userID string) ([]*models.URL, error) {
	return fs.cache.GetAllByUserID(ctx, userID)
}

// DeleteURLs deletes all URL records belonging to a specific user from the cache.
func (fs *FileStore) DeleteURLs(ctx context.Context, urls ...*models.URL) error {
	return fs.cache.DeleteURLs(ctx, urls...)
}

// Save writes a URL record to the cache and file if required.
func (fs *FileStore) Save(ctx context.Context, url *models.URL) error {
	// check if the record already exists in the cache
	record, err := fs.cache.Get(ctx, url.ShortURL)
	if err != nil && !errors.Is(err, errs.ErrNotFound) {
		return err
	}
	// if the record already exists return ErrConflict
	if record != nil && record.OriginalURL == url.OriginalURL {
		return errs.ErrConflict
	}
	// write the record to the file if required
	if fs.writeToFileRequired() {
		if err = fs.file.WriteRecord(url); err != nil {
			return fmt.Errorf("write record: %w", err)
		}
	}
	// save the record to the cache if writing to the file was successful if required
	return fs.cache.Save(ctx, url)
}

// SaveAll saves multiple URL records to the cache and file if required.
func (fs *FileStore) SaveAll(ctx context.Context, urls []*models.URL) error {
	for _, url := range urls {
		// check if the record already exists in the cache
		record, err := fs.cache.Get(ctx, url.ShortURL)
		if err != nil && !errors.Is(err, errs.ErrNotFound) {
			return err
		}
		// if the record already exists skip the record
		if record != nil && record.OriginalURL == url.OriginalURL {
			continue
		}
		// write the record to the file if required
		if fs.writeToFileRequired() {
			if err = fs.file.WriteRecord(url); err != nil {
				return fmt.Errorf("write file record: %w", err)
			}
		}
		// save the record to the cache if writing to the file was successful if required
		if err = fs.cache.Save(ctx, url); err != nil {
			return fmt.Errorf("save record: %w", err)
		}
	}
	return nil
}

// Ping is a placeholder method that returns an error
// indicating that the database is not connected [ErrDBNotConnected].
func (fs *FileStore) Ping(context.Context) error {
	return errs.ErrDBNotConnected
}

// writeToFileRequired returns true if the application should save
// to the file, otherwise - false.
// According to the specification, writing to the file should be disabled
// if an empty path for storing the file is specified.
func (fs *FileStore) writeToFileRequired() bool {
	return fs.config.FileStoragePath != ""
}
