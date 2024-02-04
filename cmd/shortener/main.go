package main

import (
	"fmt"
	"os"

	"github.com/KretovDmitry/shortener/internal/server"
)

const (
	_defaultHost = "0.0.0.0"
	_defaultPort = 8080
)

func main() {
	if err := server.Run(_defaultHost, _defaultPort); err != nil {
		fmt.Fprintf(os.Stderr, "%+v\n", err)
		os.Exit(1)
	}
}
