package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/ilyabrin/disk"
)

const pageSize = 100

// browserMode is the current overlay / interaction state.
type browserMode int

const (
	modeNormal            browserMode = iota
	modeConfirmDelete                 // delete confirmation dialog
	modeInputNewDir                   // new directory name input
	modeInputRename                   // rename input
	modeInputUpload                   // local file path input before upload
	modeInputDownload                 // local destination path input before download
	modeUpload                        // upload in progress
	modeDownload                      // download in progress
	modeMessage                       // transient success / error message
	modePublicURL                     // showing public URL after publish
	modeMetadata                      // file / directory metadata overlay
	modeConfirmBulkDelete             // confirm deletion of selected items
	modeInputDownloadDir              // destination directory for bulk download
	modeInputUploadURL                // remote URL to upload from
	modeInputUploadName               // confirm/change filename before upload (local or URL)
	modeFilter                        // live name filter — search bar shown in status bar
	modeConfirmQuit                   // quit confirmation while transfer is in progress
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
	return disk.FormatFileSize(e.resource.Size)
}

func (e entry) modifiedStr() string {
	s := e.resource.Modified
	if s == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		// Truncate to date portion only; guard against unexpectedly short strings.
		if len(s) >= 10 {
			return s[:10]
		}
		return s
	}
	return t.Format("2006-01-02 15:04")
}

// loadedMsg is sent when a directory listing fetch completes.
type loadedMsg struct {
	path    string
	entries []entry
	total   int
	err     error
}

// BrowserModel is the main file browser screen. It renders a paginated
// directory listing with multi-selection, sorting, upload/download, and
// publish support. The model is split across browser_model.go (types and
// helpers), browser_update.go (Update / handleKey), and browser_view.go (View).
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
	cursorAfterLoad int   // -1 = last item, ≥0 = explicit position (set before page load)
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
	pendingUploadSrc   string // local path or URL held between step 1 and step 2
	pendingUploadIsURL bool   // true when pendingUploadSrc is a remote URL

	// Multi-selection
	selected         map[string]bool // resource path → selected
	pendingDelete    []string        // paths queued for sequential bulk delete
	pendingDownloads []string        // remote paths queued for sequential bulk download
	downloadDir      string          // local destination dir for bulk download

	// Filter state
	filter      string          // current name filter query; empty = no filter
	filterInput textinput.Model // text input used while in modeFilter

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

// NewBrowserModel creates a BrowserModel rooted at lastPath (falls back to "/").
// defaultSort sets the initial sort order; empty string falls back to "name".
func NewBrowserModel(client *disk.Client, defaultSort, lastPath string) BrowserModel {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(colorPrimary)

	fi := textinput.New()
	fi.Placeholder = "filter…"
	fi.CharLimit = 128

	sort := defaultSort
	if sort == "" {
		sort = "name"
	}
	path := lastPath
	if path == "" {
		path = "/"
	}

	return BrowserModel{
		client:      client,
		keys:        DefaultBrowserKeyMap(),
		spinner:     sp,
		filterInput: fi,
		path:        path,
		sort:        sort,
		loading:     true,
	}
}

// IsInputActive reports whether the browser is in a mode that captures all
// keystrokes. app.go uses this to suppress global hotkeys like t/i while typing.
func (m BrowserModel) IsInputActive() bool {
	return m.mode != modeNormal
}

// showMessage sets the model into modeMessage with the given text.
func (m BrowserModel) showMessage(msg string, isError bool) BrowserModel {
	m.mode = modeMessage
	m.message = msg
	m.messageIsError = isError
	return m
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

// visibleEntries returns entries after applying the active name filter.
// When filter is empty the full loaded page is returned as-is.
func (m BrowserModel) visibleEntries() []entry {
	if m.filter == "" {
		return m.entries
	}
	q := strings.ToLower(m.filter)
	out := make([]entry, 0, len(m.entries))
	for _, e := range m.entries {
		if strings.Contains(strings.ToLower(e.resource.Name), q) {
			out = append(out, e)
		}
	}
	return out
}

// currentEntry returns the entry under the cursor, honouring the active
// filter. Every action that operates on "the highlighted row" MUST go through
// this — indexing m.entries directly targets the wrong resource whenever a
// filter is on, because the view renders visibleEntries().
func (m BrowserModel) currentEntry() (entry, bool) {
	entries := m.visibleEntries()
	if m.cursor < 0 || m.cursor >= len(entries) {
		return entry{}, false
	}
	return entries[m.cursor], true
}

// visibleCount is the number of rows currently rendered in the list.
func (m BrowserModel) visibleCount() int { return len(m.visibleEntries()) }

// clampCursor keeps the cursor inside the visible list after the filter or the
// entry set changes.
func (m *BrowserModel) clampCursor() {
	n := m.visibleCount()
	if m.cursor >= n {
		m.cursor = max(0, n-1)
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m *BrowserModel) setClient(c *disk.Client) { m.client = c }

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
