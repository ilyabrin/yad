package dns

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestParseReadsNameservers(t *testing.T) {
	conf := `# Termux
nameserver 8.8.8.8
nameserver 2001:4860:4860::8888
search lan
nameserver not-an-ip
nameserver
`
	got := parse(strings.NewReader(conf))
	want := []string{"8.8.8.8", "2001:4860:4860::8888"}
	if !slices.Equal(got, want) {
		t.Errorf("parse = %v, want %v", got, want)
	}
}

func termuxPrefix(t *testing.T, conf string) string {
	t.Helper()
	prefix := t.TempDir()
	if err := os.MkdirAll(filepath.Join(prefix, "etc"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(prefix, "etc", "resolv.conf"), []byte(conf), 0o600); err != nil {
		t.Fatal(err)
	}
	return prefix
}

func TestPick(t *testing.T) {
	termux := termuxPrefix(t, "nameserver 1.1.1.1\n")
	empty := termuxPrefix(t, "# nothing here\n")

	cases := []struct {
		name       string
		goos       string
		systemFile bool
		prefix     string
		want       []string
	}{
		{"desktop Linux keeps the system file", "linux", true, "", nil},
		{"Termux with a system file keeps it", "android", true, termux, nil},
		{"Termux, android build", "android", false, termux, []string{"1.1.1.1"}},
		{"Termux, linux build", "linux", false, termux, []string{"1.1.1.1"}},
		{"Android without Termux falls back", "android", false, "", androidDefaults},
		{"Termux file without servers falls back", "android", false, empty, androidDefaults},
		{"Linux without any file is left alone", "linux", false, "", nil},
		{"other systems are left alone", "windows", false, termux, nil},
	}
	for _, tc := range cases {
		if got := pick(tc.goos, tc.systemFile, tc.prefix); !slices.Equal(got, tc.want) {
			t.Errorf("%s: pick = %v, want %v", tc.name, got, tc.want)
		}
	}
}
