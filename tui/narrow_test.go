package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/ilyabrin/disk"
)

// A line wider than the window wraps and pushes the title off the top.
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

func TestNarrowWindows(t *testing.T) {
	for _, width := range []int{40, 60, 80, 120} {
		tr := newTestTrash()
		tr.width, tr.height = width, 20
		tr.items = []*disk.TrashResource{{
			Resource:   disk.Resource{Path: "trash:/a_1", Name: "quarterly report.pdf", Type: "file", Size: 1 << 20},
			OriginPath: "disk:/Documents/quarterly report.pdf",
			Deleted:    "2026-10-05T18:02:00+00:00",
		}}
		checkFits(t, "trash", tr.View(), width, 20)
		checkFits(t, "confirm dialog", NewConfirmDialog("Delete", "Delete it?").View(width), width, 0)
		checkFits(t, "share dialog", NewShareDialog("holiday 2026.jpg").View(width), width, 0)
	}
}
