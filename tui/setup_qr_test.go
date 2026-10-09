package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// setupAt builds the sign-in screen at a given terminal size.
func setupAt(t *testing.T, width, height int) SetupModel {
	t.Helper()
	m := NewSetupModel(nil)
	if m.step != stepShowURL {
		t.Fatalf("expected the sign-in link screen, got step %d", m.step)
	}
	if width > 0 {
		m, _ = m.Update(tea.WindowSizeMsg{Width: width, Height: height})
	}
	return m
}

func TestSetupBuildsAQRCodeForTheSignInLink(t *testing.T) {
	m := setupAt(t, 0, 0)
	if len(m.qrLines) == 0 {
		t.Fatal("the sign-in link should come with a QR code")
	}
	// version 10 is 57 modules, plus the quiet zone on both sides
	if w := ansi.StringWidth(m.qrLines[0]); w != 57+2*qrQuietZone {
		t.Errorf("QR code is %d cells wide, want %d", w, 57+2*qrQuietZone)
	}
}

func TestSetupShowsTheQRCodeWhenItFits(t *testing.T) {
	m := setupAt(t, 300, 80)
	view := ansi.Strip(m.View())

	if !strings.Contains(view, "Or scan this with your phone:") {
		t.Fatal("expected the QR caption in a large window")
	}
	for i, line := range m.qrLines {
		if !strings.Contains(view, line) {
			t.Fatalf("QR row %d is missing from the screen", i)
		}
	}
	if !strings.Contains(view, m.authURL) {
		t.Error("the link itself must stay on screen next to the QR code")
	}
}

func TestSetupOffersAHintWhenTheQRCodeDoesNotFit(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {300, 20}, {40, 200}} {
		m := setupAt(t, size[0], size[1])
		view := ansi.Strip(m.View())

		if !strings.Contains(view, "zoom out to see a QR code") {
			t.Errorf("%dx%d: expected a hint about the QR code", size[0], size[1])
		}
		if strings.Contains(view, "Or scan this with your phone:") {
			t.Errorf("%dx%d: a QR code that does not fit cannot be scanned and must not be drawn", size[0], size[1])
		}
		if !strings.Contains(view, m.authURL) {
			t.Errorf("%dx%d: the link must still be shown", size[0], size[1])
		}
	}
}

func TestSetupSaysNothingAboutQRBeforeItKnowsTheWindowSize(t *testing.T) {
	m := setupAt(t, 0, 0)
	// The link carries a random PKCE challenge, which now and then spells
	// "QR" or "scan" by chance, so look at the screen around it.
	view := strings.ReplaceAll(ansi.Strip(m.View()), m.authURL, "")
	if strings.Contains(view, "QR") || strings.Contains(view, "scan") {
		t.Error("without a window size there is nothing to decide yet")
	}
}

func TestSetupQRCodeFitsExactlyAtTheThreshold(t *testing.T) {
	m := setupAt(t, 300, 80)
	above := m.View()
	above = above[:strings.Index(above, "\n\n"+StyleMuted("Or scan"))]
	need := rowsOnScreen(above, 300) + 2 + len(m.qrLines)

	if fits := setupAt(t, 300, need); !strings.Contains(ansi.Strip(fits.View()), "Or scan this") {
		t.Errorf("a window of exactly %d rows should show the QR code", need)
	}
	if short := setupAt(t, 300, need-1); strings.Contains(ansi.Strip(short.View()), "Or scan this") {
		t.Errorf("a window of %d rows is one too short and should not", need-1)
	}
}

func TestRowsOnScreenCountsWrappedLines(t *testing.T) {
	cases := []struct {
		text  string
		width int
		want  int
	}{
		{"short", 80, 1},
		{strings.Repeat("x", 80), 80, 1},
		{strings.Repeat("x", 81), 80, 2},
		{strings.Repeat("x", 244), 80, 4},
		{"a\n\nb", 80, 3},
	}
	for _, tc := range cases {
		if got := rowsOnScreen(tc.text, tc.width); got != tc.want {
			t.Errorf("rowsOnScreen(%d chars, width %d) = %d, want %d", len(tc.text), tc.width, got, tc.want)
		}
	}
}
