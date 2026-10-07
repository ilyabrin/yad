package qr

import "strings"

// HalfBlocks renders the symbol for a terminal, two modules per character
// cell stacked vertically. A cell is about twice as tall as it is wide, so
// this keeps each module square, which phone cameras need to read the code.
//
// Characters are drawn as foreground on background: "█" means both modules
// are dark, "▀" only the top, "▄" only the bottom, and a space neither. The
// caller must style the lines dark on light. quiet is the width, in modules,
// of the light margin to add on every side; the standard asks for 4.
func (c *Code) HalfBlocks(quiet int) []string {
	var lines []string
	for y := -quiet; y < c.size+quiet; y += 2 {
		var b strings.Builder
		for x := -quiet; x < c.size+quiet; x++ {
			top, bottom := c.Dark(x, y), c.Dark(x, y+1)
			switch {
			case top && bottom:
				b.WriteString("█")
			case top:
				b.WriteString("▀")
			case bottom:
				b.WriteString("▄")
			default:
				b.WriteByte(' ')
			}
		}
		lines = append(lines, b.String())
	}
	return lines
}
