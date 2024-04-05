package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/KretovDmitry/shortener/internal/db"
	"github.com/KretovDmitry/shortener/internal/logger"
	"github.com/KretovDmitry/shortener/internal/middleware"
	"github.com/KretovDmitry/shortener/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/nanmu42/gzip"
	"go.uber.org/zap"
)

var (
	ErrOnlyGETMethodIsAllowed         = errors.New("only GET method is allowed")
	ErrOnlyPOSTMethodIsAllowed        = errors.New("only POST method is allowed")
	ErrOnlyDeleteMethodIsAllowed      = errors.New("only DELETE method is allowed")
	ErrOnlyApplicationJSONContentType = errors.New("only application/json content-type is allowed")
	ErrOnlyTextPlainContentType       = errors.New("only text/plain content-type is allowed")
	ErrURLIsNotProvided               = errors.New("URL is not provided")
	ErrNotValidURL                    = errors.New("URL is not valid")
)

type Handler struct {
	store          db.URLStorage
	logger         *zap.Logger
	deleteURLsChan chan *models.URL
	wg             sync.WaitGroup
	done           chan struct{}
}

// New constructs a new handlerContext,
// ensuring that the dependencies are valid values
func New(store db.URLStorage) (*Handler, error) {
	if store == nil {
		return nil, errors.New("nil store")
	}

	newInstance := &Handler{
		store:          store,
		logger:         logger.Get(),
		deleteURLsChan: make(chan *models.URL),
		done:           make(chan struct{}),
	}

	newInstance.wg.Add(1)
	go newInstance.flushDeleteURL()

	return newInstance, nil
}

func (h *Handler) Close() {
	sync.OnceFunc(func() {
		close(h.done)
		h.wg.Wait()
	})
}

// Register sets up the routes for the HTTP server.
func (h *Handler) Register(r chi.Router) {
	r.Use(middleware.Logger)
	r.Use(gzip.DefaultHandler().WrapHandler)
	r.Use(middleware.Unzip)
	r.Use(middleware.Authorization)

	r.Post("/", h.ShortenText)
	r.Post("/api/shorten", h.ShortenJSON)
	r.Post("/api/shorten/batch", h.ShortenBatch)

	r.Get("/ping", h.PingDB)
	r.Get("/{shortURL}", h.Redirect)

	r.Delete("/api/user/urls", h.DeleteURLs)

	r.Route("/api/user", func(r chi.Router) {
		r.Use(middleware.OnlyWithToken)
		r.Get("/urls", h.GetAllByUserID)
	})
}

func (h *Handler) flushDeleteURL() {
	ticker := time.NewTicker(10 * time.Second)
	defer h.wg.Done()

	var URLs []*models.URL

	for {
		select {
		case url := <-h.deleteURLsChan:
			h.logger.Info("incoming delete", zap.Any("url", url))
			URLs = append(URLs, url)

		case <-h.done:
			if len(URLs) == 0 {
				return
			}

			h.logger.Info("deleting", zap.Int("num", len(URLs)))

			err := h.store.DeleteURLs(context.TODO(), URLs...)
			if err != nil {
				h.logger.Error("failed delete URLs", zap.Error(err))
				return
			}

			URLs = nil

		case <-ticker.C:
			if len(URLs) == 0 {
				continue
			}

			h.logger.Info("deleting", zap.Int("num", len(URLs)))

			err := h.store.DeleteURLs(context.TODO(), URLs...)
			if err != nil {
				h.logger.Error("failed delete URLs", zap.Error(err))
				continue
			}

			URLs = nil
		}
	}

}

// textError writes error response to the response writer in a text/plain format.
func (h *Handler) textError(w http.ResponseWriter, message string, err error, code int) {
	if code >= 500 {
		h.logger.Error(message, zap.Error(err), zap.String("loc", caller(2)))
	} else {
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
