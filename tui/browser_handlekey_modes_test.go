package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// --- modeUpload / modeDownload: key only matters when progress.Done ---

func TestHandleKey_UploadMode_DoneProgress_ReturnsNormal(t *testing.T) {
	m := newBrowserWithEntries(makeEntries(1))
	m.mode = modeUpload
	m.progress.Done = true

	updated, _ := pressKey(m, "j")
	if updated.mode != modeNormal {
		t.Errorf("mode = %v, want modeNormal", updated.mode)
	}
	if updated.progress.Done {
		t.Error("progress.Done should be reset")
	}
}

func TestHandleKey_DownloadMode_DoneProgress_ReturnsNormal(t *testing.T) {
	m := newBrowserWithEntries(makeEntries(1))
	m.mode = modeDownload
	m.progress.Done = true

	updated, _ := pressKey(m, "j")
	if updated.mode != modeNormal {
		t.Errorf("mode = %v, want modeNormal", updated.mode)
	}
}

func TestHandleKey_UploadMode_NotDone_DoesNothing(t *testing.T) {
	m := newBrowserWithEntries(makeEntries(1))
	m.mode = modeUpload
	m.progress.Done = false

	updated, cmd := pressKey(m, "j")
	if updated.mode != modeUpload {
		t.Errorf("mode = %v, want modeUpload (transfer in progress)", updated.mode)
	}
	if cmd != nil {
		t.Error("expected no cmd while transfer is in progress")
	}
}

// --- modeMetadata ---

func TestHandleKey_MetadataMode_AnyKey_Dismisses(t *testing.T) {
	m := newBrowserWithEntries(makeEntries(1))
	m.mode = modeMetadata

	updated, _ := pressKey(m, "j")
	if updated.mode != modeNormal {
		t.Errorf("mode = %v, want modeNormal", updated.mode)
	}
}

// --- modeInputUpload ---

func TestHandleKey_InputUpload_Cancel(t *testing.T) {
	m := newBrowserWithEntries(makeEntries(1))
	m.mode = modeInputUpload
	m.inputDlg = NewInputDialog("Upload", "", "")

	updated, _ := cancelInput(m)
	if updated.mode != modeNormal {
		t.Errorf("mode = %v, want modeNormal", updated.mode)
	}
}

func TestHandleKey_InputUpload_EmptyPath_DoesNothing(t *testing.T) {
	m := newBrowserWithEntries(makeEntries(1))
	m.mode = modeInputUpload
	m.inputDlg = NewInputDialog("Upload", "", "")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if updated.mode != modeNormal {
		t.Errorf("mode = %v, want modeNormal", updated.mode)
	}
	if cmd != nil {
		t.Error("expected no cmd for empty path")
	}
}

func TestHandleKey_InputUpload_ValidPath_AdvancesToNameStep(t *testing.T) {
	m := newBrowserWithEntries(makeEntries(1))
	m.mode = modeInputUpload
	m.inputDlg = NewInputDialog("Upload", "", "")

	updated, _ := submitInput(m, "/home/user/photo.jpg")
	if updated.mode != modeInputUploadName {
		t.Errorf("mode = %v, want modeInputUploadName", updated.mode)
	}
	if updated.pendingUploadSrc != "/home/user/photo.jpg" {
		t.Errorf("pendingUploadSrc = %q, want /home/user/photo.jpg", updated.pendingUploadSrc)
	}
	if updated.pendingUploadIsURL {
		t.Error("pendingUploadIsURL should be false for local path")
	}
}

// --- modeInputUploadURL ---

func TestHandleKey_InputUploadURL_Cancel(t *testing.T) {
	m := newBrowserWithEntries(makeEntries(1))
	m.mode = modeInputUploadURL
	m.inputDlg = NewInputDialog("Upload URL", "", "")

	updated, _ := cancelInput(m)
	if updated.mode != modeNormal {
		t.Errorf("mode = %v, want modeNormal", updated.mode)
	}
}

func TestHandleKey_InputUploadURL_ValidURL_AdvancesToNameStep(t *testing.T) {
	m := newBrowserWithEntries(makeEntries(1))
	m.mode = modeInputUploadURL
	m.inputDlg = NewInputDialog("Upload URL", "", "")

	updated, _ := submitInput(m, "https://example.com/file.zip")
	if updated.mode != modeInputUploadName {
		t.Errorf("mode = %v, want modeInputUploadName", updated.mode)
	}
	if !updated.pendingUploadIsURL {
		t.Error("pendingUploadIsURL should be true for URL")
	}
}

