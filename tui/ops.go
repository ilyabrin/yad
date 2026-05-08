package tui

import (
	"context"
	"fmt"
	"time"

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

// cmdUpload starts an upload in a goroutine and returns the first progress tick.
// Subsequent progress is delivered via cmdWaitUpload.
func cmdUpload(client *disk.Client, localPath, remotePath string) tea.Cmd {
	ch := make(chan disk.UploadProgress, 32)
	done := make(chan uploadDoneMsg, 1)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		resource, err := client.UploadFileFromPathWithProgress(ctx, localPath, remotePath, true,
			func(p disk.UploadProgress) { ch <- p },
		)
		done <- uploadDoneMsg{resource: resource, err: err}
		close(ch)
		close(done)
	}()

	return cmdWaitUpload(ch, done)
}

// cmdWaitUpload reads the next progress event or the done signal.
func cmdWaitUpload(ch <-chan disk.UploadProgress, done <-chan uploadDoneMsg) tea.Cmd {
	return func() tea.Msg {
		select {
		case p, ok := <-ch:
			if !ok {
				// channel closed before done - drain done
				if d := <-done; true {
					return d
				}
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

func cmdDownload(client *disk.Client, remotePath, localPath string) tea.Cmd {
	ch := make(chan disk.DownloadProgress, 32)
	done := make(chan downloadDoneMsg, 1)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		err := client.DownloadFileToPathWithProgress(ctx, remotePath, localPath, true,
			func(p disk.DownloadProgress) { ch <- p },
		)
		done <- downloadDoneMsg{err: err}
		close(ch)
		close(done)
	}()

	return cmdWaitDownload(ch, done)
}

func cmdWaitDownload(ch <-chan disk.DownloadProgress, done <-chan downloadDoneMsg) tea.Cmd {
	return func() tea.Msg {
		select {
		case p, ok := <-ch:
			if !ok {
				if d := <-done; true {
					return d
				}
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
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		err := client.DeleteResource(ctx, path, false)
		return deleteDoneMsg{err: err}
	}
}

// --- New directory ---

type mkdirDoneMsg struct{ err error }

func cmdMkdir(client *disk.Client, path string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
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
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
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
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
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
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
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
