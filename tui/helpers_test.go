package tui

import "testing"

func TestBoolStr(t *testing.T) {
	if got := boolStr(true); got != "yes" {
		t.Errorf("boolStr(true) = %q, want \"yes\"", got)
	}
	if got := boolStr(false); got != "no" {
		t.Errorf("boolStr(false) = %q, want \"no\"", got)
	}
}

func TestPadToWidth(t *testing.T) {
	tests := []struct {
		name  string
		input string
		width int
		want  string
	}{
		{"zero width returns empty", "hello", 0, ""},
		{"negative width returns empty", "hello", -1, ""},
		{"exact fit", "hi", 2, "hi"},
		{"shorter than width pads with spaces", "hi", 5, "hi   "},
		{"longer than width truncates", "hello world", 5, "hello"},
		{"empty string pads", "", 3, "   "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := padToWidth(tt.input, tt.width)
			if got != tt.want {
				t.Errorf("padToWidth(%q, %d) = %q, want %q", tt.input, tt.width, got, tt.want)
			}
		})
	}
}

func TestTruncateRight(t *testing.T) {
	tests := []struct {
		name  string
		input string
		width int
		want  string
	}{
		{"zero width", "hello", 0, ""},
		{"fits", "hi", 5, "hi"},
		{"exact fit", "hello", 5, "hello"},
		{"ascii truncated", "hello world", 5, "hell…"},
		{"cyrillic not cut mid-rune", "отчёт-2026.pdf", 6, "отчёт…"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := truncateRight(tt.input, tt.width); got != tt.want {
				t.Errorf("truncateRight(%q, %d) = %q, want %q", tt.input, tt.width, got, tt.want)
			}
		})
	}
}

func TestTruncateLeft(t *testing.T) {
	tests := []struct {
		name  string
		input string
		width int
		want  string
	}{
		{"zero width", "/a/b", 0, ""},
		{"fits", "/a/b", 10, "/a/b"},
		{"keeps tail", "/very/deep/path", 5, "…path"},
		{"cyrillic keeps tail intact", "/архив/документы", 7, "…ументы"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := truncateLeft(tt.input, tt.width); got != tt.want {
				t.Errorf("truncateLeft(%q, %d) = %q, want %q", tt.input, tt.width, got, tt.want)
			}
		})
	}
}
