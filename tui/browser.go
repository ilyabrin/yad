package tui

import (
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ilyabrin/disk"
)

const pageSize = 100

// browserMode is the current overlay / interaction state.
type browserMode int

const (
	modeNormal             browserMode = iota
	modeConfirmDelete                  // delete confirmation dialog
	modeInputNewDir                    // new directory name input
	modeInputRename                    // rename input
	modeInputUpload                    // local file path input before upload
	modeInputDownload                  // local destination path input before download
	modeUpload                         // upload in progress
	modeDownload                       // download in progress
	modeMessage                        // transient success / error message
	modePublicURL                      // showing public URL after publish
	modeMetadata                       // file / directory metadata overlay
	modeConfirmBulkDelete              // confirm deletion of selected items
	modeInputDownloadDir               // destination directory for bulk download
	modeInputUploadURL                 // remote URL to upload from
	modeInputUploadName                // confirm/change filename before local upload
	modeInputUploadURLName             // confirm/change filename before URL upload
)

// entry is a single row in the file list.
type entry struct {
	resource *disk.Resource
}

func (e entry) isDir() bool { return e.resource.Type == "dir" }

func (e entry) icon() string {
	if e.isDir() {
		return "  "
	}
	return "  "
}

func (e entry) sizeStr() string {
	if e.isDir() {
		return "-"
	}
	return disk.FormatFileSize(int64(e.resource.Size))
}

func (e entry) modifiedStr() string {
	if e.resource.Modified == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, e.resource.Modified)
	if err != nil {
		return e.resource.Modified[:10]
	}
	return t.Format("2006-01-02 15:04")
}

// --- Bubbletea messages ---

type loadedMsg struct {
	path    string
	entries []entry
	total   int
	err     error
}

// --- BrowserModel ---

type BrowserModel struct {
	client  *disk.Client
	keys    BrowserKeyMap
	spinner spinner.Model

	// Directory state
	path    string
	entries []entry
	cursor  int
	total   int
	offset  int
	sort    string // API sort field: "name", "-name", "modified", "-modified", "size", "-size"

	// Terminal size
	width  int
	height int

	// Navigation state
	loading         bool
	cursorAfterLoad int   // -1 = first item, ≥0 = explicit position (set before page load)
	err             error // last load error (cleared on key press)

	// Overlay state
	mode           browserMode
	confirm        ConfirmDialog
	inputDlg       InputDialog
	progress       ProgressOverlay
	message        string // shown in modeMessage
	messageIsError bool
	publicURL      string // shown in modePublicURL

	// Upload staging
	pendingUploadSrc string // local path or URL held between step 1 and step 2

	// Multi-selection
	selected         map[string]bool // resource path → selected
	pendingDelete    []string        // paths queued for sequential bulk delete
	pendingDownloads []string        // remote paths queued for sequential bulk download
	downloadDir      string          // local destination dir for bulk download

	// Active async channels (nil when idle)
	uploadCh   <-chan disk.UploadProgress
	uploadDone <-chan uploadDoneMsg
	dlCh       <-chan disk.DownloadProgress
	dlDone     <-chan downloadDoneMsg
}

// sortCycle defines the order in which sort modes cycle on each keypress.
var sortCycle = []string{"name", "-name", "modified", "-modified", "size", "-size"}

// sortLabel returns a short human-readable label for the current sort.
func sortLabel(s string) string {
	switch s {
	case "name":
		return "name ↑"
	case "-name":
		return "name ↓"
	case "modified":
		return "date ↑"
	case "-modified":
		return "date ↓"
	case "size":
		return "size ↑"
	case "-size":
		return "size ↓"
	default:
		return s
	}
}

// nextSort returns the next sort mode in the cycle.
func nextSort(current string) string {
	for i, s := range sortCycle {
		if s == current {
			return sortCycle[(i+1)%len(sortCycle)]
		}
	}
	return sortCycle[0]
}

func NewBrowserModel(client *disk.Client) BrowserModel {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(colorPrimary)

	return BrowserModel{
		client:  client,
		keys:    DefaultBrowserKeyMap(),
		spinner: sp,
		path:    "/",
		sort:    "name",
		loading: true,
	}
}

func (m BrowserModel) Init() tea.Cmd {
	return tea.Batch(m.loadDir(m.path, 0), m.spinner.Tick)
}

