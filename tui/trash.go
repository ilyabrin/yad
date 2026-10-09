package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ilyabrin/disk"
	"github.com/ilyabrin/yad/internal/i18n"
)

// --- Messages ---------------------------------------------------------------

type trashLoadedMsg struct {
	items []*disk.TrashResource
	err   error
}

type trashRestoreDoneMsg struct{ err error }
type trashDeleteDoneMsg struct{ err error }
type trashEmptyDoneMsg struct{ err error }

// switchToBrowserMsg is sent when the user exits the trash screen.
type switchToBrowserMsg struct{}

// --- Key map ----------------------------------------------------------------

type trashKeyMap struct {
	Up      key.Binding
	Down    key.Binding
	Restore key.Binding
	Delete  key.Binding
	Empty   key.Binding
	Refresh key.Binding
	Back    key.Binding
}

func defaultTrashKeyMap() trashKeyMap {
	return trashKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Restore: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "restore"),
		),
		Delete: key.NewBinding(
			key.WithKeys("D"),
			key.WithHelp("D", "delete permanently"),
		),
		Empty: key.NewBinding(
			key.WithKeys("E"),
			key.WithHelp("E", "empty trash"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("R", "ctrl+r"),
			key.WithHelp("R", "refresh"),
		),
		Back: key.NewBinding(
			key.WithKeys("backspace", "left", "h", "q", "esc"),
			key.WithHelp("q/←", "back"),
		),
	}
}

// --- Trash mode -------------------------------------------------------------

type trashMode int

const (
	trashModeNormal trashMode = iota
	trashModeConfirmDelete
	trashModeConfirmEmpty
	trashModeMessage
)

// --- TrashModel -------------------------------------------------------------

// TrashModel is the trash management screen. It lists deleted items and
// allows restoring or permanently deleting them.
type TrashModel struct {
	client  *disk.Client
	keys    trashKeyMap
	spinner spinner.Model

	items  []*disk.TrashResource
	cursor int

	loading bool
	err     error

	mode           trashMode
	confirm        ConfirmDialog
	message        string
	messageIsError bool

	width  int
	height int
}

func (m *TrashModel) setClient(c *disk.Client) { m.client = c }

func NewTrashModel(client *disk.Client) TrashModel {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(colorPrimary)

	return TrashModel{
		client:  client,
		keys:    defaultTrashKeyMap(),
		spinner: sp,
		loading: true,
	}
}

func (m TrashModel) Init() tea.Cmd {
	return tea.Batch(m.loadTrash(), m.spinner.Tick)
}

func (m TrashModel) loadTrash() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeoutOp)
		defer cancel()

		list, err := m.client.ListTrashResources(ctx, "", 100, 0)
		if err != nil {
			return trashLoadedMsg{err: err}
		}
		return trashLoadedMsg{items: list.Items}
	}
}

// --- Update -----------------------------------------------------------------

