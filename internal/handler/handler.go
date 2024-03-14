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
	store  db.URLStorage
	logger *zap.Logger
}

var (
	ErrOnlyGETMethodIsAllowed         = errors.New("only GET method is allowed")
	ErrOnlyPOSTMethodIsAllowed        = errors.New("only POST method is allowed")
	ErrOnlyApplicationJSONContentType = errors.New("only application/json content-type is allowed")
	ErrOnlyTextContentType            = errors.New("only text/plain content-type is allowed")
	ErrURLIsNotProvided               = errors.New("URL is not provided")
)

// New constructs a new handlerContext,
// ensuring that the dependencies are valid values
func New(store db.URLStorage) (*handler, error) {
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
	chain := middleware.BuildChain(
		gzip.DefaultHandler().WrapHandler,
		middleware.RequestLogger,
		gzip.Unzip,
	)

	// Register routes.
	r.Post("/", chain(h.ShortenText))
	r.Post("/api/shorten", chain(h.ShortenJSON))
	r.Post("/api/shorten/batch", chain(h.ShortenBatch))

	r.Get("/{shortURL}", chain(h.Redirect))
	r.Get("/ping", chain(h.PingDB))
}
