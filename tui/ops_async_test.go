package tui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

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

// uploadURLServer serves the 202-then-poll shape of the upload-from-URL flow.
type uploadURLServer struct {
	mu             sync.Mutex
	polls          int
	pollsBeforeEnd int
	finalStatus    string
}

func (u *uploadURLServer) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "operations") {
			u.mu.Lock()
			u.polls++
			done := u.polls > u.pollsBeforeEnd
			status := disk.OperationInProgress
			if done {
				status = u.finalStatus
			}
			u.mu.Unlock()

			_ = json.NewEncoder(w).Encode(disk.Operation{Status: status})
			return
		}

		// The upload request itself: accepted, with an operation to poll.
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(disk.Link{
			Href:   "https://cloud-api.yandex.net/v1/disk/operations/op-123",
			Method: http.MethodGet,
		})
	}
}

func TestCmdUploadFromURLWaitsForOperation(t *testing.T) {
	u := &uploadURLServer{pollsBeforeEnd: 2, finalStatus: operationSuccess}
	srv := httptest.NewServer(u.handler())
	defer srv.Close()

	done := cmdUploadFromURL(newTestClient(t, srv), "https://example.com/f.bin", "/f.bin")().(uploadFromURLDoneMsg)
	if done.err != nil {
		t.Fatalf("unexpected error: %v", done.err)
	}
	if u.polls <= u.pollsBeforeEnd {
		t.Errorf("expected the operation to be polled past in-progress, got %d polls", u.polls)
	}
}

func TestCmdUploadFromURLReportsServerFailure(t *testing.T) {
	u := &uploadURLServer{pollsBeforeEnd: 1, finalStatus: "failed"}
	srv := httptest.NewServer(u.handler())
	defer srv.Close()

	done := cmdUploadFromURL(newTestClient(t, srv), "https://example.com/f.bin", "/f.bin")().(uploadFromURLDoneMsg)
	if done.err == nil {
		t.Fatal("expected a failed operation to be reported as an error")
	}
	if !strings.Contains(done.err.Error(), "failed") {
		t.Errorf("expected the status in the message, got %q", done.err)
	}
}

// A 200 response carries no operation, so there is nothing to poll.
func TestCmdUploadFromURLSkipsPollingWithoutOperation(t *testing.T) {
	var polled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "operations") {
			polled = true
		}
		_ = json.NewEncoder(w).Encode(disk.Link{})
	}))
	defer srv.Close()

	done := cmdUploadFromURL(newTestClient(t, srv), "https://example.com/f.bin", "/f.bin")().(uploadFromURLDoneMsg)
	if done.err != nil {
		t.Fatalf("unexpected error: %v", done.err)
	}
	if polled {
		t.Error("expected no operation polling when the API returned no href")
	}
}

func TestWaitForOperationHonoursContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(disk.Operation{Status: disk.OperationInProgress})
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := waitForOperation(ctx, newTestClient(t, srv), "op-123")
	if err == nil {
		t.Fatal("expected the expired context to end the wait")
	}
}
