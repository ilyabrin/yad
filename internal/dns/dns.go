// Package dns makes name lookups work on Android, where Go cannot find a DNS
// server on its own.
//
// Release builds have no cgo, so Go resolves names itself and reads the
// servers from /etc/resolv.conf. Android has no such file, and Go then falls
// back to 127.0.0.1:53, where nothing answers, so every request to Yandex
// fails. Termux keeps its own copy under $PREFIX/etc/resolv.conf; when that is
// missing too, we use the servers Termux itself ships by default.
package dns

import (
	"bufio"
	"context"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
)

// androidDefaults are the servers in Termux's stock resolv.conf.
var androidDefaults = []string{"8.8.8.8", "8.8.4.4"}

// Setup points the default resolver at working DNS servers when the system
// file is missing. It does nothing where /etc/resolv.conf exists, which covers
// every desktop and server system.
func Setup(goos string) {
	servers := pick(goos, fileExists("/etc/resolv.conf"), os.Getenv("PREFIX"))
	if len(servers) == 0 {
		return
	}
	var next atomic.Uint32
	var d net.Dialer
	net.DefaultResolver = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
			// Rotate, so a dead first server does not fail every lookup.
			server := servers[int(next.Add(1)-1)%len(servers)]
			return d.DialContext(ctx, network, net.JoinHostPort(server, "53"))
		},
	}
}

// pick decides which servers to use, or none to leave Go's default alone.
func pick(goos string, haveSystemFile bool, prefix string) []string {
	if haveSystemFile || (goos != "android" && goos != "linux") {
		return nil
	}
	if prefix != "" {
		if f, err := os.Open(filepath.Join(prefix, "etc", "resolv.conf")); err == nil {
			defer func() { _ = f.Close() }()
			if servers := parse(f); len(servers) > 0 {
				return servers
			}
		}
	}
	if goos == "android" {
		return androidDefaults
	}
	return nil
}

// parse returns the nameserver addresses in a resolv.conf.
func parse(r io.Reader) []string {
	var servers []string
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) >= 2 && fields[0] == "nameserver" && net.ParseIP(fields[1]) != nil {
			servers = append(servers, fields[1])
		}
	}
	return servers
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
