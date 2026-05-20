package tui

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/ilyabrin/disk"
)

func (m BrowserModel) Init() tea.Cmd {
	return m.reloadCmd()
}

// reloadCmd reloads the current directory at the current offset.
func (m BrowserModel) reloadCmd() tea.Cmd {
	return tea.Batch(m.loadDir(m.path, m.offset), m.spinner.Tick)
}

// loadDir returns a Cmd that fetches directory contents.
func (m BrowserModel) loadDir(p string, offset int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeoutMeta)
		defer cancel()

		opts := &disk.ResourceOptions{
			Limit:  pageSize,
			Offset: offset,
			Sort:   m.sort,
		}
		resource, errResp := m.client.GetMetadataWithOptions(ctx, p, opts)
		if errResp != nil {
			return loadedMsg{path: p, err: fmt.Errorf("%s: %s", errResp.Error, errResp.Message)}
		}

		var entries []entry
		total := 0
		if resource.Embedded != nil {
			total = resource.Embedded.Total
			for _, r := range resource.Embedded.Items {
				entries = append(entries, entry{resource: r})
			}
		}
		return loadedMsg{path: p, entries: entries, total: total}
	}
}

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

	case loadedMsg:
		m.loading = false
		if msg.err != nil {
			if isAuthError(msg.err) {
				return m, func() tea.Msg { return tryRefreshMsg{origErr: msg.err} }
			}
			if fe := asFatalErrorMsg(msg.err); fe != nil {
				return m, func() tea.Msg { return *fe }
			}
			// Saved path no longer exists — silently fall back to root
			// rather than showing an error on startup.
			if m.path != "/" && m.path != "disk:/" && m.offset == 0 {
				m.path = "/"
				return m, m.reloadCmd()
			}
			m.err = msg.err
			return m, nil
		}
		// Clear filter when navigating to a new directory.
		if msg.path != m.path {
			m.filter = ""
			m.filterInput.SetValue("")
			m.mode = modeNormal
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

	case deleteDoneMsg:
		if msg.err != nil {
			m.pendingDelete = nil
			return m.showMessage(iconErr+" "+msg.err.Error(), true), nil
		}
		if len(m.pendingDelete) > 0 {
			next := m.pendingDelete[0]
			m.pendingDelete = m.pendingDelete[1:]
			return m, cmdDelete(m.client, next)
		}
		m.mode = modeNormal
		m.selected = nil
		m.loading = true
		return m, m.reloadCmd()

	case mkdirDoneMsg:
		if msg.err != nil {
			return m.showMessage(iconErr+" "+msg.err.Error(), true), nil
		}
		m.mode = modeNormal
		m.loading = true
		return m, m.reloadCmd()

	case renameDoneMsg:
		if msg.err != nil {
			return m.showMessage(iconErr+" "+msg.err.Error(), true), nil
		}
		m.mode = modeNormal
		m.loading = true
		return m, m.reloadCmd()

	case publishDoneMsg:
		if msg.err != nil {
			return m.showMessage(iconErr+" "+msg.err.Error(), true), nil
		}
		m.mode = modePublicURL
		m.publicURL = msg.publicURL
		m.loading = true
		return m, m.reloadCmd()

	case unpublishDoneMsg:
		if msg.err != nil {
			return m.showMessage(iconErr+" "+msg.err.Error(), true), nil
		}
		m.loading = true
		m.mode = modeNormal
		return m, m.reloadCmd()

	case uploadFromURLDoneMsg:
		if msg.err != nil {
			return m.showMessage(iconErr+" "+msg.err.Error(), true), nil
		}
		m.loading = true
		return m, m.reloadCmd()

	case clipboardDoneMsg:
		if msg.err != nil {
			return m.showMessage(iconErr+" Cannot copy to clipboard: "+msg.err.Error(), true), nil
		}
		return m.showMessage(iconOK+" URL copied to clipboard", false), nil

	case uploadStartedMsg:
		m.uploadCh = msg.ch
		m.uploadDone = msg.done
		return m, cmdWaitUpload(m.uploadCh, m.uploadDone)

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
		m.loading = true
		m.mode = modeNormal
		return m, m.reloadCmd()

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
			filename := path.Base(next)
			m.progress = ProgressOverlay{Title: "Downloading", Filename: filename}
			return m, cmdStartDownload(m.client, next, localPath, filename)
		}
		m.selected = nil
		m.downloadDir = ""
		m.progress.Done = true

	case downloadStartedMsg:
		m.dlCh = msg.ch
		m.dlDone = msg.done
		return m, cmdWaitDownload(m.dlCh, m.dlDone)

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m BrowserModel) handleKey(msg tea.KeyMsg) (BrowserModel, tea.Cmd) {
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
			return m, cmdMkdir(m.client, path.Join(m.path, name))
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
			m.pendingUploadIsURL = false
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
			src := m.pendingUploadSrc
			isURL := m.pendingUploadIsURL
			m.pendingUploadSrc = ""
			m.mode = modeNormal
			if isURL {
				if filename == "" {
					return m, nil
				}
				return m, cmdUploadFromURL(m.client, src, path.Join(m.path, filename))
			}
			if filename == "" {
				filename = path.Base(src)
			}
			remotePath := path.Join(m.path, filename)
			m.progress = ProgressOverlay{Title: "Uploading", Filename: filename}
			m.mode = modeUpload
			return m, cmdStartUpload(m.client, src, remotePath)
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
			m.pendingUploadIsURL = true
			suggested := path.Base(remoteURL)
			if suggested == "." || suggested == "/" {
				suggested = ""
			}
			m.inputDlg = NewInputDialog("Upload URL — destination name", "Enter filename on Disk (↵ to keep as is)", "filename")
			m.inputDlg.SetValue(suggested)
			m.mode = modeInputUploadName
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
			m.progress = ProgressOverlay{Title: "Downloading", Filename: e.resource.Name}
			m.mode = modeDownload
			return m, cmdStartDownload(m.client, e.resource.Path, localPath, e.resource.Name)
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
			filename := path.Base(first)
			localPath := path.Join(dir, filename)
			m.progress = ProgressOverlay{Title: "Downloading", Filename: filename}
			m.mode = modeDownload
			m.pendingDownloads = files[1:]
			m.downloadDir = dir
			return m, cmdStartDownload(m.client, first, localPath, filename)
		}
		return m, nil

	case modeUpload, modeDownload:
		switch msg.String() {
		case "q", "ctrl+c":
			op := "upload"
			if m.mode == modeDownload {
				op = "download"
			}
			m.confirm = NewConfirmDialog("Quit", "Abort "+op+" in progress and quit?")
			m.mode = modeConfirmQuit
			return m, nil
		}
		if m.progress.Done {
			m.mode = modeNormal
			m.progress = ProgressOverlay{}
		}
		return m, nil

	case modeConfirmQuit:
		newDlg, confirmed, done := m.confirm.Update(msg)
		m.confirm = newDlg
		if done {
			if confirmed {
				return m, tea.Quit
			}
			// Cancelled — return to the active transfer mode
			if m.dlCh != nil {
				m.mode = modeDownload
			} else {
				m.mode = modeUpload
			}
		}
		return m, nil

	case modePublicURL:
		switch msg.String() {
		case "c":
			if m.publicURL != "" {
				return m, cmdCopyToClipboard(m.publicURL)
			}
		case "o":
			if m.publicURL != "" {
				return m, cmdOpenBrowser(m.publicURL)
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

	case modeFilter:
		switch msg.Type {
		case tea.KeyEsc:
			// Esc cancels filter entirely
			m.filter = ""
			m.filterInput.SetValue("")
			m.filterInput.Blur()
			m.mode = modeNormal
			m.cursor = 0
			return m, nil
		case tea.KeyEnter:
			// Confirm — keep filter active, return to navigable normal mode
			m.filter = strings.TrimSpace(m.filterInput.Value())
			m.filterInput.Blur()
			m.mode = modeNormal
			m.cursor = 0
			return m, nil
		default:
			var cmd tea.Cmd
			m.filterInput, cmd = m.filterInput.Update(msg)
			m.filter = m.filterInput.Value()
			m.cursor = 0
			return m, cmd
		}
	}

	// Esc in normal mode: clear filter first, then selection on second press
	if msg.String() == "esc" {
		if m.filter != "" {
			m.filter = ""
			m.filterInput.SetValue("")
			m.cursor = 0
			return m, nil
		}
		m.selected = nil
		return m, nil
	}

	// Clear load error on any key
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
			return m, m.reloadCmd()
		}

	case key.Matches(msg, m.keys.Down):
		if m.cursor < len(m.entries)-1 {
			m.cursor++
		} else if m.offset+len(m.entries) < m.total {
			m.offset += pageSize
			m.cursor = 0
			m.loading = true
			return m, m.reloadCmd()
		}

	case key.Matches(msg, m.keys.Enter):
		if len(m.entries) > 0 && m.entries[m.cursor].isDir() {
			m.path = m.entries[m.cursor].resource.Path
			m.offset = 0
			m.loading = true
			return m, m.reloadCmd()
		}

	case key.Matches(msg, m.keys.Back):
		if parent := parentPath(m.path); parent != m.path {
			m.path = parent
			m.offset = 0
			m.loading = true
			return m, m.reloadCmd()
		}

	case key.Matches(msg, m.keys.Sort):
		m.sort = nextSort(m.sort)
		m.offset = 0
		m.cursorAfterLoad = 0
		m.loading = true
		return m, m.reloadCmd()

	case key.Matches(msg, m.keys.Filter):
		m.filterInput.SetValue(m.filter)
		m.filterInput.Focus()
		m.filterInput.CursorEnd()
		m.mode = modeFilter
		return m, textinput.Blink

	case key.Matches(msg, m.keys.Refresh):
		m.filter = ""
		m.filterInput.SetValue("")
		m.loading = true
		return m, m.reloadCmd()

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
		p := m.entries[m.cursor].resource.Path
		m.selected[p] = !m.selected[p]
		if !m.selected[p] {
			delete(m.selected, p)
		}
		if m.cursor < len(m.entries)-1 {
			m.cursor++
		} else if m.offset+len(m.entries) < m.total {
			m.offset += pageSize
			m.cursorAfterLoad = 0
			m.loading = true
			return m, m.reloadCmd()
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
			return m.showMessage(iconErr+" No public URL — press p to publish first", true), nil
		}
		return m, cmdCopyToClipboard(url)
	}

	return m, nil
}
