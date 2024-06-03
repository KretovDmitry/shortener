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

	"github.com/KretovDmitry/shortener/internal/config"
	"github.com/KretovDmitry/shortener/internal/db"
	"github.com/KretovDmitry/shortener/internal/logger"
	"github.com/KretovDmitry/shortener/internal/middleware"
	"github.com/KretovDmitry/shortener/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/nanmu42/gzip"
	"go.uber.org/zap"
)

// Handler struct represents the main handler for the application.
type Handler struct {
	// store is the database URL storage.
	store db.URLStorage
	// logger is the application logger.
	logger *zap.Logger
	// deleteURLsChan is a channel for sending deleted URLs to be flushed from the database.
	deleteURLsChan chan *models.URL
	// wg is a wait group used to manage the goroutine that flushes deleted URLs.
	wg *sync.WaitGroup
	// done is a channel used to signal the stop of the handler.
	done chan struct{}
	// bufLen is the buffer length for storing deleted URLs before flushing them to the database.
	bufLen int
}

// New constructs a new handler, ensuring that the dependencies are valid values.
func New(store db.URLStorage, bufLen int) (*Handler, error) {
	if store == nil {
		return nil, errors.New("nil store")
	}

	h := &Handler{
		store:          store,
		logger:         logger.Get(),
		deleteURLsChan: make(chan *models.URL),
		wg:             &sync.WaitGroup{},
		done:           make(chan struct{}),
		bufLen:         bufLen,
	}

	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		h.flushDeletedURLs()
	}()

	return h, nil
}

// Stop stops the handler and waits for all goroutines to finish.
// It sends a close signal to the done channel and then waits for the
// WaitGroup to finish. If the shutdown timeout is exceeded, it logs an error.
// It is safe for concurrent use.
func (h *Handler) Stop() {
	sync.OnceFunc(func() {
		close(h.done)
	})()

	ready := make(chan struct{})
	go func() {
		defer close(ready)
		h.wg.Wait()
	}()

	select {
	case <-time.After(config.ShutdownTimeout):
		h.logger.Error("handler stop: shutdown timeout exceeded")
	case <-ready:
		return
	}
}

// Register sets up the routes for the HTTP server.
func (h *Handler) Register(r chi.Router) chi.Router {
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

	return r
}

// flushDeletedURLs is a goroutine that periodically flushes the deleted URLs
// from the buffer to the database. It uses a ticker to trigger the flush
// operation every 10 seconds. If the channel for sending deleted URLs is closed,
// the goroutine stops.
// It is safe for concurrent use.
func (h *Handler) flushDeletedURLs() {
	ticker := time.NewTicker(10 * time.Second)
	URLs := make([]*models.URL, 0, h.bufLen)

	for {
		select {
		case url := <-h.deleteURLsChan:
			URLs = append(URLs, url)

		case <-h.done:
			if len(URLs) == 0 {
				return
			}
			_ = h.flush(URLs...)
			return

		case <-ticker.C:
			if len(URLs) == 0 {
				continue
			}
			if err := h.flush(URLs...); err != nil {
				continue
			}
			// reset buffer only when flush succeeded
			URLs = URLs[:0:h.bufLen]
		}
	}
}

// flush deletes the given URLs from the database.
// If an error occurs during the deletion process, it logs an error message
// with the error details. It returns the error encountered during the deletion process.
func (h *Handler) flush(URLs ...*models.URL) error {
	if len(URLs) == 0 {
		return nil
	}

	err := h.store.DeleteURLs(context.TODO(), URLs...)
	if err != nil {
		h.logger.Error("failed to delete URLs", zap.Error(err),
			zap.Int("num", len(URLs)), zap.Any("urls", URLs))
	}

	return err
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
	_, err = fmt.Fprintf(w, "%s: %s", err, message)
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

// IsApplicationJSONContentType returns true if the content type of the
// HTTP request is application/json.
func (h *Handler) IsApplicationJSONContentType(r *http.Request) bool {
	contentType := r.Header.Get("Content-Type")
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	return contentType == "application/json"
}

// IsTextPlainContentType returns true if the content type of the
// HTTP request is text/plain.
func (h *Handler) IsTextPlainContentType(r *http.Request) bool {
	contentType := r.Header.Get("Content-Type")
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	if i := strings.Index(contentType, ";"); i > -1 {
		contentType = contentType[0:i]
	}
	return contentType == "text/plain"
}
