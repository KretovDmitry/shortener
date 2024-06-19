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
	"github.com/KretovDmitry/shortener/internal/db"
	"github.com/KretovDmitry/shortener/internal/handler"
	"github.com/KretovDmitry/shortener/internal/logger"
	_ "github.com/KretovDmitry/shortener/migrations"
	"github.com/go-chi/chi/v5"
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

	logger := logger.Get()

	err := config.ParseFlags()
	if err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}

	store, err := db.NewStore(serverCtx, logger)
	if err != nil {
		return fmt.Errorf("new store: %w", err)
	}

	handler, err := handler.New(store, logger, 5)
	if err != nil {
		return fmt.Errorf("new handler: %w", err)
	}
	defer handler.Stop()

	hs := &http.Server{
		Addr:    config.AddrToRun.String(),
		Handler: handler.Register(chi.NewRouter(), logger),
	}

	// Graceful shutdown.
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT,
			syscall.SIGTERM, syscall.SIGQUIT, os.Interrupt)

		signal := <-sig

		logger.With(serverCtx, "signal", signal.String()).
			Infof("Shutting down server with %s timeout",
				config.ShutdownTimeout)

		if err = hs.Shutdown(serverCtx); err != nil {
			logger.Errorf("graceful shutdown failed: %s", err)
		}
		serverStopCtx()
	}()

	logger.Infof("Server has started: %s", config.AddrToRun)
	logger.Infof("Return address: %s", config.AddrToReturn)
	switch config.TLSEnabled {
	case true:
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
	default:
		if err = hs.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("run server failed: %w", err)
		}
	}

	// Wait for server context to be stopped
	select {
	case <-serverCtx.Done():
	case <-time.After(config.ShutdownTimeout):
		return errors.New("graceful shutdown timed out.. forcing exit")
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
