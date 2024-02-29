package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/KretovDmitry/shortener/internal/cfg"
	"github.com/KretovDmitry/shortener/internal/db"
	"github.com/KretovDmitry/shortener/internal/handler"
	"github.com/KretovDmitry/shortener/internal/logger"
	"github.com/KretovDmitry/shortener/internal/middleware"
	"github.com/KretovDmitry/shortener/internal/middleware/gzip"
	"github.com/go-chi/chi/v5"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func main() {
	l := logger.Get()
	defer l.Sync()

	mux, err := initService()
	if err != nil {
		l.Fatal("init service failed", zap.Error(err))
	}

	server := &http.Server{
		Addr:    cfg.AddrToRun.String(),
		Handler: mux,
	}

	// Server run context
	serverCtx, serverStopCtx := context.WithCancel(context.Background())

	// Listen for syscall signals for process to interrupt/quit
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-sig

		// Shutdown signal with grace period of 30 seconds
		shutdownCtx, shutdownStopCtx := context.WithTimeout(serverCtx, 30*time.Second)

		go func() {
			<-shutdownCtx.Done()
			if shutdownCtx.Err() == context.DeadlineExceeded {
				log.Fatal("graceful shutdown timed out.. forcing exit")
			}
			shutdownStopCtx()
		}()

		// Trigger graceful shutdown
		err := server.Shutdown(shutdownCtx)
		if err != nil {
			l.Fatal("graceful shutdown failed", zap.Error(err))
		}
		serverStopCtx()
	}()

	// Run the server
	l.Info("Server has started", zap.String("addr", cfg.AddrToRun.String()))
	l.Info("Return address", zap.String("addr", cfg.AddrToReturn.String()))
	err = server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		l.Fatal("ListenAndServe failed", zap.Error(err))
	}

	// Wait for server context to be stopped
	<-serverCtx.Done()
}

func initService() (http.Handler, error) {
	err := cfg.ParseFlags()
	if err != nil {
		return nil, errors.Wrap(err, "parse flags")
	}

	store, err := db.NewFileStore(cfg.FileStorage.Path())
	if err != nil {
		return nil, errors.Wrap(err, "new store")
	}

	hctx, err := handler.NewHandlerContext(store)
	if err != nil {
		return nil, errors.Wrap(err, "new handler context")
	}

	r := chi.NewRouter()

	chain := middleware.BuildChain(
		gzip.DefaultHandler().WrapHandler,
		middleware.RequestLogger,
		gzip.Unzip,
	)

	r.Post("/", chain(hctx.ShortenText))
	r.Get("/{shortURL}", chain(hctx.HandleShortURLRedirect))
	r.Post("/api/shorten", chain(hctx.ShortenJSON))

	return r, nil
}
