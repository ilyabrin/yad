package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ilyabrin/disk"
)

// ConfirmDialog is a simple yes/no overlay.
type ConfirmDialog struct {
	Title   string
	Message string
	focused bool // true = Yes highlighted
}

func NewConfirmDialog(title, message string) ConfirmDialog {
	return ConfirmDialog{Title: title, Message: message, focused: false}
}

// Update handles key input. Returns (dialog, confirmed, done).
//   - done=true means the user made a choice (confirmed or cancelled)
//   - confirmed=true means they pressed Yes
func (d ConfirmDialog) Update(msg tea.KeyMsg) (ConfirmDialog, bool, bool) {
	switch msg.String() {
	case "left", "h", "tab":
		d.focused = !d.focused
	case "right", "l":
		d.focused = !d.focused
	case "y", "Y":
		return d, true, true
	case "n", "N", "esc", "q":
		return d, false, true
	case "enter":
		return d, d.focused, true
	}
	return d, false, false
}

func (d ConfirmDialog) View(width int) string {
	var b strings.Builder
	b.WriteString(StyleDialogTitle.Render(d.Title))
	b.WriteString("\n")
	b.WriteString(StyleError.Render(d.Message))
	b.WriteString("\n\n")

	noStyle := lipgloss.NewStyle().
		Padding(0, 3).
		Border(lipgloss.NormalBorder()).
		BorderForeground(colorMuted)
	yesStyle := lipgloss.NewStyle().
		Padding(0, 3).
		Border(lipgloss.NormalBorder()).
		BorderForeground(colorPrimary).
		Foreground(colorAccent).
		Bold(true)

	var no, yes string
	if d.focused {
		yes = yesStyle.Render("Yes")
		no = noStyle.Render("No")
	} else {
		yes = noStyle.Render("Yes")
		no = yesStyle.Render("No")
	}

	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Center, no, "  ", yes))
	b.WriteString("\n\n")
	b.WriteString(StyleMuted("←/→ select   ↵ confirm   Esc cancel"))

	return StyleDialog.Width(max(width-dialogMargin, dialogMinWidth)).Render(b.String())
}

// InputDialog is a single-line text input overlay.
type InputDialog struct {
	Title       string
	Description string
	input       textinput.Model
}

func NewInputDialog(title, description, placeholder string) InputDialog {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Focus()
	ti.Width = 50
	return InputDialog{
		Title:       title,
		Description: description,
		input:       ti,
	}
}

// Value returns the current input text.
func (d InputDialog) Value() string { return d.input.Value() }

// SetValue pre-fills the input (e.g. current filename for rename).
func (d *InputDialog) SetValue(v string) { d.input.SetValue(v) }

// Update handles key input. Returns (dialog, submitted, cancelled).
func (d InputDialog) Update(msg tea.KeyMsg) (InputDialog, bool, bool) {
	switch msg.Type {
	case tea.KeyEnter:
		return d, true, false
	case tea.KeyEsc:
		return d, false, true
	}
	// textinput.Update never returns a non-nil Cmd for key messages.
	d.input, _ = d.input.Update(msg)
	return d, false, false
}

func (d InputDialog) View(width int) string {
	var b strings.Builder
	b.WriteString(StyleDialogTitle.Render(d.Title))
	b.WriteString("\n")
	if d.Description != "" {
		b.WriteString(StyleMuted(d.Description))
		b.WriteString("\n\n")
	}
	b.WriteString(d.input.View())
	b.WriteString("\n\n")
	b.WriteString(StyleMuted("↵ confirm   Esc cancel"))

	return StyleDialog.Width(max(width-dialogMargin, dialogMinWidth)).Render(b.String())
}

// ProgressOverlay renders an upload/download progress bar.
type ProgressOverlay struct {
	Title      string
	Filename   string
	Current    int64
	Total      int64 // -1 if unknown
	Percentage float64
	Done       bool
	Err        error
}

func (p ProgressOverlay) View(width int) string {
	var b strings.Builder
	b.WriteString(StyleDialogTitle.Render(p.Title))
	b.WriteString("\n")
	b.WriteString(StyleFile.Render(p.Filename))
	b.WriteString("\n\n")

	barWidth := max(width-dialogMargin-8, 20)

	if p.Done {
		if p.Err != nil {
			b.WriteString(StyleError.Render(iconErr + " " + p.Err.Error()))
		} else {
			b.WriteString(StyleSuccess.Render(iconOK + " Done  " + disk.FormatFileSize(p.Current)))
		}
	} else {
		filled := int(float64(barWidth) * p.Percentage / 100)
		if filled < 0 {
			filled = 0
		}
		if filled > barWidth {
			filled = barWidth
		}
		bar := StyleProgressFull.Render(strings.Repeat(" ", filled)) +
			StyleProgressEmpty.Render(strings.Repeat(" ", barWidth-filled))

		if p.Total > 0 {
			fmt.Fprintf(&b, "%s / %s  %.0f%%\n",
				disk.FormatFileSize(p.Current),
				disk.FormatFileSize(p.Total),
				p.Percentage,
			)
		} else {
			b.WriteString(disk.FormatFileSize(p.Current) + "\n")
		}
		b.WriteString(bar)
	}

	return StyleDialog.Width(max(width-dialogMargin, dialogMinWidth)).Render(b.String())
}
