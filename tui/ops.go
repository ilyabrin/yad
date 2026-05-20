package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/ilyabrin/disk"
)

// --- Operation result messages ---

type opSuccessMsg struct{ info string }
type opErrMsg struct{ err error }

// --- Upload ---

type uploadProgressMsg disk.UploadProgress
type uploadDoneMsg struct {
	resource *disk.Resource
	err      error
}

// startUploadAsync spawns the upload goroutine and returns the progress channels.
// Use cmdWaitUpload to receive from them.
func startUploadAsync(client *disk.Client, localPath, remotePath string) (<-chan disk.UploadProgress, <-chan uploadDoneMsg) {
	ch := make(chan disk.UploadProgress, 32)
	done := make(chan uploadDoneMsg, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), timeoutTransfer)
		defer cancel()
		resource, err := client.UploadFileFromPathWithProgress(ctx, localPath, remotePath, true,
			func(p disk.UploadProgress) { ch <- p },
		)
		done <- uploadDoneMsg{resource: resource, err: err}
		close(ch)
		close(done)
	}()
	return ch, done
}

// cmdUpload starts an upload and returns the first progress tick.
// Subsequent progress is delivered via cmdWaitUpload.
func cmdUpload(client *disk.Client, localPath, remotePath string) tea.Cmd {
	ch, done := startUploadAsync(client, localPath, remotePath)
	return cmdWaitUpload(ch, done)
}

// cmdWaitUpload reads the next progress event or the done signal.
func cmdWaitUpload(ch <-chan disk.UploadProgress, done <-chan uploadDoneMsg) tea.Cmd {
	return func() tea.Msg {
		select {
		case p, ok := <-ch:
			if !ok {
				return <-done
			}
			return uploadProgressMsg(p)
		case d := <-done:
			return d
		}
	}
}

// --- Download ---

type downloadProgressMsg disk.DownloadProgress
type downloadDoneMsg struct{ err error }

// downloadStartedMsg carries the channels of a newly started download.
// It is used to store the channels in the model after a lazy goroutine start.
type downloadStartedMsg struct {
	ch       <-chan disk.DownloadProgress
	done     <-chan downloadDoneMsg
	filename string
}

// uploadStartedMsg carries the channels of a newly started upload.
type uploadStartedMsg struct {
	ch   <-chan disk.UploadProgress
	done <-chan uploadDoneMsg
}

// startDownloadAsync spawns the download goroutine and returns the progress channels.
// Use cmdWaitDownload to receive from them.
func startDownloadAsync(client *disk.Client, remotePath, localPath string) (<-chan disk.DownloadProgress, <-chan downloadDoneMsg) {
	ch := make(chan disk.DownloadProgress, 32)
	done := make(chan downloadDoneMsg, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), timeoutTransfer)
		defer cancel()
		err := client.DownloadFileToPathWithProgress(ctx, remotePath, localPath, true,
			func(p disk.DownloadProgress) { ch <- p },
		)
		done <- downloadDoneMsg{err: err}
		close(ch)
		close(done)
	}()
	return ch, done
}

func cmdDownload(client *disk.Client, remotePath, localPath string) tea.Cmd {
	ch, done := startDownloadAsync(client, remotePath, localPath)
	return cmdWaitDownload(ch, done)
}

// cmdStartDownload lazily starts a download goroutine inside a tea.Cmd,
// returning a downloadStartedMsg with the channels so the model can store them.
func cmdStartDownload(client *disk.Client, remotePath, localPath, filename string) tea.Cmd {
	return func() tea.Msg {
		ch, done := startDownloadAsync(client, remotePath, localPath)
		return downloadStartedMsg{ch: ch, done: done, filename: filename}
	}
}

// cmdStartUpload lazily starts an upload goroutine inside a tea.Cmd.
func cmdStartUpload(client *disk.Client, localPath, remotePath string) tea.Cmd {
	return func() tea.Msg {
		ch, done := startUploadAsync(client, localPath, remotePath)
		return uploadStartedMsg{ch: ch, done: done}
	}
}

func cmdWaitDownload(ch <-chan disk.DownloadProgress, done <-chan downloadDoneMsg) tea.Cmd {
	return func() tea.Msg {
		select {
		case p, ok := <-ch:
			if !ok {
				return <-done
			}
			return downloadProgressMsg(p)
		case d := <-done:
			return d
		}
	}
}

// --- Delete ---