// loadDir returns a Cmd that fetches directory contents.
func (m BrowserModel) loadDir(path string, offset int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		opts := &disk.ResourceOptions{
			Limit:  pageSize,
			Offset: offset,
			Sort:   m.sort,
		}
		resource, errResp := m.client.GetMetadataWithOptions(ctx, path, opts)
		if errResp != nil {
			return loadedMsg{path: path, err: fmt.Errorf("%s: %s", errResp.Error, errResp.Message)}
		}

		var entries []entry
		total := 0
		if resource.Embedded != nil {
			total = resource.Embedded.Total
			for _, r := range resource.Embedded.Items {
				entries = append(entries, entry{resource: r})
			}
		}
		return loadedMsg{path: path, entries: entries, total: total}
	}
}

// ---- Update ----------------------------------------------------------------

func (m BrowserModel) Update(msg tea.Msg) (BrowserModel, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	// --- Directory listing loaded ---
	case loadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.path = msg.path
		m.entries = msg.entries
		m.total = msg.total
		if m.cursorAfterLoad < 0 {
			m.cursor = max(0, len(m.entries)-1)
		} else if m.cursorAfterLoad < len(m.entries) {
			m.cursor = m.cursorAfterLoad
		} else {
			m.cursor = 0
		}
		m.cursorAfterLoad = 0

	// --- Operation results ---
	case deleteDoneMsg:
		if msg.err != nil {
			m.pendingDelete = nil
			return m.showMessage("✗ "+msg.err.Error(), true), nil
		}
		if len(m.pendingDelete) > 0 {
			next := m.pendingDelete[0]
			m.pendingDelete = m.pendingDelete[1:]
			return m, cmdDelete(m.client, next)
		}
		m.mode = modeNormal
		m.selected = nil
		m.loading = true
		return m, tea.Batch(m.loadDir(m.path, 0), m.spinner.Tick)

	case mkdirDoneMsg:
		if msg.err != nil {
			return m.showMessage("✗ "+msg.err.Error(), true), nil
		}
		m.mode = modeNormal
		m.loading = true
		return m, tea.Batch(m.loadDir(m.path, 0), m.spinner.Tick)

	case renameDoneMsg:
		if msg.err != nil {
			return m.showMessage("✗ "+msg.err.Error(), true), nil
		}
		m.mode = modeNormal
		m.loading = true
		return m, tea.Batch(m.loadDir(m.path, 0), m.spinner.Tick)

	case publishDoneMsg:
		if msg.err != nil {
			return m.showMessage("✗ "+msg.err.Error(), true), nil
		}
		m.mode = modePublicURL
		m.publicURL = msg.publicURL
		m.loading = true
		return m, tea.Batch(m.loadDir(m.path, 0), m.spinner.Tick)

	case unpublishDoneMsg:
		if msg.err != nil {
			return m.showMessage("✗ "+msg.err.Error(), true), nil
		}
		m.loading = true
		m.mode = modeNormal
		return m, tea.Batch(m.loadDir(m.path, 0), m.spinner.Tick)

	case uploadFromURLDoneMsg:
		if msg.err != nil {
			return m.showMessage("✗ "+msg.err.Error(), true), nil
		}
		m.loading = true
		return m, tea.Batch(m.loadDir(m.path, m.offset), m.spinner.Tick)

	case clipboardDoneMsg:
		if msg.err != nil {
			return m.showMessage("✗ Cannot copy to clipboard: "+msg.err.Error(), true), nil
		}
		return m.showMessage("✓ URL copied to clipboard", false), nil

	// --- Upload progress ---
	case uploadProgressMsg:
		m.progress.Current = msg.BytesUploaded
		m.progress.Total = msg.TotalBytes
		m.progress.Percentage = msg.Percentage
		return m, cmdWaitUpload(m.uploadCh, m.uploadDone)

	case uploadDoneMsg:
		m.uploadCh = nil
		m.uploadDone = nil
		if msg.err != nil {
			m.progress.Err = msg.err
			m.progress.Done = true
			return m, nil
		}
		m.progress.Done = true
		// Brief done state, then reload
		m.loading = true
		m.mode = modeNormal
		return m, tea.Batch(m.loadDir(m.path, 0), m.spinner.Tick)

	// --- Download progress ---
	case downloadProgressMsg:
		m.progress.Current = msg.BytesDownloaded
		m.progress.Total = msg.TotalBytes
		m.progress.Percentage = msg.Percentage
		return m, cmdWaitDownload(m.dlCh, m.dlDone)

	case downloadDoneMsg:
		m.dlCh = nil
		m.dlDone = nil
		if msg.err != nil {
			m.pendingDownloads = nil
			m.progress.Err = msg.err
			m.progress.Done = true
			return m, nil
		}
		if len(m.pendingDownloads) > 0 {
			next := m.pendingDownloads[0]
			m.pendingDownloads = m.pendingDownloads[1:]
			localPath := path.Join(m.downloadDir, path.Base(next))
			m.progress = ProgressOverlay{Title: "Downloading", Filename: path.Base(next)}
			ch := make(chan disk.DownloadProgress, 32)
			done := make(chan downloadDoneMsg, 1)
			m.dlCh = ch
			m.dlDone = done
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
				defer cancel()
				err := m.client.DownloadFileToPathWithProgress(ctx, next, localPath, true,
					func(p disk.DownloadProgress) { ch <- p },
				)
				done <- downloadDoneMsg{err: err}
				close(ch)
				close(done)
			}()
			return m, cmdWaitDownload(ch, done)
		}
		m.selected = nil
		m.downloadDir = ""
		m.progress.Done = true

	// --- Key input ---
	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m BrowserModel) handleKey(msg tea.KeyMsg) (BrowserModel, tea.Cmd) {
	// Overlay modes intercept all keys first
	switch m.mode {

	case modeMessage:
		m.mode = modeNormal
		m.message = ""
		return m, nil

	case modeConfirmDelete:
		newDlg, confirmed, done := m.confirm.Update(msg)
		m.confirm = newDlg
		if done {
			if confirmed && len(m.entries) > 0 {
				target := m.entries[m.cursor].resource.Path
				m.mode = modeNormal
				return m, cmdDelete(m.client, target)
			}
			m.mode = modeNormal
		}
		return m, nil

	case modeConfirmBulkDelete:
		newDlg, confirmed, done := m.confirm.Update(msg)
		m.confirm = newDlg
		if done {
			if confirmed {
				paths := m.selectedPaths()
				m.mode = modeNormal
				if len(paths) == 0 {
					return m, nil
				}
				m.pendingDelete = paths[1:]
				return m, cmdDelete(m.client, paths[0])
			}
			m.mode = modeNormal
		}
		return m, nil

	case modeInputNewDir:
		newDlg, submitted, cancelled := m.inputDlg.Update(msg)
		m.inputDlg = newDlg
		if cancelled {
			m.mode = modeNormal
			return m, nil
		}
		if submitted {
			name := strings.TrimSpace(m.inputDlg.Value())
			m.mode = modeNormal
			if name == "" {
				return m, nil
			}
			newPath := path.Join(m.path, name)
			return m, cmdMkdir(m.client, newPath)
		}
		return m, nil

	case modeInputRename:
		newDlg, submitted, cancelled := m.inputDlg.Update(msg)
		m.inputDlg = newDlg
		if cancelled {
			m.mode = modeNormal
			return m, nil
		}
		if submitted {
			newName := strings.TrimSpace(m.inputDlg.Value())
			m.mode = modeNormal
			if newName == "" || len(m.entries) == 0 {
				return m, nil
			}
			from := m.entries[m.cursor].resource.Path
			to := path.Join(parentPath(from), newName)
			return m, cmdRename(m.client, from, to)
		}
		return m, nil

	case modeInputUpload:
		newDlg, submitted, cancelled := m.inputDlg.Update(msg)
		m.inputDlg = newDlg
		if cancelled {
			m.mode = modeNormal
			return m, nil
		}
		if submitted {
			localPath := strings.TrimSpace(m.inputDlg.Value())
			if localPath == "" {
				m.mode = modeNormal
				return m, nil
			}
			m.pendingUploadSrc = localPath
			suggested := path.Base(localPath)
			m.inputDlg = NewInputDialog("Upload — destination name", "Enter filename on Disk (↵ to keep as is)", suggested)
			m.inputDlg.SetValue(suggested)
			m.mode = modeInputUploadName
		}
		return m, nil

	case modeInputUploadName:
		newDlg, submitted, cancelled := m.inputDlg.Update(msg)
		m.inputDlg = newDlg
		if cancelled {
			m.mode = modeNormal
			m.pendingUploadSrc = ""
			return m, nil
		}
		if submitted {
			filename := strings.TrimSpace(m.inputDlg.Value())
			localPath := m.pendingUploadSrc
			m.pendingUploadSrc = ""
			m.mode = modeNormal
			if filename == "" {
				filename = path.Base(localPath)
			}
			remotePath := path.Join(m.path, filename)
			m.progress = ProgressOverlay{Title: "Uploading", Filename: filename}
			m.mode = modeUpload

			ch := make(chan disk.UploadProgress, 32)
			done := make(chan uploadDoneMsg, 1)
			m.uploadCh = ch
			m.uploadDone = done
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
				defer cancel()
				resource, err := m.client.UploadFileFromPathWithProgress(ctx, localPath, remotePath, true,
					func(p disk.UploadProgress) { ch <- p },
				)
				done <- uploadDoneMsg{resource: resource, err: err}
				close(ch)
				close(done)
			}()
			return m, cmdWaitUpload(ch, done)
		}
		return m, nil

	case modeInputUploadURL:
		newDlg, submitted, cancelled := m.inputDlg.Update(msg)
		m.inputDlg = newDlg
		if cancelled {
			m.mode = modeNormal
			return m, nil
		}
		if submitted {
			remoteURL := strings.TrimSpace(m.inputDlg.Value())
			if remoteURL == "" {
				m.mode = modeNormal
				return m, nil
			}
			m.pendingUploadSrc = remoteURL
			suggested := path.Base(remoteURL)
			if suggested == "." || suggested == "/" {
				suggested = ""
			}
			m.inputDlg = NewInputDialog("Upload URL — destination name", "Enter filename on Disk (↵ to keep as is)", "filename")
			m.inputDlg.SetValue(suggested)
			m.mode = modeInputUploadURLName
		}
		return m, nil

	case modeInputUploadURLName:
		newDlg, submitted, cancelled := m.inputDlg.Update(msg)
		m.inputDlg = newDlg
		if cancelled {
			m.mode = modeNormal
			m.pendingUploadSrc = ""
			return m, nil
		}
		if submitted {
			filename := strings.TrimSpace(m.inputDlg.Value())
			remoteURL := m.pendingUploadSrc
			m.pendingUploadSrc = ""
			m.mode = modeNormal
			destPath := path.Join(m.path, filename)
			return m, cmdUploadFromURL(m.client, remoteURL, destPath)
		}
		return m, nil

	case modeInputDownload:
		newDlg, submitted, cancelled := m.inputDlg.Update(msg)
		m.inputDlg = newDlg
		if cancelled {
			m.mode = modeNormal
			return m, nil
		}
		if submitted {
			localPath := strings.TrimSpace(m.inputDlg.Value())
			m.mode = modeNormal
			if localPath == "" || len(m.entries) == 0 {
				return m, nil
			}
			e := m.entries[m.cursor]
			m.progress = ProgressOverlay{
				Title:    "Downloading",
				Filename: e.resource.Name,
			}
			m.mode = modeDownload

			ch := make(chan disk.DownloadProgress, 32)
			done := make(chan downloadDoneMsg, 1)
			m.dlCh = ch
			m.dlDone = done

			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
				defer cancel()
				err := m.client.DownloadFileToPathWithProgress(ctx, e.resource.Path, localPath, true,
					func(p disk.DownloadProgress) { ch <- p },
				)
				done <- downloadDoneMsg{err: err}
				close(ch)
				close(done)
			}()

			return m, cmdWaitDownload(ch, done)
		}
		return m, nil

	case modeInputDownloadDir:
		newDlg, submitted, cancelled := m.inputDlg.Update(msg)
		m.inputDlg = newDlg
		if cancelled {
			m.mode = modeNormal
			return m, nil
		}
		if submitted {
			dir := strings.TrimSpace(m.inputDlg.Value())
			m.mode = modeNormal
			if dir == "" {
				return m, nil
			}
			files := m.selectedFiles()
			if len(files) == 0 {
				return m, nil
			}
			first := files[0]
			localPath := path.Join(dir, path.Base(first))
			m.progress = ProgressOverlay{Title: "Downloading", Filename: path.Base(first)}
			m.mode = modeDownload
			m.pendingDownloads = files[1:]
			m.downloadDir = dir

			ch := make(chan disk.DownloadProgress, 32)
			done := make(chan downloadDoneMsg, 1)
			m.dlCh = ch
			m.dlDone = done
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
				defer cancel()
				err := m.client.DownloadFileToPathWithProgress(ctx, first, localPath, true,
					func(p disk.DownloadProgress) { ch <- p },
				)
				done <- downloadDoneMsg{err: err}
				close(ch)
				close(done)
			}()
			return m, cmdWaitDownload(ch, done)
		}
		return m, nil

	case modeUpload, modeDownload:
		if m.progress.Done {
			m.mode = modeNormal
			m.progress = ProgressOverlay{}
		}
		return m, nil

	case modePublicURL:
		switch msg.String() {
		case "c":
			if m.publicURL != "" {
				return m, cmdCopyToClipboard(m.publicURL)
			}
		case "u":
			if len(m.entries) > 0 {
				target := m.entries[m.cursor].resource.Path
				m.mode = modeNormal
				m.publicURL = ""
				return m, cmdUnpublish(m.client, target)
			}
		default:
			m.mode = modeNormal
			m.publicURL = ""
		}
		return m, nil

	case modeMetadata:
		m.mode = modeNormal
		return m, nil
	}

	// Esc clears selection in normal mode
	if msg.String() == "esc" {
		m.selected = nil
		return m, nil
	}

	// --- Normal mode ---

	// Clear load error
	if m.err != nil {
		m.err = nil
		return m, nil
	}
	if m.loading {
		return m, nil
	}

	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
		} else if m.offset > 0 {
			m.offset -= pageSize
			if m.offset < 0 {
				m.offset = 0
			}
			m.cursorAfterLoad = -1 // land on last item of previous page
			m.loading = true
			return m, tea.Batch(m.loadDir(m.path, m.offset), m.spinner.Tick)
		}

	case key.Matches(msg, m.keys.Down):
		if m.cursor < len(m.entries)-1 {
			m.cursor++
		} else if m.offset+len(m.entries) < m.total {
			m.offset += pageSize
			m.cursor = 0
			m.loading = true
			return m, tea.Batch(m.loadDir(m.path, m.offset), m.spinner.Tick)
		}

	case key.Matches(msg, m.keys.Enter):
		if len(m.entries) > 0 && m.entries[m.cursor].isDir() {
			m.loading = true
			m.offset = 0
			return m, tea.Batch(m.loadDir(m.entries[m.cursor].resource.Path, 0), m.spinner.Tick)
		}

	case key.Matches(msg, m.keys.Back):
		if parent := parentPath(m.path); parent != m.path {
			m.loading = true
			m.offset = 0
			return m, tea.Batch(m.loadDir(parent, 0), m.spinner.Tick)
		}

	case key.Matches(msg, m.keys.Sort):
		m.sort = nextSort(m.sort)
		m.offset = 0
		m.cursorAfterLoad = 0
		m.loading = true
		return m, tea.Batch(m.loadDir(m.path, 0), m.spinner.Tick)

	case key.Matches(msg, m.keys.Refresh):
		m.loading = true
		return m, tea.Batch(m.loadDir(m.path, m.offset), m.spinner.Tick)

	case key.Matches(msg, m.keys.NewDir):
		m.inputDlg = NewInputDialog("New directory", "Enter name for the new folder", "folder name")
		m.mode = modeInputNewDir

	case key.Matches(msg, m.keys.Rename):
		if len(m.entries) == 0 {
			return m, nil
		}
		m.inputDlg = NewInputDialog("Rename", "", "new name")
		m.inputDlg.SetValue(m.entries[m.cursor].resource.Name)
		m.mode = modeInputRename

	case key.Matches(msg, m.keys.Select):
		if len(m.entries) == 0 {
			return m, nil
		}
		if m.selected == nil {
			m.selected = make(map[string]bool)
		}
		path := m.entries[m.cursor].resource.Path
		m.selected[path] = !m.selected[path]
		if !m.selected[path] {
			delete(m.selected, path)
		}
		// advance cursor
		if m.cursor < len(m.entries)-1 {
			m.cursor++
		} else if m.offset+len(m.entries) < m.total {
			m.offset += pageSize
			m.cursorAfterLoad = 0
			m.loading = true
			return m, tea.Batch(m.loadDir(m.path, m.offset), m.spinner.Tick)
		}

	case key.Matches(msg, m.keys.SelectAll):
		if len(m.entries) == 0 {
			return m, nil
		}
		if len(m.selected) == len(m.entries) {
			m.selected = nil
		} else {
			m.selected = make(map[string]bool, len(m.entries))
			for _, e := range m.entries {
				m.selected[e.resource.Path] = true
			}
		}

	case key.Matches(msg, m.keys.Delete):
		if len(m.entries) == 0 {
			return m, nil
		}
		if len(m.selected) > 0 {
			m.confirm = NewConfirmDialog("Delete", fmt.Sprintf("Delete %d selected items?", len(m.selected)))
			m.mode = modeConfirmBulkDelete
		} else {
			target := m.entries[m.cursor].resource.Name
			m.confirm = NewConfirmDialog("Delete", "Delete \""+target+"\"?")
			m.mode = modeConfirmDelete
		}

	case key.Matches(msg, m.keys.Upload):
		m.inputDlg = NewInputDialog("Upload file", "Enter the local path of the file to upload", "/path/to/file")
		m.mode = modeInputUpload

	case key.Matches(msg, m.keys.UploadURL):
		m.inputDlg = NewInputDialog("Upload from URL", "Enter the URL to upload to current directory", "https://")
		m.mode = modeInputUploadURL

	case key.Matches(msg, m.keys.Download):
		if len(m.selected) > 0 {
			m.inputDlg = NewInputDialog("Download selected", "Enter local destination directory", "./")
			m.inputDlg.SetValue("./")
			m.mode = modeInputDownloadDir
		} else {
			if len(m.entries) == 0 || m.entries[m.cursor].isDir() {
				return m, nil
			}
			defaultPath := "./" + m.entries[m.cursor].resource.Name
			m.inputDlg = NewInputDialog("Download file", "Enter local destination path", defaultPath)
			m.inputDlg.SetValue(defaultPath)
			m.mode = modeInputDownload
		}

	case key.Matches(msg, m.keys.Meta):
		if len(m.entries) > 0 {
			m.mode = modeMetadata
		}

	case key.Matches(msg, m.keys.Publish):
		if len(m.entries) == 0 {
			return m, nil
		}
		e := m.entries[m.cursor]
		if e.resource.PublicURL != "" {
			// Already published — show URL overlay with options
			m.publicURL = e.resource.PublicURL
			m.mode = modePublicURL
			return m, nil
		}
		return m, cmdPublish(m.client, e.resource.Path)

	case key.Matches(msg, m.keys.CopyURL):
		if len(m.entries) == 0 {
			return m, nil
		}
		url := m.entries[m.cursor].resource.PublicURL
		if url == "" {
			return m.showMessage("✗ No public URL - press p to publish first", true), nil
		}
		return m, cmdCopyToClipboard(url)
	}

	return m, nil
}

