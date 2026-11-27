// Package loopbacktest stops tests from reaching the network beyond loopback.
package loopbacktest

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
)

// ErrNonLoopback is returned when a test dials a host other than loopback.
var ErrNonLoopback = errors.New("non-loopback host refused in tests")

// Suite is what TestMain runs; *testing.M satisfies it.
type Suite interface{ Run() int }

// Run makes http.DefaultTransport refuse non-loopback hosts, then runs the suite and returns its exit code.
func Run(s Suite) int {
	install()
	return s.Run()
}

func install() {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.Proxy = nil
	var d net.Dialer
	t.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, _, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		if !isLoopback(host) {
			return nil, fmt.Errorf("%w: %s", ErrNonLoopback, host)
		}
		return d.DialContext(ctx, network, addr)
	}
	http.DefaultTransport = t
}

func isLoopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
