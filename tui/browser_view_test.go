package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestBrowserViewFitsTheWindow(t *testing.T) {
	for _, width := range []int{40, 80, 120, 200} {
		for _, filter := range []string{"", "file"} {
			m := newBrowserWithEntries(makeEntries(3))
			m.width, m.height = width, 20
			m.filter = filter
			m.selected = map[string]bool{m.entries[0].resource.Path: true}

			view := m.View()
			if got := strings.Count(view, "\n") + 1; got != m.height {
				t.Errorf("width %d, filter %q: view has %d lines, want %d; a wrapped status bar pushes the title off screen",
					width, filter, got, m.height)
			}
			for i, line := range strings.Split(view, "\n") {
				if w := ansi.StringWidth(line); w > width {
					t.Errorf("width %d, filter %q: line %d is %d cells wide", width, filter, i, w)
				}
			}
		}
	}
}

func TestBrowserStatusBarKeepsQuitWhenItShrinks(t *testing.T) {
	m := newBrowserWithEntries(makeEntries(3))
	m.width, m.height = 70, 20

	bar := ansi.Strip(m.viewStatusBar())
	if !strings.Contains(bar, "q quit") {
		t.Errorf("status bar %q lost the quit hint", bar)
	}
	if strings.Contains(bar, "t trash") {
		t.Errorf("status bar %q should have dropped later hints first", bar)
	}
}