// --- modeInputUploadName ---

func TestHandleKey_InputUploadName_Cancel_ClearsPending(t *testing.T) {
	m := newBrowserWithEntries(makeEntries(1))
	m.mode = modeInputUploadName
	m.pendingUploadSrc = "/some/file.jpg"
	m.inputDlg = NewInputDialog("Name", "", "")

	updated, _ := cancelInput(m)
	if updated.mode != modeNormal {
		t.Errorf("mode = %v, want modeNormal", updated.mode)
	}
	if updated.pendingUploadSrc != "" {
		t.Errorf("pendingUploadSrc should be cleared, got %q", updated.pendingUploadSrc)
	}
}

func TestHandleKey_InputUploadName_LocalFile_StartsUpload(t *testing.T) {
	m := newBrowserWithEntries(makeEntries(1))
	m.mode = modeInputUploadName
	m.pendingUploadSrc = "/home/user/photo.jpg"
	m.pendingUploadIsURL = false
	m.inputDlg = NewInputDialog("Name", "", "")

	updated, cmd := submitInput(m, "my-photo.jpg")
	if updated.mode != modeUpload {
		t.Errorf("mode = %v, want modeUpload", updated.mode)
	}
	if cmd == nil {
		t.Error("expected cmdStartUpload")
	}
}

func TestHandleKey_InputUploadName_URL_StartsURLUpload(t *testing.T) {
	m := newBrowserWithEntries(makeEntries(1))
	m.mode = modeInputUploadName
	m.pendingUploadSrc = "https://example.com/file.zip"
	m.pendingUploadIsURL = true
	m.inputDlg = NewInputDialog("Name", "", "")

	_, cmd := submitInput(m, "file.zip")
	if cmd == nil {
		t.Error("expected cmdUploadFromURL")
	}
}

// --- modeInputDownload ---

func TestHandleKey_InputDownload_Cancel(t *testing.T) {
	m := newBrowserWithEntries(makeEntries(1))
	m.mode = modeInputDownload
	m.inputDlg = NewInputDialog("Download", "", "")

	updated, _ := cancelInput(m)
	if updated.mode != modeNormal {
		t.Errorf("mode = %v, want modeNormal", updated.mode)
	}
}

func TestHandleKey_InputDownload_ValidPath_StartsDownload(t *testing.T) {
	file := makeEntries(1)
	m := newBrowserWithEntries(file)
	m.mode = modeInputDownload
	m.inputDlg = NewInputDialog("Download", "", "")

	updated, cmd := submitInput(m, "/tmp/photo.jpg")
	if updated.mode != modeDownload {
		t.Errorf("mode = %v, want modeDownload", updated.mode)
	}
	if cmd == nil {
		t.Error("expected cmdStartDownload")
	}
}

// --- modeInputDownloadDir ---

func TestHandleKey_InputDownloadDir_Cancel(t *testing.T) {
	m := newBrowserWithEntries(makeEntries(2))
	m.mode = modeInputDownloadDir
	m.inputDlg = NewInputDialog("Download dir", "", "")

	updated, _ := cancelInput(m)
	if updated.mode != modeNormal {
		t.Errorf("mode = %v, want modeNormal", updated.mode)
	}
}

func TestHandleKey_InputDownloadDir_ValidDir_StartsDownload(t *testing.T) {
	m := newBrowserWithEntries(makeEntries(2))
	m.selected = map[string]bool{
		m.entries[0].resource.Path: true,
		m.entries[1].resource.Path: true,
	}
	m.mode = modeInputDownloadDir
	m.inputDlg = NewInputDialog("Download dir", "", "")

	updated, cmd := submitInput(m, "/tmp/downloads")
	if updated.mode != modeDownload {
		t.Errorf("mode = %v, want modeDownload", updated.mode)
	}
	if cmd == nil {
		t.Error("expected cmdStartDownload")
	}
	// Second file goes into pending queue
	if len(updated.pendingDownloads) != 1 {
		t.Errorf("pendingDownloads = %d, want 1", len(updated.pendingDownloads))
	}
}
