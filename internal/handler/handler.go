package handler

import (
	"errors"

	"github.com/KretovDmitry/shortener/internal/db"
	"github.com/KretovDmitry/shortener/internal/logger"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type handler struct {
	store    db.Store
	logger   *zap.Logger
	sqlStore *pgxpool.Pool
}

// New constructs a new handlerContext,
// ensuring that the dependencies are valid values
func New(store db.Store, sqlStore *pgxpool.Pool) (*handler, error) {
	if store == nil {
		return nil, errors.New("nil store")
	}

	return &handler{
		store:    store,
		logger:   logger.Get(),
		sqlStore: sqlStore,
	}, nil
}
