package tui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/ilyabrin/disk"
)

// newTestClient points a disk.Client at a local test server.
func newTestClient(t *testing.T, srv *httptest.Server) *disk.Client {
	t.Helper()
	cfg := disk.DefaultClientConfig()
	cfg.BaseURL = srv.URL + "/"
	cfg.MaxRetries = 0
	client, err := disk.NewWithConfig(cfg, "test-token")
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func TestCmdMkdirCreatesNestedPath(t *testing.T) {
	var mu sync.Mutex
	var created []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		created = append(created, r.URL.Query().Get("path"))
		mu.Unlock()
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(disk.Link{Href: "https://example.invalid", Method: http.MethodGet})
	}))
	defer srv.Close()

	msg := cmdMkdir(newTestClient(t, srv), "/trips/2026/iceland")()
	done, ok := msg.(mkdirDoneMsg)
	if !ok {
		t.Fatalf("expected mkdirDoneMsg, got %T", msg)
	}
	if done.err != nil {
		t.Fatalf("unexpected error: %v", done.err)
	}

	want := []string{"/trips", "/trips/2026", "/trips/2026/iceland"}
	if len(created) != len(want) {
		t.Fatalf("expected one request per level %v, got %v", want, created)
	}
	for i, path := range want {
		if created[i] != path {
			t.Errorf("request %d: expected %q, got %q", i, path, created[i])
		}
	}
}

func TestCmdMkdirReportsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInsufficientStorage)
		_ = json.NewEncoder(w).Encode(disk.ErrorResponse{
			Message: "Not enough free space",
			Error:   "DiskResourceUploadFailedError",
		})
	}))
	defer srv.Close()

	done := cmdMkdir(newTestClient(t, srv), "/full")().(mkdirDoneMsg)
	if done.err == nil {
		t.Fatal("expected the server error to be reported")
	}
	if !strings.Contains(done.err.Error(), "Not enough free space") {
		t.Errorf("expected the API message to survive, got %q", done.err)
	}
}
