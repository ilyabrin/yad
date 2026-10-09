package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/ilyabrin/disk"
	"github.com/ilyabrin/yad/internal/i18n"
)

// linkLifetimes are the choices for how long a new public link works. The
// first one, forever, is the default and means no expiry.
var linkLifetimes = []time.Duration{0, 24 * time.Hour, 7 * 24 * time.Hour, 30 * 24 * time.Hour}

// lifetimeLabel names a choice of linkLifetimes in the current language.
func lifetimeLabel(d time.Duration) string {
	if d == 0 {
		return i18n.T("share.forever")
	}
	days := int(d / (24 * time.Hour))
	return i18n.N("share.days", days)
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
	pw.Placeholder = i18n.T("share.none")
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
	if lt := linkLifetimes[d.lifetime]; lt > 0 {
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
		parts = append(parts, i18n.T("share.protected_password"))
	}
	if !s.ExpiresAt.IsZero() {
		parts = append(parts, i18n.T("share.expires")+s.ExpiresAt.Format(i18n.T("share.date_format")))
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
	b.WriteString(StyleDialogTitle.Render(i18n.T("share.title") + truncateRight(d.name, 40)))
	b.WriteString("\n\n")

	b.WriteString(label(pad(i18n.T("share.link_works")), d.focus == 0))
	b.WriteString("‹ " + lifetimeLabel(linkLifetimes[d.lifetime]) + " ›")
	b.WriteString("\n")
	if d.focus == 0 {
		b.WriteString(StyleMuted(pad("") + i18n.T("share.to_change")))
	}
	b.WriteString("\n\n")

	b.WriteString(label(pad(i18n.T("share.password_label")), d.focus == 1))
	b.WriteString(d.password.View())
	b.WriteString("\n")
	b.WriteString(StyleMuted(pad("") + i18n.T("share.optional_anyone_opening_the_link")))
	b.WriteString("\n\n")

	b.WriteString(StyleStatusKey.Render("↵") + i18n.T("hint.share") +
		StyleStatusKey.Render("tab") + i18n.T("hint.next_field") +
		StyleMuted(i18n.T("hint.esc_cancel")))

	// As compact as the link dialog that follows it, but never wider than
	// the window.
	return StyleDialog.Width(min(shareDialogWidth, dialogWidth(width))).Render(b.String())
}

const shareDialogWidth = 64

// pad makes field labels one width, so the values line up in either
// language.
func pad(label string) string {
	const width = 13
	if n := width - ansi.StringWidth(label); n > 0 {
		return label + strings.Repeat(" ", n)
	}
	return label + " "
}
