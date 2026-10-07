package tui

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/ilyabrin/disk"
)

// EnvAPIURL names the environment variable that points yad at a different
// Yandex.Disk API endpoint: a proxy, or a local server for tests and demos.
// The access token is sent there too, so it must be something you trust.
const EnvAPIURL = "YANDEX_DISK_API_URL"

// NewClient creates the Yandex.Disk client every part of yad uses, honouring
// EnvAPIURL when it is set.
func NewClient(token string) (*disk.Client, error) {
	raw := os.Getenv(EnvAPIURL)
	if raw == "" {
		return disk.New(token)
	}

	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("%s must be an http or https URL, got %q", EnvAPIURL, raw)
	}

	cfg := disk.DefaultClientConfig()
	cfg.BaseURL = strings.TrimRight(raw, "/") + "/" // the client expects a trailing slash
	return disk.NewWithConfig(cfg, token)
}
