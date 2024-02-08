package main

import (
	"fmt"
	"net/http"

	"github.com/KretovDmitry/shortener/internal/cfg"
	"github.com/KretovDmitry/shortener/internal/handler"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
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

	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Post("/", handler.CreateShortURL)
	r.Get("/{shortURL}", handler.HandleShortURLRedirect)

	fmt.Println("Running server on", cfg.AddrToRun)
	fmt.Println("Returning with", cfg.AddrToReturn)
	return http.ListenAndServe(cfg.AddrToRun.String(), r)
}