// IsInputActive reports whether the browser is in a mode that captures
// all keystrokes (text input, confirmation dialog, etc.).
// app.go uses this to suppress global hotkeys like t/i while typing.
func (m BrowserModel) IsInputActive() bool {
	switch m.mode {
	case modeInputNewDir, modeInputRename,
		modeInputUpload, modeInputUploadName,
		modeInputUploadURL, modeInputUploadURLName,
		modeInputDownload, modeInputDownloadDir,
		modeConfirmDelete, modeConfirmBulkDelete,
		modePublicURL, modeMetadata, modeMessage:
		return true
	}
	return false
}

// showMessage sets the model into modeMessage with the given text.
func (m BrowserModel) showMessage(msg string, isError bool) BrowserModel {
	m.mode = modeMessage
	m.message = msg
	m.messageIsError = isError
	return m
}

// ---- View ------------------------------------------------------------------

func (m BrowserModel) View() string {
	if m.width == 0 {
		return ""
	}

	base := m.viewBase()

	// Render overlay on top of the base view
	switch m.mode {
	case modeConfirmDelete, modeConfirmBulkDelete:
		return renderOverlay(base, m.confirm.View(m.width), m.width, m.height)
	case modeInputNewDir, modeInputRename,
		modeInputUpload, modeInputUploadName,
		modeInputUploadURL, modeInputUploadURLName,
		modeInputDownload, modeInputDownloadDir:
		return renderOverlay(base, m.inputDlg.View(m.width), m.width, m.height)
	case modeUpload, modeDownload:
		return renderOverlay(base, m.progress.View(m.width), m.width, m.height)
	case modeMessage:
		var msgView string
		if m.messageIsError {
			msgView = StyleDialog.Render(StyleError.Render(m.message) + "\n\n" + StyleMuted("Press any key to continue"))
		} else {
			msgView = StyleDialog.Render(StyleSuccess.Render(m.message) + "\n\n" + StyleMuted("Press any key to continue"))
		}
		return renderOverlay(base, msgView, m.width, m.height)

	case modePublicURL:
		urlStyle := lipgloss.NewStyle().Foreground(colorAccent)
		content := StyleSuccess.Render("⇡ Public link") + "\n\n" +
			urlStyle.Render(m.publicURL) + "\n\n" +
			StyleStatusKey.Render("c") + " copy   " +
			StyleStatusKey.Render("u") + " unpublish   " +
			StyleMuted("any other key to close")
		overlay := StyleDialog.Render(content)
		return renderOverlay(base, overlay, m.width, m.height)

	case modeMetadata:
		if len(m.entries) > 0 {
			overlay := m.viewMetadata(m.entries[m.cursor])
			return renderOverlay(base, overlay, m.width, m.height)
		}
	}

	return base
}

