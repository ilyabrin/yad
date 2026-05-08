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
	modeNormal        browserMode = iota
	modeConfirmDelete             // delete confirmation dialog
	modeInputNewDir               // new directory name input
	modeInputRename               // rename input
	modeInputUpload               // local file path input before upload
	modeInputDownload             // local destination path input before download
	modeUpload                    // upload in progress
	modeDownload                  // download in progress
	modeMessage                   // transient success / error message
	modePublicURL                 // showing public URL after publish
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

	// Terminal size
	width  int
	height int

	// Navigation state
	loading bool
	err     error // last load error (cleared on keypress)

	// Overlay state
	mode           browserMode
	confirm        ConfirmDialog
	inputDlg       InputDialog
	progress       ProgressOverlay
	message        string // shown in modeMessage
	messageIsError bool
	publicURL      string // shown in modePublicURL

	// Active async channels (nil when idle)
	uploadCh   <-chan disk.UploadProgress
	uploadDone <-chan uploadDoneMsg
	dlCh       <-chan disk.DownloadProgress
	dlDone     <-chan downloadDoneMsg
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
			Sort:   "name",
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
		if m.offset == 0 {
			m.cursor = 0
		}

	// --- Operation results ---
	case deleteDoneMsg:
		if msg.err != nil {
			return m.showMessage("✗ "+msg.err.Error(), true), nil
		}
		m.mode = modeNormal
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
		m.progress.Err = msg.err
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
			m.mode = modeNormal
			if localPath == "" {
				return m, nil
			}
			// Destination: current directory + filename
			filename := path.Base(localPath)
			remotePath := path.Join(m.path, filename)
			m.progress = ProgressOverlay{
				Title:    "Uploading",
				Filename: filename,
			}
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
		default:
			m.mode = modeNormal
			m.publicURL = ""
		}
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

	case key.Matches(msg, m.keys.Delete):
		if len(m.entries) == 0 {
			return m, nil
		}
		target := m.entries[m.cursor].resource.Name
		m.confirm = NewConfirmDialog("Delete", "Delete \""+target+"\"?")
		m.mode = modeConfirmDelete

	case key.Matches(msg, m.keys.Upload):
		m.inputDlg = NewInputDialog("Upload file", "Enter the local path of the file to upload", "/path/to/file")
		m.mode = modeInputUpload

	case key.Matches(msg, m.keys.Download):
		if len(m.entries) == 0 || m.entries[m.cursor].isDir() {
			return m, nil
		}
		defaultPath := "./" + m.entries[m.cursor].resource.Name
		m.inputDlg = NewInputDialog("Download file", "Enter local destination path", defaultPath)
		m.inputDlg.SetValue(defaultPath)
		m.mode = modeInputDownload

	case key.Matches(msg, m.keys.Publish):
		if len(m.entries) == 0 {
			return m, nil
		}
		e := m.entries[m.cursor]
		if e.resource.PublicURL != "" {
			// Already public - unpublish
			return m, cmdUnpublish(m.client, e.resource.Path)
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
	case modeConfirmDelete:
		return renderOverlay(base, m.confirm.View(m.width), m.width, m.height)
	case modeInputNewDir, modeInputRename, modeInputUpload, modeInputDownload:
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
		content := StyleSuccess.Render("✓ Published") + "\n\n" +
			urlStyle.Render(m.publicURL) + "\n\n" +
			StyleStatusKey.Render("c") + " copy to clipboard   " +
			StyleMuted("any other key to close")
		overlay := StyleDialog.Render(content)
		return renderOverlay(base, overlay, m.width, m.height)
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
	gap := m.width - lipgloss.Width(title) - lipgloss.Width(pathStr)
	if gap < 0 {
		gap = 0
	}
	return title + pathStr + strings.Repeat(" ", gap)
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

	visible := height
	if visible > len(m.entries) {
		visible = len(m.entries)
	}

	start := m.cursor - visible + 1
	if start < 0 {
		start = 0
	}
	if m.cursor < start {
		start = m.cursor
	}

	nameWidth := m.width - 9 - 18 - 4
	if nameWidth < 10 {
		nameWidth = 10
	}

	var rows []string
	for i := start; i < start+visible && i < len(m.entries); i++ {
		e := m.entries[i]
		selected := i == m.cursor

		name := e.resource.Name
		if len(name) > nameWidth {
			name = name[:nameWidth-1] + "…"
		}

		var nameStyled string
		if e.isDir() {
			nameStyled = e.icon() + StyleDir.Render(name)
		} else {
			nameStyled = e.icon() + StyleFile.Render(name)
		}

		row := lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Width(nameWidth+2).Render(nameStyled),
			StyleSize.Render(e.sizeStr()),
			"  ",
			StyleDate.Render(e.modifiedStr()),
		)

		if selected {
			row = StyleItemSelected.Width(m.width).Render(row)
		} else {
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

	hints := []string{
		StyleStatusKey.Render("↑↓") + " move",
		StyleStatusKey.Render("↵") + " open",
		StyleStatusKey.Render("u") + " upload",
		StyleStatusKey.Render("d") + " download",
		StyleStatusKey.Render("n") + " mkdir",
		StyleStatusKey.Render("r") + " rename",
		StyleStatusKey.Render("D") + " delete",
		StyleStatusKey.Render("p") + " publish",
		StyleStatusKey.Render("i") + " info",
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

	startY := (height - oH) / 2
	if startY < 0 {
		startY = 0
	}
	startX := (width - oW) / 2
	if startX < 0 {
		startX = 0
	}

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
	result := ""
	col := 0
	for _, r := range s {
		rw := 1
		if col+rw > w {
			break
		}
		result += string(r)
		col += rw
	}
	if col < w {
		result += strings.Repeat(" ", w-col)
	}
	return result
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
	return p[:idx]
}

// StyleMuted renders text in the muted color.
func StyleMuted(s string) string {
	return lipgloss.NewStyle().Foreground(colorMuted).Render(s)
}
