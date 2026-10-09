package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/ilyabrin/disk"
	"github.com/ilyabrin/yad/internal/i18n"
)

func (m BrowserModel) View() string {
	if m.width == 0 {
		return ""
	}

	base := m.viewBase()

	switch m.mode {
	case modeConfirmDelete, modeConfirmBulkDelete:
		return renderOverlay(base, m.confirm.View(m.width), m.width, m.height)
	case modeInputNewDir, modeInputRename,
		modeInputUpload, modeInputUploadName, modeInputUploadURL,
		modeInputDownload, modeInputDownloadDir:
		return renderOverlay(base, m.inputDlg.View(m.width), m.width, m.height)
	case modeUpload, modeDownload, modeConfirmQuit:
		if m.mode == modeConfirmQuit {
			return renderOverlay(base, m.confirm.View(m.width), m.width, m.height)
		}
		return renderOverlay(base, m.progress.View(m.width), m.width, m.height)
	case modeMessage:
		var msgView string
		if m.messageIsError {
			msgView = StyleDialog.Render(StyleError.Render(m.message) + "\n\n" + StyleMuted(i18n.T("common.press_any_key")))
		} else {
			msgView = StyleDialog.Render(StyleSuccess.Render(m.message) + "\n\n" + StyleMuted(i18n.T("common.press_any_key")))
		}
		return renderOverlay(base, msgView, m.width, m.height)
	case modeShare:
		return renderOverlay(base, m.shareDlg.View(m.width), m.width, m.height)
	case modePublicURL:
		urlStyle := lipgloss.NewStyle().Foreground(colorAccent)
		protection := ""
		if m.linkProtection != "" {
			protection = StyleMuted(m.linkProtection) + "\n\n"
		}
		content := StyleSuccess.Render(i18n.T("browser.public_link")) + "\n\n" +
			urlStyle.Render(m.publicURL) + "\n\n" + protection +
			StyleStatusKey.Render("c") + i18n.T("hint.copy") +
			StyleStatusKey.Render("o") + i18n.T("hint.open_link") +
			StyleStatusKey.Render("u") + i18n.T("hint.unpublish") +
			StyleMuted(i18n.T("browser.any_other_key_to_close"))
		return renderOverlay(base, StyleDialog.Render(content), m.width, m.height)
	case modeMetadata:
		if e, ok := m.currentEntry(); ok {
			return renderOverlay(base, m.viewMetadata(e), m.width, m.height)
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
	sortStr := StyleMuted("  " + sortLabel(m.sort))

	titleW := lipgloss.Width(title)
	sortW := lipgloss.Width(sortStr)
	// 2 = padding inside StylePath (1 left + 1 right)
	pathAvail := m.width - titleW - sortW - 2

	// Strip the "disk:/" prefix — redundant in a Yandex Disk context.
	display := strings.TrimPrefix(m.path, "disk:")
	if display == "" {
		display = "/"
	}

	// Truncate from the left so the deepest path segment is always visible.
	if pathAvail > 1 {
		display = truncateLeft(display, pathAvail)
	}

	pathStr := StylePath.Render(display)
	gap := max(m.width-titleW-lipgloss.Width(pathStr)-sortW, 0)
	return title + pathStr + spaces(gap) + sortStr
}

func (m BrowserModel) viewLoading(height int) string {
	msg := m.spinner.View() + i18n.T("common.loading")
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
			lines[i] = lipgloss.PlaceHorizontal(m.width, lipgloss.Center, StyleError.Render(iconErr+" "+m.err.Error()))
		case mid + 1:
			lines[i] = lipgloss.PlaceHorizontal(m.width, lipgloss.Center, StyleMuted(i18n.T("common.press_any_key")))
		}
	}
	return strings.Join(lines, "\n")
}

