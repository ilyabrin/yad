package tui

import "github.com/charmbracelet/x/ansi"

// ellipsis is appended (or prepended) when text is shortened to fit a column.
const ellipsis = "…"

// truncateRight shortens s to at most w display cells, appending an ellipsis
// when characters were dropped. It is grapheme- and width-aware, so Cyrillic,
// CJK and emoji filenames are never cut mid-rune.
func truncateRight(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if ansi.StringWidth(s) <= w {
		return s
	}
	return ansi.Truncate(s, w, ellipsis)
}

// truncateLeft shortens s to at most w display cells by removing characters
// from the *start*, prepending an ellipsis. Used for paths, where the deepest
// segment carries the most information.
func truncateLeft(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if ansi.StringWidth(s) <= w {
		return s
	}
	return ansi.TruncateLeft(s, ansi.StringWidth(s)-w+1, ellipsis)
}

// padToWidth returns the leading w display cells of s, padded with spaces when
// s is shorter. Unlike a byte/rune slice it accounts for ANSI escape sequences,
// so styled base rows keep their colours when a dialog is composited on top.
func padToWidth(s string, w int) string {
	if w <= 0 {
		return ""
	}
	s = ansi.Truncate(s, w, "")
	if pad := w - ansi.StringWidth(s); pad > 0 {
		s += spaces(pad)
	}
	return s
}

// spaces returns a run of n space characters.
func spaces(n int) string {
	if n <= 0 {
		return ""
	}
	b := make([]byte, n)
	for i := range b {
		b[i] = ' '
	}
	return string(b)
}
