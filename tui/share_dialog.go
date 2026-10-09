package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/ilyabrin/disk"
)

// linkLifetimes are the choices for how long a new public link works. The
// first one, forever, is the default and means no expiry.
var linkLifetimes = []struct {
	label string
	d     time.Duration
}{
	{"forever", 0},
	{"1 day", 24 * time.Hour},
	{"7 days", 7 * 24 * time.Hour},
	{"30 days", 30 * 24 * time.Hour},
}

// ShareDialog asks how to protect a link before a file is published: how
// long the link works and an optional password. Enter with nothing changed
// publishes an ordinary link, as before the dialog existed.
type ShareDialog struct {
	name     string // the file or folder being shared
	lifetime int    // index into linkLifetimes
	password textinput.Model
	focus    int // 0: lifetime, 1: password
}

// NewShareDialog builds the dialog for the resource called name.
func NewShareDialog(name string) ShareDialog {
	pw := textinput.New()
	pw.Placeholder = "none"
	pw.EchoMode = textinput.EchoPassword
	pw.EchoCharacter = '•'
	pw.Width = 24
	return ShareDialog{name: name, password: pw}
}

// Update handles a key. It returns the dialog and whether it was submitted
// or cancelled.
func (d ShareDialog) Update(msg tea.KeyMsg) (ShareDialog, bool, bool) {
	switch msg.Type {
	case tea.KeyEnter:
		return d, true, false
	case tea.KeyEsc:
		return d, false, true
	case tea.KeyTab, tea.KeyShiftTab, tea.KeyUp, tea.KeyDown:
		d.focus = 1 - d.focus
		if d.focus == 1 {
			d.password.Focus()
		} else {
			d.password.Blur()
		}
		return d, false, false
	}

	if d.focus == 0 {
		switch msg.String() {
		case "left", "h":
			d.lifetime = (d.lifetime + len(linkLifetimes) - 1) % len(linkLifetimes)
		case "right", "l", " ":
			d.lifetime = (d.lifetime + 1) % len(linkLifetimes)
		}
		return d, false, false
	}

	// textinput.Update never returns a non-nil Cmd for key messages.
	d.password, _ = d.password.Update(msg)
	return d, false, false
}

// Settings turns the choices into link settings, or nil for an ordinary
// link. now is when the link is created, which the expiry counts from.
func (d ShareDialog) Settings(now time.Time) *disk.PublicSettings {
	s := &disk.PublicSettings{Password: d.password.Value()}
	if lt := linkLifetimes[d.lifetime].d; lt > 0 {
		s.ExpiresAt = now.Add(lt)
	}
	if s.Password == "" && s.ExpiresAt.IsZero() {
		return nil
	}
	return s
}

// protectionSummary describes link settings for the screen, without ever
// showing the password itself.
func protectionSummary(s *disk.PublicSettings) string {
	if s == nil {
		return ""
	}
	var parts []string
	if s.Password != "" {
		parts = append(parts, "password")
	}
	if !s.ExpiresAt.IsZero() {
		parts = append(parts, "expires "+s.ExpiresAt.Format("2 Jan 15:04"))
	}
	return "🔒 " + strings.Join(parts, " · ")
}

func (d ShareDialog) View(width int) string {
	label := func(text string, focused bool) string {
		if focused {
			return StyleStatusKey.Render(text)
		}
		return StyleMuted(text)
	}

	var b strings.Builder
	b.WriteString(StyleDialogTitle.Render("⇡ Share " + truncateRight(d.name, 40)))
	b.WriteString("\n\n")

	b.WriteString(label("Link works  ", d.focus == 0))
	b.WriteString("‹ " + linkLifetimes[d.lifetime].label + " ›")
	b.WriteString("\n")
	if d.focus == 0 {
		b.WriteString(StyleMuted("            ←/→ to change"))
	}
	b.WriteString("\n\n")

	b.WriteString(label("Password    ", d.focus == 1))
	b.WriteString(d.password.View())
	b.WriteString("\n")
	b.WriteString(StyleMuted("            optional; anyone opening the link must enter it"))
	b.WriteString("\n\n")

	b.WriteString(StyleStatusKey.Render("↵") + " share   " +
		StyleStatusKey.Render("tab") + " next field   " +
		StyleMuted("Esc cancel"))

	// As compact as the link dialog that follows it, but never wider than
	// the window.
	return StyleDialog.Width(min(shareDialogWidth, dialogWidth(width))).Render(b.String())
}

const shareDialogWidth = 64
