package handler

import (
	"errors"
	"fmt"
	"net/http"
	"runtime"

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
	ErrOnlyTextPlainContentType       = errors.New("only text/plain content-type is allowed")
	ErrURLIsNotProvided               = errors.New("URL is not provided")
	ErrNotValidURL                    = errors.New("URL is not valid")
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

	r.Get("/ping", chain(h.PingDB))
	r.Get("/{shortURL}", chain(h.Redirect))
}

// textError writes error response to the response writer in a text/plain format.
func (h *handler) textError(w http.ResponseWriter, message string, err error, code int) {
	pc, _, _, ok := runtime.Caller(1)
	details := runtime.FuncForPC(pc)
	name := "unknown"
	if ok && details != nil {
		name = details.Name()
	}
	if code >= 500 {
		h.logger.Error(message, zap.Error(err), zap.String("function", name))
	}
	if code >= 400 && code < 500 {
		h.logger.Info(message, zap.Error(err), zap.String("function", name))
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(code)
	_, err = w.Write([]byte(fmt.Sprintf("%s: %s", message, err)))
	if err != nil {
		h.logger.Error("failed to write response", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
