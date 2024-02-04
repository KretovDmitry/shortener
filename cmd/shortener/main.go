package main

import (
	"fmt"
	"os"

	"github.com/KretovDmitry/shortener/internal/server"
)

const (
	_defaultHost = "127.0.0.1"
	_defaultPort = 8080
)

func main() {
	if err := server.Run(_defaultHost, _defaultPort); err != nil {
		fmt.Fprintf(os.Stderr, "%+v\n", err)
		os.Exit(1)
	}
}