func (m BrowserModel) viewBase() string {
	var b strings.Builder
	b.WriteString(m.viewTitleBar())
	b.WriteByte('\n')

	contentHeight := m.height - 2 // title line + status bar
	if m.loading {
		b.WriteString(m.viewLoading(contentHeight))
	} else if m.err != nil {
		b.WriteString(m.viewError(contentHeight))
	} else {
		b.WriteString(m.viewList(contentHeight))
	}

	b.WriteByte('\n')
	b.WriteString(m.viewStatusBar())
	return b.String()
}

func (m BrowserModel) viewTitleBar() string {
	title := StyleTitle.Render("  YaD")
	pathStr := StylePath.Render(m.path)
	sortStr := StyleMuted("  " + sortLabel(m.sort))
	gap := max(m.width-lipgloss.Width(title)-lipgloss.Width(pathStr)-lipgloss.Width(sortStr), 0)
	return title + pathStr + strings.Repeat(" ", gap) + sortStr
}

func (m BrowserModel) viewLoading(height int) string {
	msg := m.spinner.View() + " Loading…"
	lines := make([]string, height)
	for i := range lines {
		if i == height/2 {
			lines[i] = lipgloss.PlaceHorizontal(m.width, lipgloss.Center, msg)
		}
	}
	return strings.Join(lines, "\n")
}