func (m TrashModel) Update(msg tea.Msg) (TrashModel, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case trashLoadedMsg:
		m.loading = false
		if msg.err != nil {
			if isAuthError(msg.err) {
				return m, func() tea.Msg { return tryRefreshMsg{origErr: msg.err} }
			}
			if fe := asFatalErrorMsg(msg.err); fe != nil {
				return m, func() tea.Msg { return *fe }
			}
			m.err = msg.err
			return m, nil
		}
		m.items = msg.items
		if m.cursor >= len(m.items) {
			m.cursor = max(0, len(m.items)-1)
		}

	case trashRestoreDoneMsg:
		if msg.err != nil {
			return m.showMessage(iconErr+" "+msg.err.Error(), true), nil
		}
		m.loading = true
		return m, tea.Batch(m.loadTrash(), m.spinner.Tick)

	case trashDeleteDoneMsg:
		if msg.err != nil {
			return m.showMessage(iconErr+" "+msg.err.Error(), true), nil
		}
		m.loading = true
		return m, tea.Batch(m.loadTrash(), m.spinner.Tick)

	case trashEmptyDoneMsg:
		if msg.err != nil {
			return m.showMessage(iconErr+" "+msg.err.Error(), true), nil
		}
		m.items = nil
		m.cursor = 0
		return m.showMessage(iconOK+i18n.T("trash.trash_emptied"), false), nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m TrashModel) handleKey(msg tea.KeyMsg) (TrashModel, tea.Cmd) {
	switch m.mode {

	case trashModeMessage:
		m.mode = trashModeNormal
		m.message = ""
		return m, nil

	case trashModeConfirmDelete:
		newDlg, confirmed, done := m.confirm.Update(msg)
		m.confirm = newDlg
		if done {
			if confirmed && len(m.items) > 0 {
				target := m.items[m.cursor].Path
				m.mode = trashModeNormal
				return m, m.cmdDeleteFromTrash(target)
			}
			m.mode = trashModeNormal
		}
		return m, nil

	case trashModeConfirmEmpty:
		newDlg, confirmed, done := m.confirm.Update(msg)
		m.confirm = newDlg
		if done {
			if confirmed {
				m.mode = trashModeNormal
				return m, m.cmdEmptyTrash()
			}
			m.mode = trashModeNormal
		}
		return m, nil
	}

	// Normal mode
	if m.err != nil {
		m.err = nil
		return m, nil
	}
	if m.loading {
		return m, nil
	}

	switch {
	case key.Matches(msg, m.keys.Back):
		return m, func() tea.Msg { return switchToBrowserMsg{} }

	case key.Matches(msg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}

	case key.Matches(msg, m.keys.Down):
		if m.cursor < len(m.items)-1 {
			m.cursor++
		}

	case key.Matches(msg, m.keys.Restore):
		if len(m.items) == 0 {
			return m, nil
		}
		target := m.items[m.cursor].Path
		return m, m.cmdRestore(target)

	case key.Matches(msg, m.keys.Delete):
		if len(m.items) == 0 {
			return m, nil
		}
		name := m.items[m.cursor].Name
		m.confirm = NewConfirmDialog(i18n.T("trash.delete_permanently"), i18n.F("trash.permanently_delete", name))
		m.mode = trashModeConfirmDelete

	case key.Matches(msg, m.keys.Empty):
		m.confirm = NewConfirmDialog(i18n.T("trash.empty_trash"), i18n.T("trash.permanently_delete_all_items_in"))
		m.mode = trashModeConfirmEmpty

	case key.Matches(msg, m.keys.Refresh):
		m.loading = true
		return m, tea.Batch(m.loadTrash(), m.spinner.Tick)
	}

	return m, nil
}

// --- Commands ---------------------------------------------------------------

func (m TrashModel) cmdRestore(trashPath string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeoutOp)
		defer cancel()
		_, err := m.client.RestoreFromTrash(ctx, trashPath, false, "")
		return trashRestoreDoneMsg{err: err}
	}
}

func (m TrashModel) cmdDeleteFromTrash(trashPath string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeoutOp)
		defer cancel()
		err := m.client.EmptyTrash(ctx, trashPath, false)
		return trashDeleteDoneMsg{err: err}
	}
}

func (m TrashModel) cmdEmptyTrash() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeoutDelete)
		defer cancel()
		err := m.client.EmptyTrash(ctx, "", false)
		return trashEmptyDoneMsg{err: err}
	}
}

func (m TrashModel) showMessage(msg string, isError bool) TrashModel {
	m.mode = trashModeMessage
	m.message = msg
	m.messageIsError = isError
	return m
}

// --- View -------------------------------------------------------------------

func (m TrashModel) View() string {
	if m.width == 0 {
		return ""
	}

	base := m.viewBase()

	switch m.mode {
	case trashModeConfirmDelete, trashModeConfirmEmpty:
		return renderOverlay(base, m.confirm.View(m.width), m.width, m.height)
	case trashModeMessage:
		var content string
		if m.messageIsError {
			content = StyleError.Render(m.message)
		} else {
			content = StyleSuccess.Render(m.message)
		}
		overlay := StyleDialog.Render(content + "\n\n" + StyleMuted(i18n.T("common.press_any_key")))
		return renderOverlay(base, overlay, m.width, m.height)
	}

	return base
}

