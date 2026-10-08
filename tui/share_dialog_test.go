package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func keys(d ShareDialog, ks ...tea.KeyMsg) (ShareDialog, bool, bool) {
	var submitted, cancelled bool
	for _, k := range ks {
		d, submitted, cancelled = d.Update(k)
	}
	return d, submitted, cancelled
}

var (
	keyEnter = tea.KeyMsg{Type: tea.KeyEnter}
	keyEsc   = tea.KeyMsg{Type: tea.KeyEsc}
	keyTab   = tea.KeyMsg{Type: tea.KeyTab}
	keyRight = tea.KeyMsg{Type: tea.KeyRight}
	keyLeft  = tea.KeyMsg{Type: tea.KeyLeft}
)

func typed(s string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }

func TestShareDialogEnterAloneGivesAPlainLink(t *testing.T) {
	d, submitted, _ := keys(NewShareDialog("a.pdf"), keyEnter)
	if !submitted {
		t.Fatal("Enter must submit")
	}
	if s := d.Settings(time.Now()); s != nil {
		t.Errorf("settings = %+v, want nil for an ordinary link", s)
	}
}

func TestShareDialogLifetime(t *testing.T) {
	now := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	d, _, _ := keys(NewShareDialog("a.pdf"), keyRight, keyRight)
	s := d.Settings(now)
	if s == nil || !s.ExpiresAt.Equal(now.Add(7*24*time.Hour)) || s.Password != "" {
		t.Errorf("two steps right = %+v, want 7 days and no password", s)
	}
	// Left from the first choice wraps around to the last one.
	d, _, _ = keys(NewShareDialog("a.pdf"), keyLeft)
	if s := d.Settings(now); s == nil || !s.ExpiresAt.Equal(now.Add(30*24*time.Hour)) {
		t.Errorf("left from forever = %+v, want 30 days", s)
	}
}

func TestShareDialogPasswordIsHidden(t *testing.T) {
	d, _, _ := keys(NewShareDialog("a.pdf"), keyTab, typed("s3cret"))
	s := d.Settings(time.Now())
	if s == nil || s.Password != "s3cret" || !s.ExpiresAt.IsZero() {
		t.Fatalf("settings = %+v", s)
	}
	if view := ansi.Strip(d.View(100)); strings.Contains(view, "s3cret") {
		t.Error("the password is shown on screen")
	}
	if summary := protectionSummary(s); !strings.Contains(summary, "password") || strings.Contains(summary, "s3cret") {
		t.Errorf("summary = %q", summary)
	}
}

func TestShareDialogLetterKeysGoToThePassword(t *testing.T) {
	// "l" moves the lifetime only while that field has focus.
	d, _, _ := keys(NewShareDialog("a.pdf"), keyTab, typed("l"))
	if s := d.Settings(time.Now()); s == nil || s.Password != "l" || !s.ExpiresAt.IsZero() {
		t.Errorf("settings = %+v", s)
	}
}

func TestShareDialogEscCancels(t *testing.T) {
	if _, submitted, cancelled := keys(NewShareDialog("a.pdf"), keyEsc); submitted || !cancelled {
		t.Error("Esc must cancel")
	}
}

func TestPublishKeyOpensTheShareDialog(t *testing.T) {
	m := newBrowserWithEntries(makeEntries(2))
	m, cmd := pressKey(m, "p")
	if m.mode != modeShare || cmd != nil {
		t.Fatalf("mode = %v, cmd = %v; an unpublished file must open the dialog first", m.mode, cmd)
	}
	m, _ = pressSpecialKey(m, tea.KeyEsc)
	if m.mode != modeNormal {
		t.Errorf("Esc left mode %v", m.mode)
	}

	m.entries[0].resource.PublicURL = "https://yadi.sk/i/abc"
	m, _ = pressKey(m, "p")
	if m.mode != modePublicURL {
		t.Errorf("a published file must show its link, got mode %v", m.mode)
	}
}

func TestShareDialogSubmitPublishes(t *testing.T) {
	m := newBrowserWithEntries(makeEntries(1))
	m, _ = pressKey(m, "p")
	m, cmd := pressSpecialKey(m, tea.KeyEnter)
	if m.mode != modeNormal || cmd == nil {
		t.Errorf("mode = %v, cmd = %v; Enter must start publishing", m.mode, cmd)
	}
}