func (m BrowserModel) viewError(height int) string {
	lines := make([]string, height)
	mid := height / 2
	for i := range lines {
		switch i {
		case mid:
			lines[i] = lipgloss.PlaceHorizontal(m.width, lipgloss.Center,
				StyleError.Render("✗ "+m.err.Error()))
		case mid + 1:
			lines[i] = lipgloss.PlaceHorizontal(m.width, lipgloss.Center,
				StyleMuted("Press any key to continue"))
		}
	}
	return strings.Join(lines, "\n")
}

func (m BrowserModel) viewList(height int) string {
	if len(m.entries) == 0 {
		lines := make([]string, height)
		lines[height/2] = lipgloss.PlaceHorizontal(m.width, lipgloss.Center, StyleMuted("(empty directory)"))
		return strings.Join(lines, "\n")
	}

	visible := min(height, len(m.entries))
	start := max(m.cursor-visible+1, 0)
	start = min(start, m.cursor)
	nameWidth := max(m.width-9-18-4-2, 10)

	var rows []string
	for i := start; i < start+visible && i < len(m.entries); i++ {
		e := m.entries[i]
		selected := i == m.cursor

		name := e.resource.Name
		if len(name) > nameWidth {
			name = name[:nameWidth-1] + "…"
		}

		mark := "  "
		if m.selected[e.resource.Path] {
			mark = StyleSuccess.Render("✓ ")
		}

		var nameStyled string
		if e.isDir() {
			nameStyled = mark + e.icon() + StyleDir.Render(name)
		} else {
			nameStyled = mark + e.icon() + StyleFile.Render(name)
		}

		pubMark := "  "
		if e.resource.PublicURL != "" {
			pubMark = lipgloss.NewStyle().Foreground(colorAccent).Render("⇡ ")
		}

		row := lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Width(nameWidth+2).Render(nameStyled),
			StyleSize.Render(e.sizeStr()),
			"  ",
			StyleDate.Render(e.modifiedStr()),
			pubMark,
		)

		published := e.resource.PublicURL != ""
		switch {
		case selected && published:
			row = StyleItemPublishedSelected.Width(m.width).Render(row)
		case selected:
			row = StyleItemSelected.Width(m.width).Render(row)
		case published:
			row = StyleItemPublished.Width(m.width).Render(row)
		default:
			row = StyleItemNormal.Width(m.width).Render(row)
		}
		rows = append(rows, row)
	}

	for len(rows) < height {
		rows = append(rows, strings.Repeat(" ", m.width))
	}
	return strings.Join(rows, "\n")
}

