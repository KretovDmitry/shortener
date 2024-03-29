package handler

import (
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"strings"

	"github.com/KretovDmitry/shortener/internal/db"
	"github.com/KretovDmitry/shortener/internal/logger"
	"github.com/KretovDmitry/shortener/pkg/middleware"
	"github.com/KretovDmitry/shortener/pkg/middleware/gzip"
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
	logger := logger.Get()

	chain := middleware.BuildChain(
		middleware.RequestLogger(logger),
		gzip.DefaultHandler().WrapHandler,
		gzip.Unzip(logger),
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
	if code >= 500 {
		h.logger.Error(message, zap.Error(err), zap.String("loc", caller(2)))
	}
	if code >= 400 && code < 500 {
		h.logger.Info(message, zap.Error(err), zap.String("loc", caller(2)))
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(code)
	_, err = w.Write([]byte(fmt.Sprintf("%s: %s", message, err)))
	if err != nil {
		h.logger.Error("failed to write response", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// caller returns a file and line from a specified depth in the call stack.
func caller(depth int) string {
	_, file, line, _ := runtime.Caller(depth)
	idx := strings.LastIndexByte(file, '/')
	// using idx+1 below handles both of following cases:
	// idx == -1 because no "/" was found, or
	// idx >= 0 and we want to start at the character after the found "/"
	return fmt.Sprintf("%s:%d", file[idx+1:], line)
}
