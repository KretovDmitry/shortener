// Package config provides configuration related utilities.
package config

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

// Default values for config.
const (
	defaultHost                   = "0.0.0.0"
	defaultPort                   = "8080"
	defaultFileName               = "short-url-db.json"
	defaultLogPath                = "app.log"
	defaultMaxLogSizeMB           = 5
	defaultMaxLogBackups          = 10
	defaultMaxLogFileLifetimeDays = 14
	defaultMigtationsPath         = "."
	defaultDeleteBufLen           = 5
)

// Default variables.
var (
	// Default file storage path.
	defaultFileStoragePath = path.Join(os.TempDir(), defaultFileName)
	// Default address to start server and return shortened urls with.
	DefaultAddress = fmt.Sprintf("%s:%s", defaultHost, defaultPort)
)

// Config represents an application configuration.
type (
	Config struct {
		// The data source name (DSN) for connecting to the database.
		DSN string `yaml:"dsn" env:"DATABASE_DSN"`
		// Subconfigs.
		Server Server `yaml:"http_server"`
		JWT    JWT    `yaml:"jwt"`
		Logger Logger `yaml:"logger"`
		// Path to migrations.
		Migrations string `yaml:"migrations_path"`
		// Path to the file storage.
		FileStoragePath string `yaml:"file_storage_path" env:"FILE_STORAGE_PATH"`
		// TLSEnable determines whether the server will be started in the TLS mode.
		TLSEnabled Enabled `yaml:"enable_https" env:"ENABLE_HTTPS"`
		// RPCEnabled defines if the server should run as a RPC server. Default HTTP.
		RPCEnabled Enabled `yaml:"enable_rpc" env:"ENABLE_RPC"`
		// Length of the buffer for asynchronous deletion.
		DeleteBufLen int `yaml:"delete_buffer_length"`
		// Classless Inter-Domain Routing (CIDR).
		TrustedSubnet *Subnet `yaml:"trusted_subnet" env:"TRUSTED_SUBNET"`
	}
	// Config for server.
	Server struct {
		// Address to run the server.
		RunAddress *NetAddress `yaml:"server_address" env:"SERVER_ADDRESS"`
		// Address to return short URL with.
		ReturnAddress *NetAddress `yaml:"return_address" env:"BASE_URL"`
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
		Path string `yaml:"log_path" env:"LOG_PATH"`
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

// Interface implementation guards.
var (
	_ flag.Value      = (*NetAddress)(nil)
	_ cleanenv.Setter = (*NetAddress)(nil)
	_ flag.Value      = (*Subnet)(nil)
	_ cleanenv.Setter = (*Subnet)(nil)
)

// NetAddress represents a network address with a host and a port.
type NetAddress string

// NewNetAddress returns a pointer to a new NetAddress with default Host and Port.
func NewNetAddress() *NetAddress {
	a := NetAddress(DefaultAddress)
	return &a
}

// String returns a string representation of the NetAddress in the form "host:port".
func (a *NetAddress) String() string {
	return string(*a)
}

// Set sets the host and port of the NetAddress from a string
// in the form "host:port".
func (a *NetAddress) Set(s string) error {
	s = strings.TrimPrefix(s, "http://")
	s = strings.TrimPrefix(s, "https://")

	hp := strings.Split(s, ":")

	if len(hp) != 2 {
		return errors.New("need address in a form host:port")
	}

	if _, err := strconv.Atoi(hp[1]); err != nil {
		return fmt.Errorf("invalid port: %w", err)
	}

	if hp[0] != "" {
		*a = NetAddress(fmt.Sprintf("%s:%s", hp[0], hp[1]))
		return nil
	}

	*a = NetAddress(fmt.Sprintf("%s:%s", defaultHost, hp[1]))
	return nil
}

// SetValue implements cleanenv value setter.
func (a *NetAddress) SetValue(s string) error {
	return a.Set(s)
}

// Subnet represents an IP network.
type Subnet net.IPNet

// NewSubnet constructs zero valued Subnet pointer.
func NewSubnet() *Subnet {
	return &Subnet{}
}

// Contains validates ability of a client to rich resource.
func (sn *Subnet) Contains(ip net.IP) bool {
	// infer to avoid infinite recursive call.
	return (*net.IPNet)(sn).Contains(ip)
}

// String returns string representation of the IP network.
func (sn *Subnet) String() string {
	// infer to avoid infinite recursive call.
	return (*net.IPNet)(sn).String()
}

// Set parses CIDR and sets network.
func (sn *Subnet) Set(s string) error {
	// in case of empty trusted subnet leave the Subnet value set to zero
	// to ensure that the call [*net.IPNet.Contains(IP) bool]
	// will not return true to any IP address.
	if len(s) == 0 {
		return nil
	}
	// validate and set CIDR.
	_, ipNet, err := net.ParseCIDR(s)
	if err != nil {
		return err
	}
	*sn = Subnet(*ipNet)
	return nil
}

// SetValue implements cleanenv value setter.
func (sn *Subnet) SetValue(s string) error {
	return sn.Set(s)
}

// Enabled implements general setter for boolean values.
// Implements cleanenv value setter.
type Enabled bool

// Set sets Enabled value from string.
func (tls *Enabled) Set(s string) error {
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
func (tls *Enabled) SetValue(s string) error {
	return tls.Set(s)
}

// String returns a string representation of the Enabled value.
func (tls *Enabled) String() string {
	return fmt.Sprintf("%v", *tls)
}

// Order of loading configuration:
// 1. Config file (YAML, JSON supported)
// 2. Flags
// 3. Environment variables

// Load returns an application configuration which is populated
// from the given configuration file, environment variables and flags.
func MustLoad() *Config {
	var cfg Config
	// Setup default values.
	cfg.Server.RunAddress = NewNetAddress()
	cfg.Server.ReturnAddress = NewNetAddress()
	cfg.TrustedSubnet = NewSubnet()
	cfg.FileStoragePath = defaultFileStoragePath
	cfg.Logger.Path = defaultLogPath
	cfg.Logger.MaxSizeMB = defaultMaxLogSizeMB
	cfg.Logger.MaxBackups = defaultMaxLogBackups
	cfg.Logger.MaxAgeDays = defaultMaxLogFileLifetimeDays
	cfg.Migrations = defaultMigtationsPath
	cfg.DeleteBufLen = defaultDeleteBufLen

	// Configuration file path.
	configPath, set := os.LookupEnv("CONFIG")

	if set {
		// Check if file exists.
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			log.Fatalf("config file does not exist: %v", err)
		}

		// Load from config file.
		file, err := os.Open(configPath)
		if err != nil {
			log.Fatalf("failed to open config file: %v", err)
		}

		// Support different file extensions.
		ext := filepath.Ext(configPath)
		switch ext {
		case ".yaml", ".yml":
			if err = cleanenv.ParseYAML(file, &cfg); err != nil {
				log.Fatalf("failed to parse config file: %v", err)
			}
		case ".json":
			if err = cleanenv.ParseJSON(file, &cfg); err != nil {
				log.Fatalf("failed to parse config file: %v", err)
			}
		default:
			log.Fatalf("unsupported configuration file extension: %q", ext)
		}
	}

	// Read given flags. If not provided use file values.
	flag.Var(cfg.Server.RunAddress, "a", "server start address in form host:port")
	flag.Var(cfg.Server.ReturnAddress, "b", "server return address in form host:port")
	flag.Var(cfg.TrustedSubnet, "t", "trusted subnet (CIDR)")
	flag.Var(&cfg.TLSEnabled, "s", "run the server in TLS mode")
	flag.Var(&cfg.RPCEnabled, "r", "run the server in RPC mode")
	flag.StringVar(&cfg.FileStoragePath, "f", cfg.FileStoragePath, "file storage path")
	flag.StringVar(&cfg.DSN, "d", cfg.DSN, "server data source name")
	flag.StringVar(&cfg.Logger.Level, "l", cfg.Logger.Level, "logging level")
	flag.StringVar(&cfg.Migrations, "m", cfg.Migrations, "path to migration directory")
	flag.Parse()

	// Read environment variables.
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		log.Fatalf("failed to read environment variables: %v", err)
	}

	return &cfg
}

// NewForTest returns application configuration for testing.
func NewForTest() *Config {
	return &Config{
		DSN: "",
		Server: Server{
			RunAddress:      NewNetAddress(),
			ReturnAddress:   NewNetAddress(),
			Timeout:         5 * time.Second,
			IdleTimeout:     60 * time.Second,
			ShutdownTimeout: 30 * time.Second,
		},
		FileStoragePath: defaultFileStoragePath,
		JWT: JWT{
			SigningKey: "test",
			Expiration: 10 * time.Minute,
		},
		DeleteBufLen: defaultDeleteBufLen,
	}
}