func (m BrowserModel) viewStatusBar() string {
	left := ""
	if len(m.entries) > 0 {
		left = fmt.Sprintf("%d/%d", m.cursor+1+m.offset, m.total)
	}
	if n := len(m.selected); n > 0 {
		left += "  " + StyleSuccess.Render(fmt.Sprintf("%d selected", n))
	}

	hints := []string{
		StyleStatusKey.Render("↑↓") + " move",
		StyleStatusKey.Render("↵") + " open",
		StyleStatusKey.Render("spc") + " select",
		StyleStatusKey.Render("u") + " upload",
		StyleStatusKey.Render("U") + " URL",
		StyleStatusKey.Render("d") + " download",
		StyleStatusKey.Render("n") + " mkdir",
		StyleStatusKey.Render("r") + " rename",
		StyleStatusKey.Render("D") + " delete",
		StyleStatusKey.Render("s") + " sort",
		StyleStatusKey.Render("p") + " publish",
		StyleStatusKey.Render("m") + " info",
		StyleStatusKey.Render("t") + " trash",
		StyleStatusKey.Render("i") + " disk info",
		StyleStatusKey.Render("q") + " quit",
	}
	right := strings.Join(hints, "  ")

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 1 {
		gap = 1
	}
	bar := left + strings.Repeat(" ", gap) + right
	return StyleStatusBar.Width(m.width).Render(bar)
}

