package db

import (
	"encoding/json"
	"io"
	"os"

	"github.com/pkg/errors"
)

type Record struct {
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

func (p *Producer) WriteRecord(record *Record) error {
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

func (c *Consumer) ReadRecord() (*Record, error) {
	record := &Record{}
	if err := c.decoder.Decode(&record); err != nil {
		return nil, err
	}

	return record, nil
}

func (c *Consumer) ReadAll(store map[ShortURL]OriginalURL) error {
	for {
		record, err := c.ReadRecord()
		if record != nil {
			store[record.ShortURL] = record.OriginalURL
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Wrap(err, "read record")
		}
	}

	return nil
}
