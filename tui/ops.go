package tui

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/ilyabrin/disk"
)

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
			func(p disk.UploadProgress) { trySend(ch, p) },
		)
		done <- uploadDoneMsg{resource: resource, err: err}
		close(ch)
		close(done)
	}()
	return ch, done
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
			func(p disk.DownloadProgress) { trySend(ch, p) },
		)
		done <- downloadDoneMsg{err: err}
		close(ch)
		close(done)
	}()
	return ch, done
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
		// CreateDirAll rather than CreateDir so that typing a nested path such
		// as "trips/2026/iceland" works: the API only creates one level per
		// request, and the intermediate directories may not exist yet.
		if errResp := client.CreateDirAll(ctx, path); errResp != nil {
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

// cmdUploadFromURL asks Yandex Disk to fetch remoteURL and store it at
// destPath, then waits for that to actually happen.
//
// The API answers 202 Accepted with a link to an operation and only starts
// downloading afterwards, so returning as soon as the request succeeds would
// report success while the file is still on its way and absent from the
// listing we are about to reload.
func cmdUploadFromURL(client *disk.Client, remoteURL, destPath string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeoutTransfer)
		defer cancel()

		link, errResp := client.UploadFile(ctx, destPath, remoteURL)
		if errResp != nil {
			return uploadFromURLDoneMsg{err: newAPIError(errResp)}
		}

		// A 200 carries no operation to poll: the transfer is already done.
		if link == nil || link.Href == "" {
			return uploadFromURLDoneMsg{}
		}

		return uploadFromURLDoneMsg{err: waitForOperation(ctx, client, link.Href)}
	}
}

// waitForOperation polls an asynchronous operation until it leaves the
// in-progress state, the context expires, or the API stops answering.
func waitForOperation(ctx context.Context, client *disk.Client, href string) error {
	ticker := time.NewTicker(operationPollInterval)
	defer ticker.Stop()

	for {
		op, err := client.GetOperationStatus(ctx, href)
		if err != nil {
			return err
		}

		switch op.Status {
		case disk.OperationInProgress:
			// keep waiting
		case operationSuccess:
			return nil
		default:
			return fmt.Errorf("upload failed on the server (status %q)", op.Status)
		}

		select {
		case <-ticker.C:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// --- Open in browser ---

type browserOpenedMsg struct{ err error }

func cmdOpenBrowser(url string) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "windows":
			cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
		case "darwin":
			cmd = exec.Command("open", url)
		default:
			cmd = exec.Command("xdg-open", url)
		}
		return browserOpenedMsg{err: cmd.Start()}
	}
}

// --- helpers ---

// trySend delivers a progress frame without ever blocking the transfer.
// Progress is purely cosmetic: if the UI has not drained the buffer yet the
// frame is dropped rather than throttling the upload/download — and, more
// importantly, the producer goroutine can never deadlock when the model stops
// reading (quit mid-transfer, screen switch).
func trySend[T any](ch chan<- T, v T) {
	select {
	case ch <- v:
	default:
	}
}

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

// isAuthError reports whether err is an authentication failure (HTTP 401/403
// or a token-related error keyword). These errors are handled by attempting a
// silent token refresh before falling back to the fatal error screen.
func isAuthError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "401") || strings.Contains(s, "403") ||
		strings.Contains(s, "Unauthorized") || strings.Contains(s, "unauthorized") ||
		strings.Contains(s, "InvalidToken") || strings.Contains(s, "invalid_token")
}

// authFatalMsg builds a fatalErrorMsg for an authentication failure.
// Used when a token refresh has failed or is not available.
func authFatalMsg(err error) *fatalErrorMsg {
	return &fatalErrorMsg{
		title:  "Authentication Error",
		body:   "Your session has expired or the token is invalid.",
		hint:   "Re-run `yad` to authenticate again.",
		detail: err,
	}
}

// asFatalErrorMsg checks whether err represents an unrecoverable API error and,
// if so, returns the appropriate fatalErrorMsg to show the dedicated screen.
// Auth errors (401/403) are NOT handled here — they are routed through
// tryRefreshMsg first so a silent token refresh can be attempted.
// Returns nil when the error is ordinary and should be shown inline.
func asFatalErrorMsg(err error) *fatalErrorMsg {
	if err == nil {
		return nil
	}
	s := err.Error()

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
