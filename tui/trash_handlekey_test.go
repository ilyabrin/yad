package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ilyabrin/disk"
)

func newTestTrashReady() TrashModel {
	m := newTestTrash()
	m.loading = false
	return m
}

func trashKey(m TrashModel, s string) (TrashModel, tea.Cmd) {
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)})
	return updated, cmd
}

func trashSpecialKey(m TrashModel, t tea.KeyType) (TrashModel, tea.Cmd) {
	updated, cmd := m.Update(tea.KeyMsg{Type: t})
	return updated, cmd
}

// --- Navigation ---

func TestTrashKey_Up_DecrementsCursor(t *testing.T) {
	m := newTestTrashReady()
	m.cursor = 1

	updated, _ := trashSpecialKey(m, tea.KeyUp)
	if updated.cursor != 0 {
		t.Errorf("cursor = %d, want 0", updated.cursor)
	}
}

func TestTrashKey_Up_AtTopDoesNothing(t *testing.T) {
	m := newTestTrashReady()
	m.cursor = 0

	updated, _ := trashSpecialKey(m, tea.KeyUp)
	if updated.cursor != 0 {
		t.Errorf("cursor = %d, want 0", updated.cursor)
	}
}

func TestTrashKey_Down_IncrementsCursor(t *testing.T) {
	m := newTestTrashReady()
	m.cursor = 0

	updated, _ := trashSpecialKey(m, tea.KeyDown)
	if updated.cursor != 1 {
		t.Errorf("cursor = %d, want 1", updated.cursor)
	}
}

func TestTrashKey_Down_AtBottomDoesNothing(t *testing.T) {
	m := newTestTrashReady()
	m.cursor = len(m.items) - 1

	updated, _ := trashSpecialKey(m, tea.KeyDown)
	if updated.cursor != len(m.items)-1 {
		t.Errorf("cursor = %d, want %d", updated.cursor, len(m.items)-1)
	}
}

// --- Back ---

func TestTrashKey_Back_EmitsSwitchToBrowserMsg(t *testing.T) {
	m := newTestTrashReady()

	_, cmd := trashKey(m, "q")
	if cmd == nil {
		t.Fatal("expected switchToBrowserMsg cmd")
	}
	msg := cmd()
	if _, ok := msg.(switchToBrowserMsg); !ok {
		t.Errorf("cmd() = %T, want switchToBrowserMsg", msg)
	}
}

// --- Restore ---

func TestTrashKey_Restore_SendsCmd(t *testing.T) {
	m := newTestTrashReady()
	m.cursor = 0

	_, cmd := trashKey(m, "r")
	if cmd == nil {
		t.Error("expected cmdRestore")
	}
}

func TestTrashKey_Restore_EmptyList_DoesNothing(t *testing.T) {
	m := newTestTrashReady()
	m.items = nil

	_, cmd := trashKey(m, "r")
	if cmd != nil {
		t.Error("expected no cmd on empty list")
	}
}

// --- Delete permanently ---

func TestTrashKey_Delete_ShowsConfirmDialog(t *testing.T) {
	m := newTestTrashReady()

	updated, _ := trashKey(m, "D")
	if updated.mode != trashModeConfirmDelete {
		t.Errorf("mode = %v, want trashModeConfirmDelete", updated.mode)
	}
}

func TestTrashKey_Delete_EmptyList_DoesNothing(t *testing.T) {
	m := newTestTrashReady()
	m.items = nil

	updated, _ := trashKey(m, "D")
	if updated.mode != trashModeNormal {
		t.Errorf("mode = %v, want trashModeNormal on empty list", updated.mode)
	}
}

func TestTrashKey_ConfirmDelete_Y_SendsCmd(t *testing.T) {
	m := newTestTrashReady()
	m.mode = trashModeConfirmDelete
	m.confirm = NewConfirmDialog("Delete", "Sure?")

	_, cmd := trashKey(m, "y")
	if cmd == nil {
		t.Error("expected delete cmd after confirming")
	}
}

func TestTrashKey_ConfirmDelete_N_ReturnsNormal(t *testing.T) {
	m := newTestTrashReady()
	m.mode = trashModeConfirmDelete
	m.confirm = NewConfirmDialog("Delete", "Sure?")

	updated, cmd := trashKey(m, "n")
	if updated.mode != trashModeNormal {
		t.Errorf("mode = %v, want trashModeNormal after cancel", updated.mode)
	}
	if cmd != nil {
		t.Error("expected no cmd after cancel")
	}
}

// --- Empty trash ---

func TestTrashKey_Empty_ShowsConfirmDialog(t *testing.T) {
	m := newTestTrashReady()

	updated, _ := trashKey(m, "E")
	if updated.mode != trashModeConfirmEmpty {
		t.Errorf("mode = %v, want trashModeConfirmEmpty", updated.mode)
	}
}

func TestTrashKey_ConfirmEmpty_Y_SendsCmd(t *testing.T) {
	m := newTestTrashReady()
	m.mode = trashModeConfirmEmpty
	m.confirm = NewConfirmDialog("Empty trash", "Sure?")

	_, cmd := trashKey(m, "y")
	if cmd == nil {
		t.Error("expected empty-trash cmd after confirming")
	}
}

func TestTrashKey_ConfirmEmpty_N_ReturnsNormal(t *testing.T) {
	m := newTestTrashReady()
	m.mode = trashModeConfirmEmpty
	m.confirm = NewConfirmDialog("Empty trash", "Sure?")

	updated, _ := trashKey(m, "n")
	if updated.mode != trashModeNormal {
		t.Errorf("mode = %v, want trashModeNormal after cancel", updated.mode)
	}
}

// --- Refresh ---

func TestTrashKey_Refresh_TriggersReload(t *testing.T) {
	m := newTestTrashReady()

	updated, cmd := trashKey(m, "R")
	if !updated.loading {
		t.Error("loading should be true after refresh")
	}
	if cmd == nil {
		t.Error("expected reload cmd")
	}
}

// --- Message mode ---

func TestTrashKey_MessageMode_ClearedOnAnyKey(t *testing.T) {
	m := newTestTrashReady()
	m.mode = trashModeMessage
	m.message = "done"

	updated, _ := trashKey(m, "j")
	if updated.mode != trashModeNormal {
		t.Errorf("mode = %v, want trashModeNormal", updated.mode)
	}
	if updated.message != "" {
		t.Error("message should be cleared")
	}
}

// --- Error cleared on any key ---

func TestTrashKey_ErrorCleared_OnAnyKey(t *testing.T) {
	m := newTestTrashReady()
	m.err = errTest("network error")

	updated, _ := trashKey(m, "j")
	if updated.err != nil {
		t.Error("err should be cleared on keypress")
	}
}

// --- Loading blocks keys ---

func TestTrashKey_Loading_IgnoresKeys(t *testing.T) {
	m := newTestTrashReady()
	m.loading = true
	orig := m.cursor

	updated, cmd := trashSpecialKey(m, tea.KeyDown)
	if updated.cursor != orig {
		t.Error("cursor should not change while loading")
	}
	if cmd != nil {
		t.Error("cmd should be nil while loading")
	}
}

// --- Ensure TrashResource embeds Resource correctly for test helper ---

var _ = &disk.TrashResource{}
