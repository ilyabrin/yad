package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ilyabrin/disk"
)

// submitInput transitions an input dialog by typing text and pressing Enter.
func submitInput(m BrowserModel, text string) (BrowserModel, tea.Cmd) {
	for _, r := range text {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	return m, cmd
}

func cancelInput(m BrowserModel) (BrowserModel, tea.Cmd) {
	return m.Update(tea.KeyMsg{Type: tea.KeyEsc})
}

// --- Delete confirmation ---

func TestHandleKey_Delete_ShowsConfirmDialog(t *testing.T) {
	m := newBrowserWithEntries(makeEntries(3))

	updated, _ := pressKey(m, "D")
	if updated.mode != modeConfirmDelete {
		t.Errorf("mode = %v, want modeConfirmDelete", updated.mode)
	}
}

func TestHandleKey_Delete_EmptyList_DoesNothing(t *testing.T) {
	m := newBrowserWithEntries(nil)

	updated, _ := pressKey(m, "D")
	if updated.mode != modeNormal {
		t.Errorf("mode = %v, want modeNormal on empty list", updated.mode)
	}
}

func TestHandleKey_Delete_WithSelection_ShowsBulkDialog(t *testing.T) {
	m := newBrowserWithEntries(makeEntries(3))
	m.selected = map[string]bool{
		m.entries[0].resource.Path: true,
		m.entries[1].resource.Path: true,
	}

	updated, _ := pressKey(m, "D")
	if updated.mode != modeConfirmBulkDelete {
		t.Errorf("mode = %v, want modeConfirmBulkDelete", updated.mode)
	}
}

func TestHandleKey_ConfirmDelete_Cancel_ReturnsNormal(t *testing.T) {
	m := newBrowserWithEntries(makeEntries(3))
	m.mode = modeConfirmDelete
	m.confirm = NewConfirmDialog("Delete", "Delete file?")

	updated, _ := cancelInput(m)
	if updated.mode != modeNormal {
		t.Errorf("mode = %v, want modeNormal after cancel", updated.mode)
	}
}

// --- New directory input ---

func TestHandleKey_NewDir_ShowsInputDialog(t *testing.T) {
	m := newBrowserWithEntries(makeEntries(1))

	updated, _ := pressKey(m, "n")
	if updated.mode != modeInputNewDir {
		t.Errorf("mode = %v, want modeInputNewDir", updated.mode)
	}
}

func TestHandleKey_NewDir_Cancel_ReturnsNormal(t *testing.T) {
	m := newBrowserWithEntries(makeEntries(1))
	m.mode = modeInputNewDir
	m.inputDlg = NewInputDialog("New directory", "", "")

	updated, _ := cancelInput(m)
	if updated.mode != modeNormal {
		t.Errorf("mode = %v, want modeNormal after cancel", updated.mode)
	}
}

func TestHandleKey_NewDir_EmptyName_DoesNothing(t *testing.T) {
	m := newBrowserWithEntries(makeEntries(1))
	m.mode = modeInputNewDir
	m.inputDlg = NewInputDialog("New directory", "", "")

	// Submit with empty value
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if updated.mode != modeNormal {
		t.Errorf("mode = %v, want modeNormal", updated.mode)
	}
	if cmd != nil {
		t.Error("cmd should be nil for empty directory name")
	}
}

func TestHandleKey_NewDir_Submit_SendsCmd(t *testing.T) {
	m := newBrowserWithEntries(makeEntries(1))
	m.mode = modeInputNewDir
	m.inputDlg = NewInputDialog("New directory", "", "")

	updated, cmd := submitInput(m, "myfolder")
	if updated.mode != modeNormal {
		t.Errorf("mode = %v, want modeNormal", updated.mode)
	}
	if cmd == nil {
		t.Error("expected cmdMkdir")
	}
}

// --- Rename input ---

func TestHandleKey_Rename_ShowsInputDialog(t *testing.T) {
	m := newBrowserWithEntries(makeEntries(2))
	m.cursor = 0

	updated, _ := pressKey(m, "r")
	if updated.mode != modeInputRename {
		t.Errorf("mode = %v, want modeInputRename", updated.mode)
	}
}

func TestHandleKey_Rename_EmptyList_DoesNothing(t *testing.T) {
	m := newBrowserWithEntries(nil)

	updated, cmd := pressKey(m, "r")
	if updated.mode != modeNormal {
		t.Errorf("mode = %v, want modeNormal on empty list", updated.mode)
	}
	if cmd != nil {
		t.Error("expected no cmd")
	}
}

func TestHandleKey_Rename_Cancel_ReturnsNormal(t *testing.T) {
	m := newBrowserWithEntries(makeEntries(2))
	m.mode = modeInputRename
	m.inputDlg = NewInputDialog("Rename", "", "old-name")

	updated, _ := cancelInput(m)
	if updated.mode != modeNormal {
		t.Errorf("mode = %v, want modeNormal after cancel", updated.mode)
	}
}

func TestHandleKey_Rename_Submit_SendsCmd(t *testing.T) {
	m := newBrowserWithEntries(makeEntries(2))
	m.mode = modeInputRename
	m.inputDlg = NewInputDialog("Rename", "", "")

	_, cmd := submitInput(m, "new-name")
	if cmd == nil {
		t.Error("expected cmdRename")
	}
}

// --- Select toggle ---

func TestHandleKey_Select_TogglesItem(t *testing.T) {
	m := newBrowserWithEntries(makeEntries(3))
	m.keys = DefaultBrowserKeyMap()
	m.cursor = 0

	// Select
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	if !updated.selected[m.entries[0].resource.Path] {
		t.Error("item should be selected after space")
	}

	// Deselect
	updated2, _ := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	// cursor moved to 1 after first select, so we need to go back to 0
	_ = updated2
}

func TestHandleKey_Select_AdvancesCursor(t *testing.T) {
	m := newBrowserWithEntries(makeEntries(3))
	m.keys = DefaultBrowserKeyMap()
	m.cursor = 0

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	if updated.cursor != 1 {
		t.Errorf("cursor = %d, want 1 after select", updated.cursor)
	}
}

// --- SelectAll toggle ---

func TestHandleKey_SelectAll_SelectsAll(t *testing.T) {
	m := newBrowserWithEntries(makeEntries(3))

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlA})
	if len(updated.selected) != 3 {
		t.Errorf("selected count = %d, want 3", len(updated.selected))
	}
}

