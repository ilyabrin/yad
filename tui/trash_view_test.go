package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/ilyabrin/disk"
)

func TestTrashRowsFitTheWindow(t *testing.T) {
	for _, width := range []int{80, 120, 175} {
		m := newTestTrash()
		m.width, m.height = width, 20
		m.items = []*disk.TrashResource{{
			Resource:   disk.Resource{Path: "trash:/blurry-photo.jpg_1", Name: "blurry-photo.jpg", Type: "file", Size: 3_201_455},
			OriginPath: "disk:/Photos/blurry-photo.jpg",
			Deleted:    "2026-10-05T18:02:00+00:00",
		}}

		view := m.View()
		if got := strings.Count(view, "\n") + 1; got != m.height {
			t.Errorf("width %d: view has %d lines, want %d; a wrapped row pushes the title off screen", width, got, m.height)
		}
		for i, line := range strings.Split(view, "\n") {
			if w := ansi.StringWidth(line); w > width {
				t.Errorf("width %d: line %d is %d cells wide", width, i, w)
			}
		}
	}
}
