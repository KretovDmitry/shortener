package cfg

import (
	"flag"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

const (
	defaultHost = "0.0.0.0"
	defaultPort = 8080
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
	path     string
	required bool
}

func NewFileStorage() *fileStorage {
	const defaultFileName = "short-url-db.json"
	tmp := os.TempDir()
	return &fileStorage{
		path:     path.Join(tmp, defaultFileName),
		required: true,
	}
}

func (fs *fileStorage) String() string {
	return fs.path
}

func (fs *fileStorage) Set(s string) error {
	if s == "" {
		fs.required = false
		return nil
	}

	fs.path = s

	return nil
}

func (fs *fileStorage) Path() string {
	return fs.path
}

func (fs *fileStorage) Required() bool {
	return fs.required
}

var (
	AddrToRun    = NewNetAddress()
	AddrToReturn = NewNetAddress()
	FileStorage  = NewFileStorage()
	LogLevel     string
)

func ParseFlags() error {
	// flags take precedence over the default values
	flag.Var(AddrToRun, "a", "Net address host:port to run server")
	flag.Var(AddrToReturn, "b", "Net address host:port to return short URLs")
	flag.Var(FileStorage, "f", "file storage path")
	flag.StringVar(&LogLevel, "l", "info", "log level")
	flag.Parse()

	// ENV variables have the highest priority
	if envRunAddr := os.Getenv("SERVER_ADDRESS"); envRunAddr != "" {
		err := AddrToRun.Set(envRunAddr)
		if err != nil {
			return errors.Wrap(err, "invalid SERVER_ADDRESS")
		}
	}

	if envReturnAddr := os.Getenv("BASE_URL"); envReturnAddr != "" {
		err := AddrToReturn.Set(envReturnAddr)
		if err != nil {
			return errors.Wrap(err, "invalid BASE_URL")
		}
	}

	envFileStoragePath, set := os.LookupEnv("FILE_STORAGE_PATH")
	if set {
		err := FileStorage.Set(envFileStoragePath)
		return errors.Wrap(err, "invalid FILE_STORAGE_PATH")
	}

	if envLogLevel := os.Getenv("LOG_LEVEL"); envLogLevel != "" {
		LogLevel = envLogLevel
	}

	return nil
}
