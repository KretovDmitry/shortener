// Package config provides configuration related utilities.
package config

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

// Default values for config.
const (
	defaultHost     = "0.0.0.0"
	defaultPort     = 8080
	defaultFileName = "short-url-db.json"
)

type (
	// Config represents an application configuration.
	Config struct {
		// The data source name (DSN) for connecting to the database.
		DSN string `yaml:"dsn" env:"DATABASE_DSN"`
		// Subconfigs.
		HTTPServer HTTPServer   `yaml:"http_server"`
		FS         *FileStorage `yaml:"file_staroge" env:"FILE_STORAGE_PATH"`
		JWT        JWT          `yaml:"jwt"`
		Logger     Logger       `yaml:"logger"`
		// Path to migrations.
		Migrations string `yaml:"migrations_path"`
		// TLSEnable determines whether the server will be started in the TLS mode.
		TLSEnabled TLSEnabled `yaml:"tls"`
	}
	// Config for HTTP server.
	HTTPServer struct {
		// Address to run the server.
		RunAddress *NetAddress `yaml:"server_address" json:"server_address" env:"SERVER_ADDRESS"`
		// Address to return short URL with.
		ReturnAddress *NetAddress `yaml:"return_address" json:"return_address" env:"BASE_URL"`
		// Read header timeout.
		Timeout time.Duration `yaml:"timeout" env-default:"5s"`
		// Idle timeout.
		IdleTimeout time.Duration `yaml:"idle_timeout" end-default:"60s"`
		// Shutdown timeout.
		ShutdownTimeout time.Duration `yaml:"shutdown_timeout" env:"SHUTDOWN_TIMEOUT" env-default:"30s"`
	}
	// Config for application's logger.
	Logger struct {
		// Path to store log files.
		Path string `ymal:"log_path" env:"LOG_PATH"`
		// Application logging level.
		Level string `yaml:"level" env:"LOG_LEVEL" env-default:"info"`
		// Log files details.
		MaxSizeMB  int `yaml:"max_size_mb"`
		MaxBackups int `yaml:"max_backups"`
		MaxAgeDays int `yaml:"max_age_days"`
	}
	// Config for JWT.
	JWT struct {
		// JWT signing key.
		SigningKey string `yaml:"signing_key" env:"JWT_SIGNING_KEY"`
		// JWT expiration.
		Expiration time.Duration `yaml:"expiration" env:"JWT_EXPIRATION" env-default:"24h"`
	}
)

// Flag Value interface implementation guards.
var (
	_ flag.Value = (*NetAddress)(nil)
	_ flag.Value = (*FileStorage)(nil)
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
	s = strings.TrimPrefix(s, "https://")

	hp := strings.Split(s, ":")

	if len(hp) != 2 {
		return errors.New("need address in a form host:port")
	}

	port, err := strconv.Atoi(hp[1])
	if err != nil {
		return err
	}
	a.Port = port

	if hp[0] != "" {
		a.Host = hp[0]
	}

	return nil
}

// SetValue implements cleanenv value setter.
func (a *NetAddress) SetValue(s string) error {
	return a.Set(s)
}

// FileStorage represents a file storage configuration
// with a path and a write requirement flag.
type FileStorage struct {
	Path          string `yaml:"path"`
	WriteRequired bool   `yaml:"write_required"`
}

// NewFileStorage returns a pointer to a new fileStorage configuration
// with default path and write requirement.
func NewFileStorage() *FileStorage {
	tmp := os.TempDir()
	return &FileStorage{
		Path:          path.Join(tmp, defaultFileName),
		WriteRequired: true,
	}
}

// String returns a string representation of the fileStorage path.
func (fs *FileStorage) String() string {
	return fs.Path
}

// Set sets the file storage path from a string.
//
// If provided string is empty there will be no file writing.
func (fs *FileStorage) Set(s string) error {
	if s == "" {
		fs.WriteRequired = false
		return nil
	}

	fs.Path = s

	return nil
}

// SetValue implements cleanenv value setter.
func (fs *FileStorage) SetValue(s string) error {
	return fs.Set(s)
}

// TLSEnabled determines whether the server will be started in the TLS mode.
type TLSEnabled bool

// Set sets TLSEnabled flag from string.
func (tls *TLSEnabled) Set(s string) error {
	trueValues := []string{
		"true", "1", "t", "T", "TRUE", "True",
	}
	falseValues := []string{
		"false", "0", "f", "F", "FALSE", "False",
	}
	switch {
	case slices.Contains(trueValues, s):
		*tls = true
	case slices.Contains(falseValues, s):
		*tls = false
	default:
		msg := fmt.Sprintf(
			"invalid value: %q; need boolean value in form: true: %q false: %q",
			s,
			strings.Join(trueValues, "\", \""),
			strings.Join(falseValues, "\", \""),
		)
		return errors.New(msg)
	}
	return nil
}

// SetValue implements cleanenv value setter.
func (tls *TLSEnabled) SetValue(s string) error {
	return tls.Set(s)
}

// String returns a string representation of the TLSEnabled flag.
func (tls *TLSEnabled) String() string {
	switch *tls {
	case true:
		return "true"
	default:
		return "false"
	}
}

// Order of loading configuration:
// 1. Config file (YAML, JSON supported)
// 2. Flags
// 3. Environment variables

// Load returns an application configuration which is populated
// from the given configuration file, environment variables and flags.
func MustLoad() *Config {
	// Configuration file path.
	configPath := flag.String("config", "./config/local.yml", "path to the config file")
	flag.Parse()

	// Check if file exists.
	if _, err := os.Stat(*configPath); os.IsNotExist(err) {
		log.Fatalf("config file does not exist: %v", err)
	}

	var cfg Config
	// Setup default values.
	cfg.HTTPServer.RunAddress = NewNetAddress()
	cfg.HTTPServer.ReturnAddress = NewNetAddress()
	cfg.FS = NewFileStorage()

	// Load from YAML cfg file.
	file, err := os.Open(*configPath)
	if err != nil {
		log.Fatalf("failed to open config file: %v", err)
	}
	if err = cleanenv.ParseYAML(file, &cfg); err != nil {
		log.Fatalf("failed to parse config file: %v", err)
	}

	// Read given flags. If not provided use file values.
	flag.Var(cfg.HTTPServer.RunAddress, "a", "server start address in form host:port")
	flag.Var(cfg.HTTPServer.ReturnAddress, "b", "server return address in form host:port")
	flag.Var(cfg.FS, "f", "file storage path")
	flag.Var(&cfg.TLSEnabled, "s", "run the server in TLS mode")
	flag.StringVar(&cfg.DSN, "d", cfg.DSN, "server data source name")
	flag.StringVar(&cfg.Logger.Level, "l", "info", "logging level")
	flag.StringVar(&cfg.Migrations, "m", ".", "path to migration directory")
	flag.Parse()

	// Read environment variables.
	if err = cleanenv.ReadEnv(&cfg); err != nil {
		log.Fatalf("failed to read environment variables: %v", err)
	}

	return &cfg
}
