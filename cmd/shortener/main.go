package main

import (
	"fmt"
	"net/http"

	"github.com/KretovDmitry/shortener/internal/cfg"
	"github.com/KretovDmitry/shortener/internal/db"
	"github.com/KretovDmitry/shortener/internal/handler"
	"github.com/KretovDmitry/shortener/internal/logger"
	"github.com/KretovDmitry/shortener/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/pkg/errors"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	err := cfg.ParseFlags()
	if err != nil {
		return errors.Wrap(err, "parse flags")
	}

	logger, err := logger.New(cfg.LogLevel)
	if err != nil {
		return errors.Wrap(err, "new logger")
	}

	store := db.NewInMemoryStore()

	hctx, err := handler.NewHandlerContext(store)
	if err != nil {
		return errors.Wrap(err, "new handler context")
	}

	r := chi.NewRouter()

	r.Post("/", middleware.RequestInfo(logger, hctx.CreateShortURL))
	r.Get("/{shortURL}", middleware.RequestInfo(logger, hctx.HandleShortURLRedirect))

	fmt.Println("Running server on", cfg.AddrToRun)
	fmt.Println("Returning with", cfg.AddrToReturn)
	return http.ListenAndServe(cfg.AddrToRun.String(), r)
}