type deleteDoneMsg struct{ err error }

func cmdDelete(client *disk.Client, path string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeoutDelete)
		defer cancel()
		err := client.DeleteResource(ctx, path, false)
		return deleteDoneMsg{err: err}
	}
}

// --- New directory ---

type mkdirDoneMsg struct{ err error }

func cmdMkdir(client *disk.Client, path string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeoutOp)
		defer cancel()
		_, errResp := client.CreateDir(ctx, path)
		if errResp != nil {
			return mkdirDoneMsg{err: newAPIError(errResp)}
		}
		return mkdirDoneMsg{}
	}
}

// --- Rename (move) ---

type renameDoneMsg struct{ err error }

func cmdRename(client *disk.Client, from, to string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeoutOp)
		defer cancel()
		_, errResp := client.MoveResource(ctx, from, to)
		if errResp != nil {
			return renameDoneMsg{err: newAPIError(errResp)}
		}
		return renameDoneMsg{}
	}
}

// --- Publish / Unpublish ---

type publishDoneMsg struct {
	publicURL string // non-empty means resource is now public
	err       error
}

type unpublishDoneMsg struct{ err error }

func cmdPublish(client *disk.Client, path string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeoutOp)
		defer cancel()

		_, errResp := client.PublishResource(ctx, path)
		if errResp != nil {
			return publishDoneMsg{err: newAPIError(errResp)}
		}

		// Fetch updated metadata to get the public URL
		resource, errResp := client.GetMetadata(ctx, path)
		if errResp != nil {
			// Published OK but couldn't fetch URL - not fatal
			return publishDoneMsg{publicURL: ""}
		}
		return publishDoneMsg{publicURL: resource.PublicURL}
	}
}

func cmdUnpublish(client *disk.Client, path string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeoutOp)
		defer cancel()
		_, errResp := client.UnpublishResource(ctx, path)
		if errResp != nil {
			return unpublishDoneMsg{err: newAPIError(errResp)}
		}
		return unpublishDoneMsg{}
	}
}

// --- Clipboard ---

type clipboardDoneMsg struct{ err error }

func cmdCopyToClipboard(text string) tea.Cmd {
	return func() tea.Msg {
		err := clipboard.WriteAll(text)
		return clipboardDoneMsg{err: err}
	}
}

// --- Upload from URL ---

type uploadFromURLDoneMsg struct{ err error }

func cmdUploadFromURL(client *disk.Client, remoteURL, destDir string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeoutOp)
		defer cancel()
		_, errResp := client.UploadFile(ctx, destDir, remoteURL)
		if errResp != nil {
			return uploadFromURLDoneMsg{err: newAPIError(errResp)}
		}
		return uploadFromURLDoneMsg{}
	}
}

// --- helpers ---

func newAPIError(e *disk.ErrorResponse) error {
	if e == nil {
		return nil
	}
	msg := e.Message
	if msg == "" {
		msg = e.Error
	}
	if e.Description != "" {
		msg += ": " + e.Description
	}
	return fmt.Errorf("%s", msg)
}

// asFatalErrorMsg checks whether err represents an unrecoverable API error and,
// if so, returns the appropriate fatalErrorMsg to show the dedicated screen.
// Returns nil when the error is ordinary and should be shown inline.
func asFatalErrorMsg(err error) *fatalErrorMsg {
	if err == nil {
		return nil
	}
	s := err.Error()

	// Expired or invalid token (HTTP 401 / 403)
	if strings.Contains(s, "401") || strings.Contains(s, "403") ||
		strings.Contains(s, "Unauthorized") || strings.Contains(s, "unauthorized") ||
		strings.Contains(s, "InvalidToken") || strings.Contains(s, "invalid_token") {
		return &fatalErrorMsg{
			title:  "Authentication Error",
			body:   "Your session has expired or the token is invalid.",
			hint:   "Re-run `yad` to authenticate again.",
			detail: err,
		}
	}

	// Storage overdraft — API disabled until quota is restored
	if strings.Contains(s, "DiskAPIDisabledForOverdraftUserError") ||
		strings.Contains(s, "OverDraft") || strings.Contains(s, "overdraft") {
		return &fatalErrorMsg{
			title:  "Storage Overdraft",
			body:   "API access is disabled: your files exceed your available storage.",
			hint:   "Free up space or upgrade your Yandex Disk plan, then restart `yad`.",
			detail: err,
		}
	}

	return nil
}
