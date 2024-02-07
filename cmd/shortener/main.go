package main

import (
	"fmt"
	"net/http"

	"github.com/KretovDmitry/shortener/internal/cfg"
	"github.com/KretovDmitry/shortener/internal/handler"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	cfg.ParseFlags()

	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Post("/", handler.CreateShortURL)
	r.Get("/{shortURL}", handler.HandleShortURLRedirect)

	fmt.Println(cfg.AddrToReturn)
	fmt.Println("Running server on", cfg.AddrToRun)
	return http.ListenAndServe(cfg.AddrToRun.String(), r)
}
