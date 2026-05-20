package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/ilyabrin/disk"
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
			StyleStatusKey.Render("o") + " open   " +
			StyleStatusKey.Render("u") + " unpublish   " +
			StyleMuted("any other key to close")
		return renderOverlay(base, StyleDialog.Render(content), m.width, m.height)
	case modeMetadata:
		if len(m.entries) > 0 {
			return renderOverlay(base, m.viewMetadata(m.entries[m.cursor]), m.width, m.height)
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
			lines[i] = lipgloss.PlaceHorizontal(m.width, lipgloss.Center, StyleError.Render(iconErr+" "+m.err.Error()))
		case mid + 1:
			lines[i] = lipgloss.PlaceHorizontal(m.width, lipgloss.Center, StyleMuted("Press any key to continue"))
		}
	}
	return strings.Join(lines, "\n")
}

func (m BrowserModel) viewList(height int) string {
	entries := m.visibleEntries()

	if len(entries) == 0 {
		lines := make([]string, height)
		msg := "(empty directory)"
		if m.filter != "" {
			msg = `no matches for "` + m.filter + `"`
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

		name := e.resource.Name
		if len(name) > nameWidth {
			name = name[:nameWidth-1] + "…"
		}

		mark := "  "
		if m.selected[e.resource.Path] {
			mark = StyleSuccess.Render(iconOK+" ")
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
		rows = append(rows, strings.Repeat(" ", m.width))
	}

	// Wrap list rows with pagination indicators.
	remaining := m.total - m.offset - len(m.entries)
	if hasPrev {
		indicator := lipgloss.PlaceHorizontal(m.width, lipgloss.Center, StyleMuted("▲ previous page"))
		rows = append([]string{indicator}, rows...)
	}
	if hasNext {
		label := fmt.Sprintf("▼  %d more  (↓ to load)", remaining)
		indicator := lipgloss.PlaceHorizontal(m.width, lipgloss.Center, StyleMuted(label))
		rows = append(rows, indicator)
	}

	for len(rows) < height {
		rows = append(rows, strings.Repeat(" ", m.width))
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
			left = fmt.Sprintf("%d matches", len(entries))
		} else {
			left = fmt.Sprintf("%d/%d", m.cursor+1+m.offset, m.total)
		}
	}
	if n := len(m.selected); n > 0 {
		left += "  " + StyleSuccess.Render(fmt.Sprintf("%d selected", n))
	}
	if m.filter != "" {
		left += "  " + StyleMuted(`filter: "`+m.filter+`"`)
	}

	hints := []string{
		StyleStatusKey.Render("↑↓") + " move",
		StyleStatusKey.Render("↵") + " open",
		StyleStatusKey.Render("spc") + " select",
		StyleStatusKey.Render("/") + " filter",
		StyleStatusKey.Render("u") + " upload",
		StyleStatusKey.Render("d") + " download",
		StyleStatusKey.Render("n") + " mkdir",
		StyleStatusKey.Render("r") + " rename",
		StyleStatusKey.Render("D") + " delete",
		StyleStatusKey.Render("s") + " sort",
		StyleStatusKey.Render("p") + " publish",
		StyleStatusKey.Render("t") + " trash",
		StyleStatusKey.Render("q") + " quit",
	}
	right := strings.Join(hints, "  ")

	gap := max(m.width-lipgloss.Width(left)-lipgloss.Width(right)-2, 1)
	bar := left + strings.Repeat(" ", gap) + right
	return StyleStatusBar.Width(m.width).Render(bar)
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
		blW := lipgloss.Width(bl)
		if blW < width {
			bl += strings.Repeat(" ", width-blW)
		}
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

// StyleMuted renders text in the muted color.
func StyleMuted(s string) string {
	return lipgloss.NewStyle().Foreground(colorMuted).Render(s)
}
