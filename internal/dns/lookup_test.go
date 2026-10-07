package dns

import (
	"context"
	"net"
	"os"
	"runtime"
	"testing"
	"time"
)

// TestLookupOnThisDevice resolves the Yandex API host for real. It needs the
// network, so it only runs when YAD_NETWORK_TEST is set; CI sets it inside an
// Android emulator, where Go alone cannot find a DNS server.
func TestLookupOnThisDevice(t *testing.T) {
	if os.Getenv("YAD_NETWORK_TEST") == "" {
		t.Skip("set YAD_NETWORK_TEST=1 to resolve a real host name")
	}
	Setup(runtime.GOOS)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	addrs, err := net.DefaultResolver.LookupHost(ctx, "cloud-api.yandex.net")
	if err != nil {
		t.Fatalf("lookup failed on %s: %v", runtime.GOOS, err)
	}
	t.Logf("cloud-api.yandex.net -> %v", addrs)
}
