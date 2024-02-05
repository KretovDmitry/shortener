package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/KretovDmitry/shortener/internal/handler"
	"github.com/KretovDmitry/shortener/internal/router"
)

const (
	_defaultHost = "0.0.0.0"
	_defaultPort = 8080
)

func main() {
	validContentType := &[]string{"text/plain", "text/plain; charset=utf-8"}

	router := &router.Router{}
	router.Route(handler.HomeRegexp, http.MethodPost, validContentType, handler.CreateShortURL)
	router.Route(handler.Base58Regexp, http.MethodGet, validContentType, handler.HandleShortURLRedirect)

	addr := fmt.Sprintf("%s:%d", _defaultHost, _defaultPort)

	if err := http.ListenAndServe(addr, router); err != nil {
		fmt.Fprintf(os.Stderr, "%+v\n", err)
		os.Exit(1)
	}
}
