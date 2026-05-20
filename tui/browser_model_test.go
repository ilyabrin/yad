package tui

import "testing"

func TestParentPath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/", "/"},
		{"disk:/", "disk:/"},
		{"disk:/folder", "disk:/"},
		{"disk:/folder/sub", "disk:/folder"},
		{"disk:/a/b/c", "disk:/a/b"},
		{"/folder", "/"},
		{"/folder/sub", "/folder"},
	}
	for _, tt := range tests {
		if got := parentPath(tt.input); got != tt.want {
			t.Errorf("parentPath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestNextSort(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"name", "-name"},
		{"-name", "modified"},
		{"modified", "-modified"},
		{"-modified", "size"},
		{"size", "-size"},
		{"-size", "name"},      // wrap-around
		{"unknown", "name"},    // unknown value → start of cycle
	}
	for _, tt := range tests {
		if got := nextSort(tt.input); got != tt.want {
			t.Errorf("nextSort(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSortLabel(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"name", "name ↑"},
		{"-name", "name ↓"},
		{"modified", "date ↑"},
		{"-modified", "date ↓"},
		{"size", "size ↑"},
		{"-size", "size ↓"},
		{"other", "other"}, // default: return as-is
	}
	for _, tt := range tests {
		if got := sortLabel(tt.input); got != tt.want {
			t.Errorf("sortLabel(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