// selectedPaths returns paths of all selected entries in stable order.
func (m BrowserModel) selectedPaths() []string {
	var paths []string
	for _, e := range m.entries {
		if m.selected[e.resource.Path] {
			paths = append(paths, e.resource.Path)
		}
	}
	return paths
}

// selectedFiles returns paths of selected entries that are files (not dirs).
func (m BrowserModel) selectedFiles() []string {
	var paths []string
	for _, e := range m.entries {
		if m.selected[e.resource.Path] && !e.isDir() {
			paths = append(paths, e.resource.Path)
		}
	}
	return paths
}

func (m BrowserModel) viewMetadata(e entry) string {
	r := e.resource

	label := lipgloss.NewStyle().Foreground(colorMuted).Width(10)
	value := lipgloss.NewStyle().Foreground(colorAccent)

	row := func(k, v string) string {
		if v == "" {
			return ""
		}
		return label.Render(k) + value.Render(v) + "\n"
	}

	parseTime := func(s string) string {
		if s == "" {
			return ""
		}
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return s
		}
		return t.Format("2006-01-02  15:04:05")
	}

	kind := "file"
	if e.isDir() {
		kind = "directory"
	}

	var sb strings.Builder
	sb.WriteString(StyleTitle.Render(" Info") + "\n\n")
	sb.WriteString(row("Name", r.Name))
	sb.WriteString(row("Type", kind))
	if !e.isDir() {
		sb.WriteString(row("Size", disk.FormatFileSize(int64(r.Size))))
		sb.WriteString(row("MIME", r.MimeType))
		if r.MediaType != "" {
			sb.WriteString(row("Media", r.MediaType))
		}
	}
	sb.WriteString(row("Created", parseTime(r.Created)))
	sb.WriteString(row("Modified", parseTime(r.Modified)))
	if r.Md5 != "" {
		sb.WriteString(row("MD5", r.Md5))
	}
	if r.Sha256 != "" {
		sb.WriteString(row("SHA256", r.Sha256[:16]+"…"))
	}
	if r.PublicURL != "" {
		sb.WriteString(row("Public", r.PublicURL))
	}
	sb.WriteString("\n" + StyleMuted("any key to close"))

	return StyleDialog.Render(sb.String())
}

