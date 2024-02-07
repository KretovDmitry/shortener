package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/KretovDmitry/shortener/internal/handler"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

const (
	_defaultHost = "0.0.0.0"
	_defaultPort = 8080
)

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Post("/", handler.CreateShortURL)
	r.Get("/{shortURL}", handler.HandleShortURLRedirect)

	addr := fmt.Sprintf("%s:%d", _defaultHost, _defaultPort)

	if err := http.ListenAndServe(addr, r); err != nil {
		fmt.Fprintf(os.Stderr, "%+v\n", err)
		os.Exit(1)
	}
}
