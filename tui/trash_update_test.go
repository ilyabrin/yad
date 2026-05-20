package tui

import (
	"errors"
	"testing"

	"github.com/ilyabrin/disk"
)

func newTestTrash() TrashModel {
	return TrashModel{
		keys: defaultTrashKeyMap(),
		items: []*disk.TrashResource{
			{Resource: disk.Resource{Path: "trash:/a.txt", Name: "a.txt", Type: "file"}},
			{Resource: disk.Resource{Path: "trash:/b.txt", Name: "b.txt", Type: "file"}},
		},
		cursor: 0,
	}
}

// --- trashLoadedMsg ---

func TestTrashUpdate_LoadedMsg_SetsItems(t *testing.T) {
	m := newTestTrash()
	m.loading = true

	items := []*disk.TrashResource{
		{Resource: disk.Resource{Path: "trash:/x.txt", Name: "x.txt"}},
	}
	updated, cmd := m.Update(trashLoadedMsg{items: items})

	if updated.loading {
		t.Error("loading should be false after trashLoadedMsg")
	}
	if len(updated.items) != 1 {
		t.Errorf("items count = %d, want 1", len(updated.items))
	}
	if cmd != nil {
		t.Error("cmd should be nil on success")
	}
}

func TestTrashUpdate_LoadedMsg_Error_ShowsInline(t *testing.T) {
	m := newTestTrash()
	m.loading = true

	updated, cmd := m.Update(trashLoadedMsg{err: errors.New("network error")})

	if updated.loading {
		t.Error("loading should be false")
	}
	if updated.err == nil {
		t.Error("err should be set for non-fatal error")
	}
	if cmd != nil {
		t.Error("cmd should be nil")
	}
}

func TestTrashUpdate_LoadedMsg_AuthError_RoutesFatal(t *testing.T) {
	m := newTestTrash()
	m.loading = true

	_, cmd := m.Update(trashLoadedMsg{err: errors.New("HTTP 401: Unauthorized")})

	if cmd == nil {
		t.Fatal("expected fatalErrorMsg cmd on auth error")
	}
	msg := cmd()
	fe, ok := msg.(fatalErrorMsg)
	if !ok {
		t.Fatalf("expected fatalErrorMsg, got %T", msg)
	}
	if fe.title != "Authentication Error" {
		t.Errorf("title = %q, want Authentication Error", fe.title)
	}
}

func TestTrashUpdate_LoadedMsg_CursorClamped(t *testing.T) {
	m := newTestTrash()
	m.cursor = 10 // beyond list
	m.loading = true

	items := []*disk.TrashResource{
		{Resource: disk.Resource{Path: "trash:/a.txt", Name: "a.txt"}},
		{Resource: disk.Resource{Path: "trash:/b.txt", Name: "b.txt"}},
	}
	updated, _ := m.Update(trashLoadedMsg{items: items})

	if updated.cursor != 1 {
		t.Errorf("cursor = %d, want 1 (clamped to last)", updated.cursor)
	}
}

// --- trashRestoreDoneMsg ---

func TestTrashUpdate_RestoreDoneMsg_Success_Reloads(t *testing.T) {
	m := newTestTrash()

	updated, cmd := m.Update(trashRestoreDoneMsg{})

	if !updated.loading {
		t.Error("loading should be true after successful restore")
	}
	if cmd == nil {
		t.Error("expected reload cmd")
	}
}

func TestTrashUpdate_RestoreDoneMsg_Error_ShowsMessage(t *testing.T) {
	m := newTestTrash()

	updated, cmd := m.Update(trashRestoreDoneMsg{err: errors.New("restore failed")})

	if updated.mode != trashModeMessage {
		t.Errorf("mode = %v, want trashModeMessage", updated.mode)
	}
	if !updated.messageIsError {
		t.Error("messageIsError should be true")
	}
	if cmd != nil {
		t.Error("cmd should be nil on error")
	}
}

// --- trashDeleteDoneMsg ---

func TestTrashUpdate_DeleteDoneMsg_Success_Reloads(t *testing.T) {
	m := newTestTrash()

	updated, cmd := m.Update(trashDeleteDoneMsg{})

	if !updated.loading {
		t.Error("loading should be true after successful delete")
	}
	if cmd == nil {
		t.Error("expected reload cmd")
	}
}

func TestTrashUpdate_DeleteDoneMsg_Error_ShowsMessage(t *testing.T) {
	m := newTestTrash()

	updated, _ := m.Update(trashDeleteDoneMsg{err: errors.New("delete failed")})

	if updated.mode != trashModeMessage {
		t.Errorf("mode = %v, want trashModeMessage", updated.mode)
	}
	if !updated.messageIsError {
		t.Error("messageIsError should be true")
	}
}

// --- trashEmptyDoneMsg ---

func TestTrashUpdate_EmptyDoneMsg_Success_ClearsItems(t *testing.T) {
	m := newTestTrash()

	updated, _ := m.Update(trashEmptyDoneMsg{})

	if len(updated.items) != 0 {
		t.Errorf("items should be empty after emptying trash, got %d", len(updated.items))
	}
	if updated.cursor != 0 {
		t.Errorf("cursor = %d, want 0", updated.cursor)
	}
	if updated.mode != trashModeMessage {
		t.Errorf("mode = %v, want trashModeMessage (success message)", updated.mode)
	}
	if updated.messageIsError {
		t.Error("messageIsError should be false on success")
	}
}

func TestTrashUpdate_EmptyDoneMsg_Error_ShowsMessage(t *testing.T) {
	m := newTestTrash()

	updated, _ := m.Update(trashEmptyDoneMsg{err: errors.New("empty failed")})

	if updated.mode != trashModeMessage {
		t.Errorf("mode = %v, want trashModeMessage", updated.mode)
	}
	if !updated.messageIsError {
		t.Error("messageIsError should be true on error")
	}
}
