package tui

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ilyabrin/disk"
)

// pressKey sends a rune key to the model's handleKey and returns the result.
func pressKey(m BrowserModel, s string) (BrowserModel, tea.Cmd) {
	return m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)})
}

// pressSpecialKey sends a special (non-rune) key such as Up/Down/Enter.
func pressSpecialKey(m BrowserModel, t tea.KeyType) (BrowserModel, tea.Cmd) {
	return m.Update(tea.KeyMsg{Type: t})
}

func newBrowserWithEntries(entries []entry) BrowserModel {
	m := newTestBrowser()
	m.keys = DefaultBrowserKeyMap()
	m.entries = entries
	m.total = len(entries)
	return m
}

// --- Cursor movement ---

func TestHandleKey_Up_DecrementsCursor(t *testing.T) {
	m := newBrowserWithEntries(makeEntries(3))
	m.cursor = 2

	updated, _ := pressSpecialKey(m, tea.KeyUp)
	if updated.cursor != 1 {
		t.Errorf("cursor = %d, want 1", updated.cursor)
	}
}

func TestHandleKey_Up_AtTopDoesNothing(t *testing.T) {
	m := newBrowserWithEntries(makeEntries(3))
	m.cursor = 0

	updated, cmd := pressSpecialKey(m, tea.KeyUp)
	if updated.cursor != 0 {
		t.Errorf("cursor = %d, want 0", updated.cursor)
	}
	if cmd != nil {
		t.Error("expected no cmd when already at top with no previous page")
	}
}

func TestHandleKey_Down_IncrementsCursor(t *testing.T) {
	m := newBrowserWithEntries(makeEntries(3))
	m.cursor = 0

	updated, _ := pressSpecialKey(m, tea.KeyDown)
	if updated.cursor != 1 {
		t.Errorf("cursor = %d, want 1", updated.cursor)
	}
}

func TestHandleKey_Down_AtBottomDoesNothing(t *testing.T) {
	m := newBrowserWithEntries(makeEntries(3))
	m.cursor = 2 // last item, total == len(entries) → no next page

	updated, cmd := pressSpecialKey(m, tea.KeyDown)
	if updated.cursor != 2 {
		t.Errorf("cursor = %d, want 2", updated.cursor)
	}
	if cmd != nil {
		t.Error("expected no cmd when at bottom with no next page")
	}
}

// --- Enter (open directory) ---

func TestHandleKey_Enter_OpensDir(t *testing.T) {
	dir := entry{resource: &disk.Resource{
		Path: "disk:/photos",
		Name: "photos",
		Type: "dir",
	}}
	m := newBrowserWithEntries([]entry{dir})
	m.cursor = 0

	updated, cmd := pressSpecialKey(m, tea.KeyEnter)
	if updated.path != "disk:/photos" {
		t.Errorf("path = %q, want disk:/photos", updated.path)
	}
	if updated.offset != 0 {
		t.Errorf("offset should reset to 0, got %d", updated.offset)
	}
	if !updated.loading {
		t.Error("loading should be true after entering a directory")
	}
	if cmd == nil {
		t.Error("expected reload cmd")
	}
}

func TestHandleKey_Enter_DoesNothingOnFile(t *testing.T) {
	file := entry{resource: &disk.Resource{
		Path: "disk:/photo.jpg",
		Name: "photo.jpg",
		Type: "file",
	}}
	m := newBrowserWithEntries([]entry{file})
	m.cursor = 0

	updated, cmd := pressSpecialKey(m, tea.KeyEnter)
	if updated.path != m.path {
		t.Error("path should not change when entering a file")
	}
	if cmd != nil {
		t.Error("expected no cmd when pressing enter on a file")
	}
}

// --- Back (go to parent) ---

func TestHandleKey_Back_GoesToParent(t *testing.T) {
	m := newBrowserWithEntries(makeEntries(1))
	m.path = "disk:/photos/vacation"

	updated, cmd := pressSpecialKey(m, tea.KeyLeft)
	if updated.path != "disk:/photos" {
		t.Errorf("path = %q, want disk:/photos", updated.path)
	}
	if updated.offset != 0 {
		t.Errorf("offset should reset to 0, got %d", updated.offset)
	}
	if cmd == nil {
		t.Error("expected reload cmd")
	}
}

func TestHandleKey_Back_AtRootDoesNothing(t *testing.T) {
	m := newBrowserWithEntries(makeEntries(1))
	m.path = "/"

	updated, cmd := pressSpecialKey(m, tea.KeyLeft)
	if updated.path != "/" {
		t.Errorf("path = %q, want /", updated.path)
	}
	if cmd != nil {
		t.Error("expected no cmd at root")
	}
}

// --- Sort cycling ---

func TestHandleKey_Sort_CyclesSort(t *testing.T) {
	m := newBrowserWithEntries(makeEntries(1))
	m.sort = "name"

	updated, cmd := pressKey(m, "s")
	if updated.sort == "name" {
		t.Error("sort should have changed")
	}
	if updated.offset != 0 {
		t.Errorf("offset should reset on sort change, got %d", updated.offset)
	}
	if cmd == nil {
		t.Error("expected reload cmd after sort change")
	}
}

// --- Error cleared on keypress ---

func TestHandleKey_ClearsErrorOnAnyKey(t *testing.T) {
	m := newBrowserWithEntries(makeEntries(1))
	m.err = errors.New("some error")

	updated, cmd := pressKey(m, "j")
	if updated.err != nil {
		t.Error("err should be cleared on keypress")
	}
	if cmd != nil {
		t.Error("expected no cmd after clearing error")
	}
}

// --- Esc clears selection in normal mode ---

func TestHandleKey_Esc_ClearsSelection(t *testing.T) {
	m := newBrowserWithEntries(makeEntries(3))
	m.selected = map[string]bool{"disk:/file0": true, "disk:/file1": true}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if updated.selected != nil {
		t.Errorf("selected should be nil after esc, got %v", updated.selected)
	}
}

// --- Message mode cleared on any key ---

func TestHandleKey_MessageMode_ClearedOnKey(t *testing.T) {
	m := newBrowserWithEntries(makeEntries(1))
	m.mode = modeMessage
	m.message = "some message"

	updated, _ := pressKey(m, "j")
	if updated.mode != modeNormal {
		t.Errorf("mode = %v, want modeNormal", updated.mode)
	}
	if updated.message != "" {
		t.Error("message should be cleared")
	}
}

// --- Refresh ---

func TestHandleKey_Refresh_TriggersReload(t *testing.T) {
	m := newBrowserWithEntries(makeEntries(3))

	updated, cmd := pressKey(m, "R")
	if !updated.loading {
		t.Error("loading should be true after refresh")
	}
	if cmd == nil {
		t.Error("expected reload cmd")
	}
}
