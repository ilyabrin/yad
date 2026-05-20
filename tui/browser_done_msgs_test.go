package tui

import (
	"errors"
	"testing"

	"github.com/ilyabrin/disk"
)

// --- mkdirDoneMsg ---

func TestMkdirDoneMsg_Success_Reloads(t *testing.T) {
	m := newTestBrowser()
	m.mode = modeInputNewDir

	updated, cmd := m.Update(mkdirDoneMsg{})

	if updated.mode != modeNormal {
		t.Errorf("mode = %v, want modeNormal", updated.mode)
	}
	if !updated.loading {
		t.Error("loading should be true after mkdir")
	}
	if cmd == nil {
		t.Error("expected reload cmd")
	}
}

func TestMkdirDoneMsg_Error_ShowsMessage(t *testing.T) {
	m := newTestBrowser()

	updated, cmd := m.Update(mkdirDoneMsg{err: errors.New("already exists")})

	if updated.mode != modeMessage {
		t.Errorf("mode = %v, want modeMessage", updated.mode)
	}
	if !updated.messageIsError {
		t.Error("messageIsError should be true")
	}
	if cmd != nil {
		t.Error("cmd should be nil on error")
	}
}

// --- renameDoneMsg ---

func TestRenameDoneMsg_Success_Reloads(t *testing.T) {
	m := newTestBrowser()
	m.mode = modeInputRename

	updated, cmd := m.Update(renameDoneMsg{})

	if updated.mode != modeNormal {
		t.Errorf("mode = %v, want modeNormal", updated.mode)
	}
	if !updated.loading {
		t.Error("loading should be true after rename")
	}
	if cmd == nil {
		t.Error("expected reload cmd")
	}
}

func TestRenameDoneMsg_Error_ShowsMessage(t *testing.T) {
	m := newTestBrowser()

	updated, cmd := m.Update(renameDoneMsg{err: errors.New("permission denied")})

	if updated.mode != modeMessage {
		t.Errorf("mode = %v, want modeMessage", updated.mode)
	}
	if !updated.messageIsError {
		t.Error("messageIsError should be true")
	}
	if cmd != nil {
		t.Error("cmd should be nil on error")
	}
}

// --- publishDoneMsg ---

func TestPublishDoneMsg_Success_ShowsURL(t *testing.T) {
	m := newTestBrowser()

	updated, cmd := m.Update(publishDoneMsg{publicURL: "https://disk.yandex.ru/i/abc"})

	if updated.mode != modePublicURL {
		t.Errorf("mode = %v, want modePublicURL", updated.mode)
	}
	if updated.publicURL != "https://disk.yandex.ru/i/abc" {
		t.Errorf("publicURL = %q, want https://disk.yandex.ru/i/abc", updated.publicURL)
	}
	if !updated.loading {
		t.Error("loading should be true (reload starts alongside showing the URL)")
	}
	if cmd == nil {
		t.Error("expected reload cmd")
	}
}

func TestPublishDoneMsg_Error_ShowsMessage(t *testing.T) {
	m := newTestBrowser()

	updated, cmd := m.Update(publishDoneMsg{err: errors.New("publish failed")})

	if updated.mode != modeMessage {
		t.Errorf("mode = %v, want modeMessage", updated.mode)
	}
	if !updated.messageIsError {
		t.Error("messageIsError should be true")
	}
	if cmd != nil {
		t.Error("cmd should be nil on error")
	}
}

// --- unpublishDoneMsg ---

func TestUnpublishDoneMsg_Success_Reloads(t *testing.T) {
	m := newTestBrowser()
	m.mode = modePublicURL

	updated, cmd := m.Update(unpublishDoneMsg{})

	if updated.mode != modeNormal {
		t.Errorf("mode = %v, want modeNormal", updated.mode)
	}
	if !updated.loading {
		t.Error("loading should be true after unpublish")
	}
	if cmd == nil {
		t.Error("expected reload cmd")
	}
}

func TestUnpublishDoneMsg_Error_ShowsMessage(t *testing.T) {
	m := newTestBrowser()

	updated, _ := m.Update(unpublishDoneMsg{err: errors.New("unpublish failed")})

	if updated.mode != modeMessage {
		t.Errorf("mode = %v, want modeMessage", updated.mode)
	}
	if !updated.messageIsError {
		t.Error("messageIsError should be true")
	}
}

