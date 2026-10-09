package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/ilyabrin/disk"
	"github.com/ilyabrin/yad/internal/i18n"
)

// Every screen must fit the window in either language, and Russian text is
// longer than English: a line that wraps pushes the title off the top.
func checkFits(t *testing.T, what string, view string, width, height int) {
	t.Helper()
	lines := strings.Split(view, "\n")
	if height > 0 && len(lines) != height {
		t.Errorf("%s at width %d: %d lines, want %d", what, width, len(lines), height)
	}
	for i, line := range lines {
		if w := ansi.StringWidth(line); w > width {
			t.Errorf("%s at width %d: line %d is %d cells wide", what, width, i, w)
		}
	}
}

func TestScreensFitTheWindowInBothLanguages(t *testing.T) {
	defer i18n.Set(i18n.English)
	for _, lang := range []i18n.Lang{i18n.English, i18n.Russian} {
		i18n.Set(lang)
		checkScreens(t)
	}
}

func checkScreens(t *testing.T) {
	t.Helper()
	for _, width := range []int{40, 60, 80, 120} {
		for _, filter := range []string{"", "file"} {
			m := newBrowserWithEntries(makeEntries(3))
			m.width, m.height = width, 20
			m.filter = filter
			m.selected = map[string]bool{m.entries[0].resource.Path: true}
			checkFits(t, i18n.Current().String()+" browser", m.View(), width, 20)
		}

		tr := newTestTrash()
		tr.width, tr.height = width, 20
		tr.items = []*disk.TrashResource{{
			Resource:   disk.Resource{Path: "trash:/a_1", Name: "отчёт за квартал.pdf", Type: "file", Size: 1 << 20},
			OriginPath: "disk:/Документы/отчёт за квартал.pdf",
			Deleted:    "2026-10-05T18:02:00+00:00",
		}}
		checkFits(t, i18n.Current().String()+" trash", tr.View(), width, 20)

		checkFits(t, i18n.Current().String()+" share dialog", NewShareDialog("отпуск 2026.jpg").View(width), width, 0)
	}
}

func TestRussianStatusBarKeepsQuit(t *testing.T) {
	i18n.Set(i18n.Russian)
	defer i18n.Set(i18n.English)
	m := newBrowserWithEntries(makeEntries(3))
	m.width, m.height = 60, 20
	if bar := ansi.Strip(m.viewStatusBar()); !strings.Contains(bar, "q выход") {
		t.Errorf("status bar %q lost the quit hint", bar)
	}
}
