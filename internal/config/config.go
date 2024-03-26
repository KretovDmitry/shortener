package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

const (
	defaultHost     = "0.0.0.0"
	defaultPort     = 8080
	defaultFileName = "short-url-db.json"
)

type netAddress struct {
	Host string
	Port int
}

// NewNetAddress returns pointer to a new netAddress with default Host and Port
func NewNetAddress() *netAddress {
	return &netAddress{
		Host: defaultHost,
		Port: defaultPort,
	}
}

func (a netAddress) String() string {
	return fmt.Sprintf("%s:%d", a.Host, a.Port)
}

func (a *netAddress) Set(s string) error {
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

type fileStorage struct {
	path          string
	writeRequired bool
}

func NewFileStorage() *fileStorage {
	tmp := os.TempDir()
	return &fileStorage{
		path:          path.Join(tmp, defaultFileName),
		writeRequired: true,
	}
}

func (fs *fileStorage) String() string {
	return fs.path
}

func (fs *fileStorage) Set(s string) error {
	if s == "" {
		fs.writeRequired = false
		return nil
	}

	fs.path = s

	return nil
}

func (fs *fileStorage) Path() string {
	return fs.path
}

func (fs *fileStorage) WriteRequired() bool {
	return fs.writeRequired
}

type JWTExpiration struct {
	Expiration time.Duration
}

func NewJWTExpiration() *JWTExpiration {
	return &JWTExpiration{
		Expiration: time.Hour * 3,
	}
}

func (e *JWTExpiration) String() string {
	return e.Expiration.String()
}

func (e *JWTExpiration) Set(s string) error {
	d, err := time.ParseDuration(s)
	if err != nil {
		return err
	}

	e.Expiration = d

	return nil
}

var (
	AddrToRun    = NewNetAddress()
	AddrToReturn = NewNetAddress()
	FileStorage  = NewFileStorage()
	DSN          string
	LogLevel     string
	Secret       string
	JWT          = NewJWTExpiration()
	MigrationDir string
)

func ParseFlags() error {
	// flags take precedence over the default values
	flag.Var(AddrToRun, "a", "Net address host:port to run server")
	flag.Var(AddrToReturn, "b", "Net address host:port to return short URLs")
	flag.Var(FileStorage, "f", "File storage path")
	flag.Var(JWT, "j", `JWT lifetime in form of time.Duration: such as "2h45m".`)
	flag.StringVar(&DSN, "d", "", "Data source name in form postgres URL or DSN string")
	flag.StringVar(&LogLevel, "l", "info", "Log level")
	flag.StringVar(&Secret, "s", "test", "Secret key for JWT")
	flag.StringVar(&MigrationDir, "m", ".", "Path to migration directory")
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
		if err := FileStorage.Set(envFileStoragePath); err != nil {
			return fmt.Errorf("invalid FILE_STORAGE_PATH: %w", err)
		}
	}

	if envDSN := os.Getenv("DATABASE_DSN"); envDSN != "" {
		DSN = envDSN
	}

	if envLogLevel := os.Getenv("LOG_LEVEL"); envLogLevel != "" {
		LogLevel = envLogLevel
	}

	return nil
}
