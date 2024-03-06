package main

import (
	"context"
	"fmt"
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

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, os.Interrupt)
	go func() {
		<-sig

		err := server.Shutdown(serverCtx)
		if err != nil {
			l.Fatal("graceful shutdown failed", zap.Error(err))
		}
		serverStopCtx()
	}()

	l.Info("Server has started", zap.String("addr", cfg.AddrToRun.String()))
	l.Info("Return address", zap.String("addr", cfg.AddrToReturn.String()))
	err = server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		l.Fatal("ListenAndServe failed", zap.Error(err))
	}

	// Wait for server context to be stopped
	select {
	case <-serverCtx.Done():
	case <-time.After(30 * time.Second):
		l.Fatal("graceful shutdown timed out.. forcing exit")
	}
}

func initService() (http.Handler, error) {
	err := cfg.ParseFlags()
	if err != nil {
		return nil, fmt.Errorf("parse flags: %w", err)
	}

	store, err := db.NewFileStore(cfg.FileStorage.Path())
	if err != nil {
		return nil, fmt.Errorf("new store: %w", err)
	}

	hctx, err := handler.NewHandlerContext(store)
	if err != nil {
		return nil, fmt.Errorf("new handler context: %w", err)
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
