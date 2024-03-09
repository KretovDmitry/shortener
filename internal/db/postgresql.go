package db

import (
	"github.com/KretovDmitry/shortener/internal/logger"
	"github.com/KretovDmitry/shortener/pkg/client/postgresql"
	"go.uber.org/zap"
)

type repository struct {
	store  postgresql.Client
	logger *zap.Logger
}

func NewRepository(client postgresql.Client) *repository {
	return &repository{
		store:  client,
		logger: logger.Get(),
	}
}

func (r *repository) SaveURL(ShortURL, OriginalURL) error {
	return nil
}

func (r *repository) RetrieveInitialURL(ShortURL) (OriginalURL, error) {
	return OriginalURL(""), nil
}
