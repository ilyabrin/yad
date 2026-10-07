package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ilyabrin/disk"
)

// --- Messages ---------------------------------------------------------------

type diskInfoLoadedMsg struct {
	info *disk.Disk
	err  error
}

type switchToBrowserFromInfoMsg struct{}

// --- DiskInfoModel ----------------------------------------------------------

// DiskInfoModel is the disk usage overview screen.
type DiskInfoModel struct {
	client  *disk.Client
	spinner spinner.Model

	info    *disk.Disk
	loading bool
	err     error

	width  int
	height int
}

func (m *DiskInfoModel) setClient(c *disk.Client) { m.client = c }

func NewDiskInfoModel(client *disk.Client) DiskInfoModel {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(colorPrimary)

	return DiskInfoModel{
		client:  client,
		spinner: sp,
		loading: true,
	}
}

func (m DiskInfoModel) Init() tea.Cmd {
	return tea.Batch(m.loadInfo(), m.spinner.Tick)
}

func (m DiskInfoModel) loadInfo() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeoutMeta)
		defer cancel()
		info, err := m.client.DiskInfo(ctx)
		return diskInfoLoadedMsg{info: info, err: err}
	}
}

// --- Update -----------------------------------------------------------------

func (m DiskInfoModel) Update(msg tea.Msg) (DiskInfoModel, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case diskInfoLoadedMsg:
		m.loading = false
		if msg.err != nil {
			if isAuthError(msg.err) {
				return m, func() tea.Msg { return tryRefreshMsg{origErr: msg.err} }
			}
			if fe := asFatalErrorMsg(msg.err); fe != nil {
				return m, func() tea.Msg { return *fe }
			}
		}
		m.err = msg.err
		m.info = msg.info

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "backspace", "left", "h":
			return m, func() tea.Msg { return switchToBrowserFromInfoMsg{} }
		case "R", "ctrl+r":
			m.loading = true
			m.info = nil
			return m, tea.Batch(m.loadInfo(), m.spinner.Tick)
		}
	}

	return m, nil
}

// --- View -------------------------------------------------------------------

func (m DiskInfoModel) View() string {
	if m.width == 0 {
		return ""
	}

	var b strings.Builder

	// Title bar
	title := StyleTitle.Render("  YaD  ·  Disk Info")
	b.WriteString(title + strings.Repeat(" ", max(0, m.width-lipgloss.Width(title))))
	b.WriteByte('\n')

	if m.loading {
		msg := m.spinner.View() + " Loading…"
		for i := 0; i < m.height/2-1; i++ {
			b.WriteByte('\n')
		}
		b.WriteString(lipgloss.PlaceHorizontal(m.width, lipgloss.Center, msg))
	} else if m.err != nil {
		for i := 0; i < m.height/2-1; i++ {
			b.WriteByte('\n')
		}
		b.WriteString(lipgloss.PlaceHorizontal(m.width, lipgloss.Center, StyleError.Render(iconErr+" "+m.err.Error())))
	} else {
		b.WriteString(m.viewContent())
	}

	// Status bar
	b.WriteByte('\n')
	hints := StyleStatusKey.Render("R") + " refresh   " + StyleStatusKey.Render("q/←") + " back"
	b.WriteString(StyleStatusBar.Width(m.width).Render(hints))

	return b.String()
}

func (m DiskInfoModel) viewContent() string {
	if m.info == nil {
		return ""
	}

	labelStyle := lipgloss.NewStyle().
		Foreground(colorMuted).
		Width(22).
		Align(lipgloss.Right)

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#E5E7EB")).
		Bold(true).
		PaddingLeft(2)

	row := func(label, value string) string {
		return labelStyle.Render(label) + valueStyle.Render(value) + "\n"
	}

	used := m.info.UsedSpace
	total := m.info.TotalSpace
	trash := m.info.TrashSize

	usedPct := 0.0
	if total > 0 {
		usedPct = float64(used) / float64(total) * 100
	}

	barWidth := 40
	filled := int(float64(barWidth) * usedPct / 100)
	bar := StyleProgressFull.Render(strings.Repeat(" ", filled)) +
		StyleProgressEmpty.Render(strings.Repeat(" ", barWidth-filled))
	pctLabel := fmt.Sprintf("  %.1f%% used", usedPct)

	var content strings.Builder

	content.WriteString("\n")

	// User section
	if m.info.User != nil {
		content.WriteString(StyleDialogTitle.PaddingLeft(4).Render("User"))
		content.WriteString("\n")
		content.WriteString(row("Login", m.info.User.Login))
		content.WriteString(row("Display name", m.info.User.DisplayName))
		content.WriteString(row("Country", m.info.User.Country))
		content.WriteString(row("Paid account", boolStr(m.info.IsPaid)))
		content.WriteString("\n")
	}

	// Storage section
	content.WriteString(StyleDialogTitle.PaddingLeft(4).Render("Storage"))
	content.WriteString("\n")
	content.WriteString(row("Total", disk.FormatFileSize(total)))
	content.WriteString(row("Used", disk.FormatFileSize(used)+fmt.Sprintf("  (%.1f%%)", usedPct)))
	content.WriteString(row("Free", disk.FormatFileSize(total-used)))
	content.WriteString(row("Trash", disk.FormatFileSize(trash)))
	content.WriteString(row("Max file size", disk.FormatFileSize(m.info.MaxFileSize)))
	content.WriteString("\n")

	// Usage bar
	content.WriteString(strings.Repeat(" ", 24))
	content.WriteString(bar)
	content.WriteString(StyleMuted(pctLabel))
	content.WriteString("\n")

	// System folders section
	if m.info.SystemFolders != nil {
		content.WriteString("\n")
		content.WriteString(StyleDialogTitle.PaddingLeft(4).Render("System folders"))
		content.WriteString("\n")
		sf := m.info.SystemFolders
		for _, pair := range [][2]string{
			{"Downloads", sf.Downloads},
			{"Screenshots", sf.Screenshots},
			{"Applications", sf.Applications},
			{"Photostream", sf.Photostream},
			{"Social", sf.Social},
		} {
			if pair[1] != "" {
				content.WriteString(row(pair[0], pair[1]))
			}
		}
	}

	return content.String()
}

func boolStr(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