// renderOverlay centres a dialog on top of a base view string.
func renderOverlay(base, overlay string, width, height int) string {
	overlayLines := strings.Split(overlay, "\n")
	baseLines := strings.Split(base, "\n")

	oH := len(overlayLines)
	oW := 0
	for _, l := range overlayLines {
		if w := lipgloss.Width(l); w > oW {
			oW = w
		}
	}

	startY := max((height-oH)/2, 0)
	startX := max((width-oW)/2, 0)

	for y, ol := range overlayLines {
		row := startY + y
		if row >= len(baseLines) {
			break
		}
		bl := baseLines[row]
		// Pad base line to full width
		blW := lipgloss.Width(bl)
		if blW < width {
			bl += strings.Repeat(" ", width-blW)
		}
		// Splice overlay into base line at startX
		prefix := truncateToWidth(bl, startX)
		baseLines[row] = prefix + ol
	}

	return strings.Join(baseLines, "\n")
}

// truncateToWidth returns the leading portion of s that fits within w visible columns.
func truncateToWidth(s string, w int) string {
	if w <= 0 {
		return ""
	}
	var buf strings.Builder
	col := 0
	for _, r := range s {
		if col+1 > w {
			break
		}
		buf.WriteRune(r)
		col++
	}
	if col < w {
		buf.WriteString(strings.Repeat(" ", w-col))
	}
	return buf.String()
}

// parentPath returns the parent of a Yandex Disk path.
func parentPath(p string) string {
	if p == "/" || p == "disk:/" {
		return p
	}
	p = strings.TrimRight(p, "/")
	idx := strings.LastIndex(p, "/")
	if idx <= 0 {
		return "/"
	}
	parent := p[:idx]
	if parent == "disk:" {
		return "disk:/"
	}
	return parent
}

// StyleMuted renders text in the muted color.
func StyleMuted(s string) string {
	return lipgloss.NewStyle().Foreground(colorMuted).Render(s)
}
