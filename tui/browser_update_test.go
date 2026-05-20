package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ilyabrin/disk"
)

// makeEntries builds n stub entries with paths "/file0", "/file1", ...
func makeEntries(n int) []entry {
	entries := make([]entry, n)
	for i := range entries {
		entries[i] = entry{resource: &disk.Resource{
			Path: "/file" + string(rune('0'+i)),
			Name: "file" + string(rune('0'+i)),
			Type: "file",
		}}
	}
	return entries
}

// newTestBrowser returns a BrowserModel suitable for unit tests (no live client).
func newTestBrowser() BrowserModel {
	return BrowserModel{
		path: "disk:/",
		sort: "name",
	}
}

// --- loadedMsg / cursorAfterLoad ---

func TestLoadedMsg_CursorAfterLoad(t *testing.T) {
	tests := []struct {
		name            string
		cursorAfterLoad int
		entries         int
		wantCursor      int
	}{
		{"explicit 0", 0, 5, 0},
		{"explicit mid", 2, 5, 2},
		{"explicit last", 4, 5, 4},
		{"explicit out of range → 0", 10, 5, 0},
		{"negative → last item", -1, 5, 4},
		{"negative empty list → 0", -1, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestBrowser()
			m.cursorAfterLoad = tt.cursorAfterLoad

			msg := loadedMsg{
				path:    "disk:/",
				entries: makeEntries(tt.entries),
				total:   tt.entries,
			}
			updated, _ := m.Update(msg)
			if updated.cursor != tt.wantCursor {
				t.Errorf("cursor = %d, want %d", updated.cursor, tt.wantCursor)
			}
			if updated.loading {
				t.Error("loading should be false after loadedMsg")
			}
		})
	}
}

func TestLoadedMsg_Error(t *testing.T) {
	m := newTestBrowser()
	msg := loadedMsg{path: "disk:/", err: errTest("load failed")}

	updated, cmd := m.Update(msg)
	if updated.err == nil {
		t.Error("err should be set")
	}
	if updated.loading {
		t.Error("loading should be false")
	}
	if cmd != nil {
		t.Error("cmd should be nil on error")
	}
}

// --- deleteDoneMsg ---

func TestDeleteDoneMsg_ContinuesChain(t *testing.T) {
	m := newTestBrowser()
	m.pendingDelete = []string{"disk:/b", "disk:/c"}

	updated, cmd := m.Update(deleteDoneMsg{})
	if len(updated.pendingDelete) != 1 || updated.pendingDelete[0] != "disk:/c" {
		t.Errorf("pendingDelete = %v, want [disk:/c]", updated.pendingDelete)
	}
	if cmd == nil {
		t.Error("expected next cmdDelete, got nil")
	}
}

func TestDeleteDoneMsg_LastItem_Reloads(t *testing.T) {
	m := newTestBrowser()
	m.pendingDelete = nil

	updated, cmd := m.Update(deleteDoneMsg{})
	if updated.mode != modeNormal {
		t.Errorf("mode = %v, want modeNormal", updated.mode)
	}
	if updated.selected != nil {
		t.Error("selected should be cleared")
	}
	if !updated.loading {
		t.Error("loading should be true")
	}
	if cmd == nil {
		t.Error("expected reload cmd, got nil")
	}
}

func TestDeleteDoneMsg_Error_ClearsPending(t *testing.T) {
	m := newTestBrowser()
	m.pendingDelete = []string{"disk:/b", "disk:/c"}

	updated, cmd := m.Update(deleteDoneMsg{err: errTest("permission denied")})
	if len(updated.pendingDelete) != 0 {
		t.Errorf("pendingDelete should be cleared on error, got %v", updated.pendingDelete)
	}
	if updated.mode != modeMessage {
		t.Errorf("mode = %v, want modeMessage", updated.mode)
	}
	if cmd != nil {
		t.Error("cmd should be nil on error")
	}
}

// --- downloadDoneMsg ---

func TestDownloadDoneMsg_ContinuesChain(t *testing.T) {
	m := newTestBrowser()
	m.pendingDownloads = []string{"disk:/b.txt", "disk:/c.txt"}
	m.downloadDir = "/tmp"

	updated, cmd := m.Update(downloadDoneMsg{})
	if len(updated.pendingDownloads) != 1 || updated.pendingDownloads[0] != "disk:/c.txt" {
		t.Errorf("pendingDownloads = %v, want [disk:/c.txt]", updated.pendingDownloads)
	}
	if cmd == nil {
		t.Error("expected next download cmd, got nil")
	}
}

func TestDownloadDoneMsg_LastItem_Done(t *testing.T) {
	m := newTestBrowser()
	m.pendingDownloads = nil
	m.downloadDir = "/tmp"
	m.selected = map[string]bool{"disk:/a.txt": true}

	updated, _ := m.Update(downloadDoneMsg{})
	if updated.selected != nil {
		t.Error("selected should be cleared after last download")
	}
	if updated.downloadDir != "" {
		t.Errorf("downloadDir should be cleared, got %q", updated.downloadDir)
	}
	if !updated.progress.Done {
		t.Error("progress.Done should be true")
	}
}

func TestDownloadDoneMsg_Error_ClearsPending(t *testing.T) {
	m := newTestBrowser()
	m.pendingDownloads = []string{"disk:/b.txt"}

	updated, cmd := m.Update(downloadDoneMsg{err: errTest("disk full")})
	if len(updated.pendingDownloads) != 0 {
		t.Errorf("pendingDownloads should be cleared on error, got %v", updated.pendingDownloads)
	}
	if !updated.progress.Done {
		t.Error("progress.Done should be true on error")
	}
	if updated.progress.Err == nil {
		t.Error("progress.Err should be set")
	}
	if cmd != nil {
		t.Error("cmd should be nil on error")
	}
}

// --- helpers ---

type errTest string

func (e errTest) Error() string { return string(e) }

// unwrapCmd runs a tea.Cmd and returns the message it produces (nil-safe).
func unwrapCmd(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	return cmd()
}
