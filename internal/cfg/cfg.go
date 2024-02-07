package cfg

import (
	"errors"
	"flag"
	"strconv"
	"strings"
)

const (
	_defaultHost = "0.0.0.0"
	_defaultPort = 8080
)

type netAddress struct {
	Host string
	Port int
}

func (a netAddress) String() string {
	return a.Host + ":" + strconv.Itoa(a.Port)
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

var (
	AddrToRun    = new(netAddress)
	AddrToReturn = new(netAddress)
)

func ParseFlags() {
	// compile time flag.Value interface implementation check
	_ = flag.Value(AddrToRun)
	_ = flag.Value(AddrToReturn)

	AddrToRun.Host = _defaultHost
	AddrToRun.Port = _defaultPort

	AddrToReturn.Host = _defaultHost
	AddrToReturn.Port = _defaultPort

	flag.Var(AddrToRun, "a", "Net address host:port to run server")
	flag.Var(AddrToReturn, "b", "Net address host:port to return short URLs")
	flag.Parse()
}
