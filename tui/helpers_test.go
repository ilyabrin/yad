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

func TestTruncateToWidth(t *testing.T) {
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
			got := truncateToWidth(tt.input, tt.width)
			if got != tt.want {
				t.Errorf("truncateToWidth(%q, %d) = %q, want %q", tt.input, tt.width, got, tt.want)
			}
		})
	}
}
