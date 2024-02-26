package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/KretovDmitry/shortener/internal/cfg"
	"github.com/KretovDmitry/shortener/internal/db"
	"github.com/KretovDmitry/shortener/internal/handler"
	"github.com/KretovDmitry/shortener/internal/logger"
	"github.com/KretovDmitry/shortener/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

func main() {
	l := logger.Get()
	defer l.Sync()

	if err := run(); err != nil {
		l.Error("run", zap.Error(err))
		os.Exit(1)
	}
}

func run() error {
	err := cfg.ParseFlags()
	if err != nil {
		return errors.Wrap(err, "parse flags")
	}

	store := db.NewInMemoryStore()

	hctx, err := handler.NewHandlerContext(store)
	if err != nil {
		return errors.Wrap(err, "new handler context")
	}

	r := chi.NewRouter()

	r.Post("/", middleware.RequestLogger(hctx.ShortenText))
	r.Get("/{shortURL}", middleware.RequestLogger(hctx.HandleShortURLRedirect))
	r.Post("/api/shorten", middleware.RequestLogger(hctx.ShortenJSON))

	httpServer := &http.Server{
		Addr:    cfg.AddrToRun.String(),
		Handler: r,
	}

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)

		<-c
		cancel()
	}()

	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		fmt.Println("Running server on", cfg.AddrToRun)
		fmt.Println("Returning with", cfg.AddrToReturn)
		return httpServer.ListenAndServe()
	})
	g.Go(func() error {
		<-gCtx.Done()
		return httpServer.Shutdown(context.Background())
	})

	return g.Wait()
}
