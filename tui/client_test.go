package tui

import (
	"testing"

	"github.com/ilyabrin/disk"
)

func TestNewClientUsesYandexByDefault(t *testing.T) {
	t.Setenv(EnvAPIURL, "")
	c, err := NewClient("token")
	if err != nil {
		t.Fatal(err)
	}
	if c.Config.BaseURL != disk.API_URL {
		t.Errorf("BaseURL = %q, want the Yandex endpoint %q", c.Config.BaseURL, disk.API_URL)
	}
}

func TestNewClientHonoursTheAPIURLVariable(t *testing.T) {
	for in, want := range map[string]string{
		"http://127.0.0.1:8080/v1/disk":  "http://127.0.0.1:8080/v1/disk/",
		"http://127.0.0.1:8080/v1/disk/": "http://127.0.0.1:8080/v1/disk/",
		"https://proxy.example/disk///":  "https://proxy.example/disk/",
	} {
		t.Setenv(EnvAPIURL, in)
		c, err := NewClient("token")
		if err != nil {
			t.Fatalf("%q: %v", in, err)
		}
		if c.Config.BaseURL != want {
			t.Errorf("%q: BaseURL = %q, want %q", in, c.Config.BaseURL, want)
		}
	}
}

func TestNewClientRejectsAMalformedAPIURL(t *testing.T) {
	for _, bad := range []string{"not a url", "ftp://example.com/disk", "http://", "/v1/disk"} {
		t.Setenv(EnvAPIURL, bad)
		if _, err := NewClient("token"); err == nil {
			t.Errorf("%q should be rejected", bad)
		}
	}
}
