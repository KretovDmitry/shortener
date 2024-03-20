package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/KretovDmitry/shortener/internal/config"
	"github.com/KretovDmitry/shortener/internal/db"
	"github.com/KretovDmitry/shortener/internal/handler"
	"github.com/KretovDmitry/shortener/internal/logger"
	_ "github.com/KretovDmitry/shortener/migrations"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func main() {
	l := logger.Get()
	defer l.Sync()

	// Server run context
	serverCtx, serverStopCtx := context.WithCancel(context.Background())

	mux, err := initService(serverCtx)
	if err != nil {
		l.Fatal("init service failed", zap.Error(err))
	}

	server := &http.Server{
		Addr:    config.AddrToRun.String(),
		Handler: mux,
	}

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

	l.Info("Server has started", zap.String("addr", config.AddrToRun.String()))
	l.Info("Return address", zap.String("addr", config.AddrToReturn.String()))
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

func initService(ctx context.Context) (http.Handler, error) {
	err := config.ParseFlags()
	if err != nil {
		return nil, fmt.Errorf("parse flags: %w", err)
	}

	store, err := db.NewStore(ctx)
	if err != nil {
		return nil, fmt.Errorf("new store: %w", err)
	}

	handler, err := handler.New(store)
	if err != nil {
		return nil, fmt.Errorf("new handler context: %w", err)
	}

	r := chi.NewRouter()
	handler.Register(r)

	return r, nil
}
