// Package server provides an HTTP server.
package server

import (
	"fmt"
	"net"
	"net/http"

	"github.com/KretovDmitry/shortener/internal/handler"
	"github.com/pkg/errors"
)

const (
	minPortNumber = 0
	maxPortNumber = 1<<16 - 1
)

type notValidIPError struct {
	ip string
}

func (e *notValidIPError) Error() string {
	return fmt.Sprintf("not valid IP: %s\n", e.ip)
}

type notValidPortError struct {
	port int
}

func (e *notValidPortError) Error() string {
	return fmt.Sprintf("not valid port: %d\n", e.port)
}

// Run starts a server on specified ip adress and port.
func Run(ip string, port int) error {
	if port < minPortNumber || port > maxPortNumber {
		return errors.Wrap(&notValidPortError{port: port}, "server failed")
	}

	validIP := net.ParseIP(ip)
	if validIP == nil {
		return errors.Wrap(&notValidIPError{ip: ip}, "server failed")
	}

	router := &handler.Router{}
	router.Route(handler.HomeRegexp, http.MethodPost, "text/plain", handler.CreateShortUrl)
	router.Route(handler.Base58Regexp, http.MethodGet, "text/plain", handler.HandleShortUrlRedirecth)

	addr := fmt.Sprintf("%s:%d", validIP, port)

	if err := http.ListenAndServe(addr, router); err != nil {
		return errors.Wrap(err, "server failed")
	}

	return nil
}