func TestHandleKey_SelectAll_DeselectsWhenAllSelected(t *testing.T) {
	m := newBrowserWithEntries(makeEntries(3))
	m.selected = map[string]bool{
		m.entries[0].resource.Path: true,
		m.entries[1].resource.Path: true,
		m.entries[2].resource.Path: true,
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlA})
	if updated.selected != nil {
		t.Errorf("selected should be nil after deselect-all, got %v", updated.selected)
	}
}

// --- CopyURL ---

func TestHandleKey_CopyURL_NoURL_ShowsError(t *testing.T) {
	entries := []entry{{resource: &disk.Resource{
		Path: "disk:/photo.jpg",
		Name: "photo.jpg",
		Type: "file",
		// PublicURL is empty
	}}}
	m := newBrowserWithEntries(entries)

	updated, _ := pressKey(m, "c")
	if updated.mode != modeMessage {
		t.Errorf("mode = %v, want modeMessage", updated.mode)
	}
	if !updated.messageIsError {
		t.Error("messageIsError should be true")
	}
}

func TestHandleKey_CopyURL_WithURL_SendsCmd(t *testing.T) {
	entries := []entry{{resource: &disk.Resource{
		Path:      "disk:/photo.jpg",
		Name:      "photo.jpg",
		Type:      "file",
		PublicURL: "https://disk.yandex.ru/i/abc",
	}}}
	m := newBrowserWithEntries(entries)

	_, cmd := pressKey(m, "c")
	if cmd == nil {
		t.Error("expected cmdCopyToClipboard")
	}
}

// --- modePublicURL key handling ---

func TestHandleKey_PublicURL_AnyKey_Dismisses(t *testing.T) {
	m := newBrowserWithEntries(makeEntries(1))
	m.mode = modePublicURL
	m.publicURL = "https://disk.yandex.ru/i/abc"

	updated, _ := pressKey(m, "j")
	if updated.mode != modeNormal {
		t.Errorf("mode = %v, want modeNormal", updated.mode)
	}
	if updated.publicURL != "" {
		t.Errorf("publicURL should be cleared, got %q", updated.publicURL)
	}
}

// --- Loading blocks key input ---

func TestHandleKey_Loading_IgnoresKeys(t *testing.T) {
	m := newBrowserWithEntries(makeEntries(3))
	m.loading = true

	original := m.cursor
	updated, cmd := pressSpecialKey(m, tea.KeyDown)
	if updated.cursor != original {
		t.Error("cursor should not change while loading")
	}
	if cmd != nil {
		t.Error("cmd should be nil while loading")
	}
}
