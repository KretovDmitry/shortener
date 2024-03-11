package handler

import (
	"errors"

	"github.com/KretovDmitry/shortener/internal/db"
	"github.com/KretovDmitry/shortener/internal/logger"
	"github.com/KretovDmitry/shortener/internal/middleware"
	"github.com/KretovDmitry/shortener/pkg/gzip"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type handler struct {
	store  db.Store
	logger *zap.Logger
}

// New constructs a new handlerContext,
// ensuring that the dependencies are valid values
func New(store db.Store) (*handler, error) {
	if store == nil {
		return nil, errors.New("nil store")
	}

	return &handler{
		store:  store,
		logger: logger.Get(),
	}, nil
}

// Register sets up the routes for the HTTP server.
func (h *handler) Register(r *chi.Mux) {
	// Build middleware chain for request handling.
	chain := middleware.BuildChain(
		gzip.DefaultHandler().WrapHandler,
		middleware.RequestLogger,
		gzip.Unzip,
	)

	// Register routes.
	r.Post("/", chain(h.ShortenText))
	r.Post("/api/shorten", chain(h.ShortenJSON))
	r.Get("/{shortURL}", chain(h.Redirect))
	r.Get("/ping", middleware.RequestLogger(h.PingDB))
}