// --- uploadFromURLDoneMsg ---

func TestUploadFromURLDoneMsg_Success_Reloads(t *testing.T) {
	m := newTestBrowser()

	updated, cmd := m.Update(uploadFromURLDoneMsg{})

	if !updated.loading {
		t.Error("loading should be true after URL upload")
	}
	if cmd == nil {
		t.Error("expected reload cmd")
	}
}

func TestUploadFromURLDoneMsg_Error_ShowsMessage(t *testing.T) {
	m := newTestBrowser()

	updated, _ := m.Update(uploadFromURLDoneMsg{err: errors.New("invalid URL")})

	if updated.mode != modeMessage {
		t.Errorf("mode = %v, want modeMessage", updated.mode)
	}
	if !updated.messageIsError {
		t.Error("messageIsError should be true")
	}
}

// --- clipboardDoneMsg ---

func TestClipboardDoneMsg_Success_ShowsSuccessMessage(t *testing.T) {
	m := newTestBrowser()

	updated, _ := m.Update(clipboardDoneMsg{})

	if updated.mode != modeMessage {
		t.Errorf("mode = %v, want modeMessage", updated.mode)
	}
	if updated.messageIsError {
		t.Error("messageIsError should be false on success")
	}
}

func TestClipboardDoneMsg_Error_ShowsErrorMessage(t *testing.T) {
	m := newTestBrowser()

	updated, _ := m.Update(clipboardDoneMsg{err: errors.New("clipboard unavailable")})

	if updated.mode != modeMessage {
		t.Errorf("mode = %v, want modeMessage", updated.mode)
	}
	if !updated.messageIsError {
		t.Error("messageIsError should be true")
	}
}

// --- uploadDoneMsg ---

func TestUploadDoneMsg_Success_Reloads(t *testing.T) {
	m := newTestBrowser()
	m.mode = modeUpload

	updated, cmd := m.Update(uploadDoneMsg{})

	if !updated.progress.Done {
		t.Error("progress.Done should be true")
	}
	if updated.mode != modeNormal {
		t.Errorf("mode = %v, want modeNormal", updated.mode)
	}
	if !updated.loading {
		t.Error("loading should be true")
	}
	if cmd == nil {
		t.Error("expected reload cmd")
	}
}

func TestUploadDoneMsg_Error_SetsProgressErr(t *testing.T) {
	m := newTestBrowser()
	m.mode = modeUpload

	updated, cmd := m.Update(uploadDoneMsg{err: errors.New("disk full")})

	if !updated.progress.Done {
		t.Error("progress.Done should be true even on error")
	}
	if updated.progress.Err == nil {
		t.Error("progress.Err should be set")
	}
	if cmd != nil {
		t.Error("cmd should be nil on upload error")
	}
}

// --- uploadStartedMsg / uploadProgressMsg ---

func TestUploadStartedMsg_StoresChannels(t *testing.T) {
	m := newTestBrowser()
	ch := make(chan uploadDoneMsg, 1)
	ch <- uploadDoneMsg{} // pre-fill so cmdWaitUpload doesn't block

	msg := uploadStartedMsg{
		ch:   make(chan disk.UploadProgress),
		done: ch,
	}
	updated, cmd := m.Update(msg)

	if updated.uploadDone == nil {
		t.Error("uploadDone channel should be stored in model")
	}
	if cmd == nil {
		t.Error("expected cmdWaitUpload to be returned")
	}
}

// --- downloadStartedMsg ---

func TestDownloadStartedMsg_StoresChannels(t *testing.T) {
	m := newTestBrowser()
	ch := make(chan downloadDoneMsg, 1)
	ch <- downloadDoneMsg{} // pre-fill so cmdWaitDownload doesn't block

	msg := downloadStartedMsg{
		ch:   make(chan disk.DownloadProgress),
		done: ch,
	}
	updated, cmd := m.Update(msg)

	if updated.dlDone == nil {
		t.Error("dlDone channel should be stored in model")
	}
	if cmd == nil {
		t.Error("expected cmdWaitDownload to be returned")
	}
}
