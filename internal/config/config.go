// Package config provides configuration related utilities.
package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Default values for config.
const (
	defaultHost     = "0.0.0.0"
	defaultPort     = 8080
	defaultFileName = "short-url-db.json"
)

// Flag Value interface implementation guards.
var (
	_ flag.Value = (*NetAddress)(nil)
	_ flag.Value = (*FileStorage)(nil)
	_ flag.Value = (*Duration)(nil)
)

// NetAddress represents a network address with a host and a port.
type NetAddress struct {
	Host string
	Port int
}

// NewNetAddress returns a pointer to a new NetAddress with default Host and Port.
func NewNetAddress() *NetAddress {
	return &NetAddress{
		Host: defaultHost,
		Port: defaultPort,
	}
}

// String returns a string representation of the NetAddress in the form "host:port".
func (a *NetAddress) String() string {
	return fmt.Sprintf("%s:%d", a.Host, a.Port)
}

// Set sets the host and port of the NetAddress from a string in the form "host:port".
//
// It returns an error if the input string is not in the correct format.
func (a *NetAddress) Set(s string) error {
	s = strings.TrimPrefix(s, "http://")

	hp := strings.Split(s, ":")

	if len(hp) != 2 {
		return errors.New("need address in a form host:port")
	}

	port, err := strconv.Atoi(hp[1])
	if err != nil {
		return err
	}

	if hp[0] != "" {
		a.Host = hp[0]
	}

	a.Port = port

	return nil
}

// FileStorage represents a file storage configuration
// with a path and a write requirement flag.
type FileStorage struct {
	path          string
	writeRequired bool
}

// NewFileStorage returns a pointer to a new fileStorage configuration
// with default path and write requirement.
func NewFileStorage() *FileStorage {
	tmp := os.TempDir()
	return &FileStorage{
		path:          path.Join(tmp, defaultFileName),
		writeRequired: true,
	}
}

// String returns a string representation of the fileStorage path.
func (fs *FileStorage) String() string {
	return fs.path
}

// Set sets the file storage path from a string.
//
// If provided string is empty there will be no file writing.
func (fs *FileStorage) Set(s string) error {
	if s == "" {
		fs.writeRequired = false
		return nil
	}

	fs.path = s

	return nil
}

// Path returns the file storage path.
func (fs *FileStorage) Path() string {
	return fs.path
}

// WriteRequired returns whether the file storage requires writing.
func (fs *FileStorage) WriteRequired() bool {
	return fs.writeRequired
}

// Duration represents a duration in time.Duration format.
// Defaults to time.Duration zero value.
type Duration time.Duration

// String returns a string representation of the Duration in the form of time.Duration.
func (d *Duration) String() string {
	return time.Duration(*d).String()
}

// Set sets the duration from a string in the form of time.Duration.
//
// It returns an error if the input string is not in the correct format.
func (d *Duration) Set(s string) error {
	dd, err := time.ParseDuration(s)
	if err != nil {
		return err
	}

	*d = Duration(dd)

	return nil
}

// Global singleton configuration variables.
var (
	AddrToRun       = NewNetAddress()
	AddrToReturn    = NewNetAddress()
	FS              = NewFileStorage()
	DSN             string
	LogLevel        string
	Secret          string
	JWT             = Duration(time.Hour * 3)
	MigrationDir    string
	ShutdownTimeout = 30 * time.Second
	TLSEnabled      bool
)

// ParseFlags parses the command line flags and sets the corresponding values.
// It also reads the environment variables and sets the values if they are present.
//
// Flags take precedence over the default values.
// Environment variables have the highest priority.
func ParseFlags() error {
	// flags take precedence over the default values
	flag.Var(AddrToRun, "a", "Net address host:port to run server")
	flag.Var(AddrToReturn, "b", "Net address host:port to return short URLs")
	flag.Var(FS, "f", "File storage path")
	flag.Var(&JWT, "j", `JWT lifetime in form of time.Duration: such as "2h45m"`)
	flag.StringVar(&DSN, "d", "", "Data source name in form postgres URL or DSN string")
	flag.StringVar(&LogLevel, "l", "info", "Log level")
	flag.StringVar(&Secret, "k", "test", "Secret key for JWT")
	flag.StringVar(&MigrationDir, "m", ".", "Path to migration directory")
	flag.BoolVar(&TLSEnabled, "s", false, "Enable HTTPS")
	flag.Parse()

	// ENV variables have the highest priority
	if envRunAddr := os.Getenv("SERVER_ADDRESS"); envRunAddr != "" {
		if err := AddrToRun.Set(envRunAddr); err != nil {
			return fmt.Errorf("invalid SERVER_ADDRESS: %w", err)
		}
	}

	if envReturnAddr := os.Getenv("BASE_URL"); envReturnAddr != "" {
		if err := AddrToReturn.Set(envReturnAddr); err != nil {
			return fmt.Errorf("invalid BASE_URL: %w", err)
		}
	}

	if envFileStoragePath, set := os.LookupEnv("FILE_STORAGE_PATH"); set {
		if err := FS.Set(envFileStoragePath); err != nil {
			return fmt.Errorf("invalid FILE_STORAGE_PATH: %w", err)
		}
	}

	if envDSN := os.Getenv("DATABASE_DSN"); envDSN != "" {
		DSN = envDSN
	}

	if envLogLevel := os.Getenv("LOG_LEVEL"); envLogLevel != "" {
		LogLevel = envLogLevel
	}

	if envTLSEnabled := os.Getenv("ENABLE_HTTPS"); envTLSEnabled != "" {
		trueValues := []string{
			"true", "1", "t", "T", "TRUE", "True",
		}
		falseValues := []string{
			"false", "0", "f", "F", "FALSE", "False",
		}
		switch {
		case slices.Contains(trueValues, envTLSEnabled):
			TLSEnabled = true
		case slices.Contains(falseValues, envTLSEnabled):
			TLSEnabled = false
		default:
			msg := fmt.Sprintf(
				"invalid ENABLE_HTTPS environment variable: %s; "+
					"need boolean value in form:\n\t"+
					"true: \"%s\"\n\t"+
					"false: \"%s\"",
				envTLSEnabled,
				strings.Join(trueValues, "\", \""),
				strings.Join(falseValues, "\", \""),
			)
			return errors.New(msg)
		}
	}

	return nil
}
