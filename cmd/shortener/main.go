package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "net/http/pprof"

	"github.com/KretovDmitry/shortener/internal/config"
	"github.com/KretovDmitry/shortener/internal/handler"
	"github.com/KretovDmitry/shortener/internal/logger"
	"github.com/KretovDmitry/shortener/internal/repository"
	"github.com/go-chi/chi/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"golang.org/x/crypto/acme/autocert"
)

var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

func main() {
	printBuildInfo()
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// Server run context.
	serverCtx, serverStopCtx := context.WithCancel(context.Background())
	defer serverStopCtx()

	// Load application configuration.
	cfg := config.MustLoad()

	// Create root logger tagged with server version.
	logger := logger.New(cfg).With(serverCtx, "version", buildVersion)
	defer func() {
		_ = logger.Sync()
	}()

	// Init URL repository.
	store, err := repository.NewURLStore(cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to init store: %w", err)
	}

	// Init HTTP handlers.
	handler, err := handler.New(store, cfg, logger)
	if err != nil {
		return fmt.Errorf("new handler: %w", err)
	}
	// Stop async short URL deletion.
	defer handler.Stop()

	// Init HTTP server.
	hs := &http.Server{
		Addr:              cfg.HTTPServer.RunAddress.String(),
		ReadHeaderTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:       cfg.HTTPServer.IdleTimeout,
		Handler:           handler.Register(chi.NewRouter(), cfg, logger),
	}

	// Graceful shutdown.
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT,
			syscall.SIGTERM, syscall.SIGQUIT, os.Interrupt)

		signal := <-sig

		logger.With(serverCtx, "signal", signal.String()).
			Infof("Shutting down server with %s timeout",
				cfg.HTTPServer.ShutdownTimeout)

		if err = hs.Shutdown(serverCtx); err != nil {
			logger.Errorf("graceful shutdown failed: %s", err)
		}
		serverStopCtx()
	}()

	logger.Infof("Server has started: %s", cfg.HTTPServer.RunAddress)
	logger.Infof("Return address: %s", cfg.HTTPServer.ReturnAddress)
	if cfg.TLSEnabled {
		cm := &autocert.Manager{
			Cache:  autocert.DirCache("cache/certs"),
			Prompt: autocert.AcceptTOS,
		}
		hs.TLSConfig = cm.TLSConfig()
		logger.Info("The server is running over the SSL protocol")
		if err = hs.ListenAndServeTLS("", ""); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("run server failed: %w", err)
		}
	} else {
		if err = hs.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("run server failed: %w", err)
		}
	}

	// Wait for server context to be stopped
	select {
	case <-serverCtx.Done():
	case <-time.After(cfg.HTTPServer.ShutdownTimeout):
		return errors.New("graceful shutdown timed out... forcing exit")
	}

	return nil
}

func printBuildInfo() {
	if buildVersion == "" {
		fmt.Println("Build version: N/A")
	} else {
		fmt.Printf("Build version: %s\n", buildVersion)
	}
	if buildDate == "" {
		fmt.Println("Build date: N/A")
	} else {
		fmt.Printf("Build date: %s\n", buildDate)
	}
	if buildCommit == "" {
		fmt.Println("Build commit: N/A")
	} else {
		fmt.Printf("Build commit: %s\n", buildCommit)
	}
}