func (m TrashModel) viewBase() string {
	var b strings.Builder

	// Title bar
	title := StyleTitle.Render(i18n.T("trash.title"))
	count := StylePath.Render(" " + i18n.N("trash.items", len(m.items)))
	gap := m.width - lipgloss.Width(title) - lipgloss.Width(count)
	if gap < 0 {
		gap = 0
	}
	b.WriteString(title + count + spaces(gap))
	b.WriteByte('\n')

	contentHeight := m.height - 2
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

func (m TrashModel) viewLoading(height int) string {
	msg := m.spinner.View() + i18n.T("common.loading")
	lines := make([]string, height)
	for i := range lines {
		if i == height/2 {
			lines[i] = lipgloss.PlaceHorizontal(m.width, lipgloss.Center, msg)
		}
	}
	return strings.Join(lines, "\n")
}

func (m TrashModel) viewError(height int) string {
	lines := make([]string, height)
	mid := height / 2
	for i := range lines {
		switch i {
		case mid:
			lines[i] = lipgloss.PlaceHorizontal(m.width, lipgloss.Center, StyleError.Render(iconErr+" "+m.err.Error()))
		case mid + 1:
			lines[i] = lipgloss.PlaceHorizontal(m.width, lipgloss.Center, StyleMuted(i18n.T("common.press_any_key")))
		}
	}
	return strings.Join(lines, "\n")
}

func (m TrashModel) viewList(height int) string {
	if len(m.items) == 0 {
		lines := make([]string, height)
		lines[height/2] = lipgloss.PlaceHorizontal(m.width, lipgloss.Center, StyleMuted(i18n.T("trash.is_empty")))
		return strings.Join(lines, "\n")
	}

	visible := height
	if visible > len(m.items) {
		visible = len(m.items)
	}

	start := m.cursor - visible + 1
	if start < 0 {
		start = 0
	}
	if m.cursor < start {
		start = m.cursor
	}

	// the two-space gaps before the date and origin columns take 2 more cells
	// than colDateWidth already counts; miss them and every row wraps
	// Columns: padding and icon, name, size, then the deletion date and where
	// the item came from. In a narrow window the origin goes first, then the
	// date, so the name always keeps enough room.
	const (
		fixedCols  = 1 + 2 + colSizeWidth // left padding, icon, size
		dateCols   = 2 + colDateWidth - 1 // gap, date
		originCols = 2 + trashOriginWidth // gap, origin
	)
	showDate, showOrigin := true, true
	if m.width < fixedCols+colNameMinWidth+dateCols+originCols {
		showOrigin = false
	}
	if m.width < fixedCols+colNameMinWidth+dateCols {
		showDate = false
	}
	nameWidth := m.width - fixedCols
	if showDate {
		nameWidth -= dateCols
	}
	if showOrigin {
		nameWidth -= originCols
	}
	nameWidth = max(nameWidth, colNameMinWidth)
	originWidth := trashOriginWidth

	var rows []string
	for i := start; i < start+visible && i < len(m.items); i++ {
		item := m.items[i]
		selected := i == m.cursor

		name := truncateRight(item.Name, nameWidth)

		icon := "  "
		if item.Type == "dir" {
			icon = "  "
		}
		nameStyled := icon + StyleFile.Render(name)

		origin := item.OriginPath
		origin = truncateLeft(origin, originWidth)

		deleted := ""
		if item.Deleted != "" {
			t, err := time.Parse(time.RFC3339, item.Deleted)
			if err == nil {
				deleted = t.Format("2006-01-02 15:04")
			}
		}

		cells := []string{
			lipgloss.NewStyle().Width(nameWidth + 2).Render(nameStyled),
			StyleSize.Render(disk.FormatFileSize(item.Size)),
		}
		if showDate {
			cells = append(cells, "  ", StyleDate.Render(deleted))
		}
		if showOrigin {
			cells = append(cells, "  ", lipgloss.NewStyle().Foreground(colorMuted).Width(originWidth).Render(origin))
		}
		row := lipgloss.JoinHorizontal(lipgloss.Top, cells...)

		if selected {
			row = StyleItemSelected.Width(m.width).Render(row)
		} else {
			row = StyleItemNormal.Width(m.width).Render(row)
		}
		rows = append(rows, row)
	}

	for len(rows) < height {
		rows = append(rows, spaces(m.width))
	}
	return strings.Join(rows, "\n")
}

func (m TrashModel) viewStatusBar() string {
	left := ""
	if len(m.items) > 0 {
		left = fmt.Sprintf("%d/%d", m.cursor+1, len(m.items))
	}

	hints := []string{
		StyleStatusKey.Render("↑↓") + i18n.T("hint.move"),
		StyleStatusKey.Render("r") + i18n.T("hint.restore"),
		StyleStatusKey.Render("D") + i18n.T("hint.delete"),
		StyleStatusKey.Render("E") + i18n.T("hint.empty_trash"),
		StyleStatusKey.Render("q/←") + i18n.T("hint.back"),
	}
	return statusBar(m.width, left, hints)
}
