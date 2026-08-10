package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ilyabrin/disk"
)

// namedEntries builds a page of file entries with the given names.
func namedEntries(names ...string) []entry {
	entries := make([]entry, len(names))
	for i, n := range names {
		entries[i] = entry{resource: &disk.Resource{
			Path: "disk:/" + n,
			Name: n,
			Type: "file",
		}}
	}
	return entries
}

// Regression: with a filter active the view renders visibleEntries(), so the
// cursor indexes the *filtered* list. Actions used to index m.entries directly
// and therefore targeted the wrong resource — deleting or renaming a file the
// user could not even see.
func TestCurrentEntry_FollowsFilter(t *testing.T) {
	m := newBrowserWithEntries(namedEntries("alpha.txt", "beta.txt", "gamma.txt", "beta2.txt"))
	m.filter = "beta"
	m.cursor = 1 // second *match* → "beta2.txt"

	e, ok := m.currentEntry()
	if !ok {
		t.Fatal("currentEntry() returned ok=false, want an entry")
	}
	if e.resource.Name != "beta2.txt" {
		t.Errorf("currentEntry() = %q, want %q", e.resource.Name, "beta2.txt")
	}
}

func TestCurrentEntry_OutOfRange(t *testing.T) {
	m := newBrowserWithEntries(namedEntries("a.txt"))
	m.cursor = 5

	if _, ok := m.currentEntry(); ok {
		t.Error("currentEntry() returned ok=true for an out-of-range cursor")
	}
}

func TestCurrentEntry_EmptyList(t *testing.T) {
	m := newBrowserWithEntries(nil)

	if _, ok := m.currentEntry(); ok {
		t.Error("currentEntry() returned ok=true for an empty list")
	}
}

func TestDeleteConfirm_TargetsFilteredEntry(t *testing.T) {
	m := newBrowserWithEntries(namedEntries("alpha.txt", "beta.txt"))
	m.filter = "beta"
	m.cursor = 0

	updated, _ := pressKey(m, "D")
	if updated.mode != modeConfirmDelete {
		t.Fatalf("mode = %v, want modeConfirmDelete", updated.mode)
	}
	// The dialog must name the file the user is looking at.
	if got := updated.confirm.View(60); !strings.Contains(got, "beta.txt") {
		t.Errorf("confirm dialog does not mention beta.txt:\n%s", got)
	}
}

func TestDown_StopsAtEndOfFilteredList(t *testing.T) {
	m := newBrowserWithEntries(namedEntries("alpha.txt", "beta.txt", "gamma.txt"))
	m.filter = "beta"
	m.total = 3
	m.cursor = 0

	updated, cmd := pressKey(m, "j")
	if updated.cursor != 0 {
		t.Errorf("cursor = %d, want 0 (only one match)", updated.cursor)
	}
	if cmd != nil {
		t.Error("expected no page-load cmd while a filter is active")
	}
}

func TestSelectAll_OnlySelectsVisibleEntries(t *testing.T) {
	m := newBrowserWithEntries(namedEntries("alpha.txt", "beta.txt", "gamma.txt"))
	m.filter = "beta"

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlA})
	if len(updated.selected) != 1 {
		t.Fatalf("selected %d entries, want 1", len(updated.selected))
	}
	if !updated.selected["disk:/beta.txt"] {
		t.Errorf("selected = %v, want only disk:/beta.txt", updated.selected)
	}
}

func TestClampCursor_AfterFilterNarrows(t *testing.T) {
	m := newBrowserWithEntries(namedEntries("alpha.txt", "beta.txt", "gamma.txt"))
	m.cursor = 2
	m.filter = "beta"
	m.clampCursor()

	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0 after the filter narrowed the list", m.cursor)
	}
}