func (m BrowserModel) viewList(height int) string {
	entries := m.visibleEntries()

	if len(entries) == 0 {
		lines := make([]string, height)
		msg := i18n.T("browser.empty_directory")
		if m.filter != "" {
			msg = i18n.F("browser.no_matches_for", m.filter)
		}
		lines[height/2] = lipgloss.PlaceHorizontal(m.width, lipgloss.Center, StyleMuted(msg))
		return strings.Join(lines, "\n")
	}

	hasPrev := m.offset > 0
	hasNext := m.offset+len(m.entries) < m.total

	// Reserve one row each for the prev/next page indicators.
	listHeight := height
	if hasPrev {
		listHeight--
	}
	if hasNext {
		listHeight--
	}

	visible := min(listHeight, len(entries))
	start := max(m.cursor-visible+1, 0)
	start = min(start, m.cursor)
	nameWidth := max(m.width-colSizeWidth-colDateWidth-colPubMarkWidth-colRowPadding, colNameMinWidth)

	rows := make([]string, 0, height)
	for i := start; i < start+visible && i < len(entries); i++ {
		e := entries[i]
		selected := i == m.cursor

		name := truncateRight(e.resource.Name, nameWidth)

		mark := "  "
		if m.selected[e.resource.Path] {
			mark = StyleSuccess.Render(iconOK + " ")
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

	for len(rows) < listHeight {
		rows = append(rows, spaces(m.width))
	}

	// Wrap list rows with pagination indicators.
	remaining := m.total - m.offset - len(m.entries)
	if hasPrev {
		indicator := lipgloss.PlaceHorizontal(m.width, lipgloss.Center, StyleMuted(i18n.T("browser.previous_page")))
		rows = append([]string{indicator}, rows...)
	}
	if hasNext {
		label := i18n.F("browser.more_to_load", remaining)
		indicator := lipgloss.PlaceHorizontal(m.width, lipgloss.Center, StyleMuted(label))
		rows = append(rows, indicator)
	}

	for len(rows) < height {
		rows = append(rows, spaces(m.width))
	}
	return strings.Join(rows, "\n")
}

func (m BrowserModel) viewStatusBar() string {
	// Filter mode: show the search input instead of hints.
	if m.mode == modeFilter {
		prompt := StyleStatusKey.Render("/") + " " + m.filterInput.View()
		return StyleStatusBar.Width(m.width).Render(prompt)
	}

	left := ""
	entries := m.visibleEntries()
	if len(entries) > 0 {
		if m.filter != "" {
			left = i18n.N("browser.matches", len(entries))
		} else {
			left = fmt.Sprintf("%d/%d", m.cursor+1+m.offset, m.total)
		}
	}
	if n := len(m.selected); n > 0 {
		left += "  " + StyleSuccess.Render(i18n.N("browser.selected", n))
	}
	if m.filter != "" {
		left += "  " + StyleMuted(i18n.F("browser.filter_status", m.filter))
	}

	hints := []string{
		StyleStatusKey.Render("↑↓") + i18n.T("hint.move"),
		StyleStatusKey.Render("↵") + i18n.T("hint.open"),
		StyleStatusKey.Render("spc") + i18n.T("hint.select"),
		StyleStatusKey.Render("/") + i18n.T("hint.filter"),
		StyleStatusKey.Render("u") + i18n.T("hint.upload"),
		StyleStatusKey.Render("d") + i18n.T("hint.download"),
		StyleStatusKey.Render("n") + i18n.T("hint.mkdir"),
		StyleStatusKey.Render("r") + i18n.T("hint.rename"),
		StyleStatusKey.Render("D") + i18n.T("hint.delete"),
		StyleStatusKey.Render("s") + i18n.T("hint.sort"),
		StyleStatusKey.Render("p") + i18n.T("hint.publish"),
		StyleStatusKey.Render("t") + i18n.T("hint.trash"),
		StyleStatusKey.Render("q") + i18n.T("hint.quit"),
	}
	return statusBar(m.width, left, hints)
}

// statusBar lays out a bottom bar: left-aligned status, right-aligned key
// hints. A bar wider than the window wraps and pushes the title off screen,
// so hints are dropped from the end until it fits, always keeping the last
// one (quit, or back).
func statusBar(width int, left string, hints []string) string {
	room := width - 2 - lipgloss.Width(left) - 1
	right := strings.Join(hints, "  ")
	for len(hints) > 1 && lipgloss.Width(right) > room {
		hints = append(hints[:len(hints)-2:len(hints)-2], hints[len(hints)-1])
		right = strings.Join(hints, "  ")
	}
	if lipgloss.Width(right) > room {
		right = ""
	}

	gap := max(width-lipgloss.Width(left)-lipgloss.Width(right)-2, 1)
	bar := truncateRight(left+spaces(gap)+right, max(width-2, 0))
	return StyleStatusBar.Width(width).Render(bar)
}

// dialogWidth is the width of a dialog's content: roomy in a wide window,
// never narrower than dialogMinWidth unless the window itself is, and never
// so wide that the border runs off screen.
func dialogWidth(width int) int {
	return max(min(max(width-dialogMargin, dialogMinWidth), width-2), 0)
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

	kind := i18n.T("info.kind_file")
	if e.isDir() {
		kind = i18n.T("info.kind_dir")
	}

	var sb strings.Builder
	sb.WriteString(StyleTitle.Render(i18n.T("info.title")) + "\n\n")
	sb.WriteString(row(i18n.T("info.name"), r.Name))
	sb.WriteString(row(i18n.T("info.type"), kind))
	if !e.isDir() {
		sb.WriteString(row(i18n.T("info.size"), disk.FormatFileSize(r.Size)))
		sb.WriteString(row("MIME", r.MimeType))
		if r.MediaType != "" {
			sb.WriteString(row(i18n.T("info.media"), r.MediaType))
		}
	}
	sb.WriteString(row(i18n.T("info.created"), parseTime(r.Created)))
	sb.WriteString(row(i18n.T("info.modified"), parseTime(r.Modified)))
	if r.Md5 != "" {
		sb.WriteString(row("MD5", r.Md5))
	}
	if r.Sha256 != "" {
		sb.WriteString(row("SHA256", r.Sha256[:16]+"…"))
	}
	if r.PublicURL != "" {
		sb.WriteString(row(i18n.T("info.public"), r.PublicURL))
	}
	sb.WriteString("\n" + StyleMuted(i18n.T("info.any_key_to_close")))

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
		// padToWidth is ANSI-aware: it keeps the styled base row intact
		// instead of slicing through an escape sequence.
		baseLines[row] = padToWidth(baseLines[row], startX) + ol
	}

	return strings.Join(baseLines, "\n")
}

// StyleMuted renders text in the muted color.
func StyleMuted(s string) string {
	return lipgloss.NewStyle().Foreground(colorMuted).Render(s)
}
