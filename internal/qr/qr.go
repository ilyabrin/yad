// Package qr encodes text as a QR Code, following ISO/IEC 18004.
//
// It covers only what yad needs: byte mode, any of the four error correction
// levels, and versions 1 to 40, with the mask chosen by the standard's
// penalty rules. There is no numeric, alphanumeric or kanji mode and no
// structured append, which keeps it small enough to read in one sitting.
package qr

import (
	"errors"
	"math"
)

// Level is the error correction level: how much of the symbol can be damaged
// before it stops decoding. Higher levels give larger symbols.
type Level int

const (
	L Level = iota // about 7% recoverable
	M              // about 15%
	Q              // about 25%
	H              // about 30%
)

// formatBits is the two-bit value the format information stores for a level.
func (l Level) formatBits() int {
	return [...]int{L: 1, M: 0, Q: 3, H: 2}[l]
}

// ErrTooLong is returned when the text does not fit even the largest version.
var ErrTooLong = errors.New("qr: text too long for a QR Code")

// Code is an encoded QR Code symbol.
type Code struct {
	version int
	size    int
	modules []bool // row-major, true is dark
}

// Version is the symbol version, from 1 to 40.
func (c *Code) Version() int { return c.version }

// Size is the width and height of the symbol in modules, without the quiet
// zone.
func (c *Code) Size() int { return c.size }

// Dark reports whether the module at column x, row y is dark. Coordinates
// outside the symbol are light, which is what the quiet zone around it needs.
func (c *Code) Dark(x, y int) bool {
	if x < 0 || y < 0 || x >= c.size || y >= c.size {
		return false
	}
	return c.modules[y*c.size+x]
}

// Encode encodes text in byte mode at the given level, using the smallest
// version it fits.
func Encode(text string, level Level) (*Code, error) {
	if level < L || level > H {
		return nil, errors.New("qr: unknown error correction level")
	}
	data := []byte(text)

	version := 0
	for v := 1; v <= 40; v++ {
		if len(data) <= capacity(v, level) {
			version = v
			break
		}
	}
	if version == 0 {
		return nil, ErrTooLong
	}

	codewords := addErrorCorrection(encodeData(data, version, level), version, level)

	m := newMatrix(version)
	m.drawFunctionPatterns()
	m.drawCodewords(codewords)

	best, bestScore := 0, math.MaxInt
	for mask := 0; mask < 8; mask++ {
		m.applyMask(mask)
		m.drawFormatBits(level, mask)
		if score := m.penalty(); score < bestScore {
			best, bestScore = mask, score
		}
		m.applyMask(mask) // masking is an XOR, so applying it again undoes it
	}
	m.applyMask(best)
	m.drawFormatBits(level, best)

	return &Code{version: version, size: m.size, modules: m.modules}, nil
}

// encodeData builds the data codewords: mode, length, the bytes themselves,
// a terminator, and padding up to the version's capacity.
func encodeData(data []byte, version int, level Level) []byte {
	var bits bitBuffer
	bits.append(0b0100, 4) // byte mode
	bits.append(len(data), charCountBits(version))
	for _, b := range data {
		bits.append(int(b), 8)
	}

	capacityBits := dataCodewords(version, level) * 8
	bits.append(0, min(4, capacityBits-bits.len()))
	bits.append(0, (8-bits.len()%8)%8)
	for pad := 0xEC; bits.len() < capacityBits; pad ^= 0xEC ^ 0x11 {
		bits.append(pad, 8)
	}
	return bits.bytes()
}

// addErrorCorrection splits the data into blocks, appends each block's error
// correction codewords and interleaves the result, as the standard requires.
func addErrorCorrection(data []byte, version int, level Level) []byte {
	blocks := numBlocks[level][version]
	eccLen := eccPerBlock[level][version]
	raw := rawDataModules(version) / 8
	shortBlocks := blocks - raw%blocks
	shortLen := raw / blocks

	divisor := rsDivisor(eccLen)
	all := make([][]byte, blocks)
	for i, k := 0, 0; i < blocks; i++ {
		n := shortLen - eccLen
		if i >= shortBlocks {
			n++
		}
		block := append([]byte(nil), data[k:k+n]...)
		k += n
		ecc := rsRemainder(block, divisor)
		if i < shortBlocks {
			block = append(block, 0) // placeholder so every block has the same length
		}
		all[i] = append(block, ecc...)
	}

	out := make([]byte, 0, raw)
	for i := range all[0] {
		for j, block := range all {
			// skip the placeholder byte of the short blocks
			if i != shortLen-eccLen || j >= shortBlocks {
				out = append(out, block[i])
			}
		}
	}
	return out
}

// matrix is a symbol under construction.
type matrix struct {
	size     int
	modules  []bool
	function []bool // true where a function pattern lives; masking skips these
}

func newMatrix(version int) *matrix {
	size := version*4 + 17
	return &matrix{
		size:     size,
		modules:  make([]bool, size*size),
		function: make([]bool, size*size),
	}
}

func (m *matrix) version() int { return (m.size - 17) / 4 }

// setFunction places a function module, which data and masking leave alone.
func (m *matrix) setFunction(x, y int, dark bool) {
	m.modules[y*m.size+x] = dark
	m.function[y*m.size+x] = true
}

func (m *matrix) drawFunctionPatterns() {
	for i := 0; i < m.size; i++ {
		m.setFunction(6, i, i%2 == 0)
		m.setFunction(i, 6, i%2 == 0)
	}

	m.drawFinder(3, 3)
	m.drawFinder(m.size-4, 3)
	m.drawFinder(3, m.size-4)

	pos := alignmentPositions(m.version())
	last := len(pos) - 1
	for i := range pos {
		for j := range pos {
			// the three corners are taken by finder patterns
			if (i == 0 && j == 0) || (i == 0 && j == last) || (i == last && j == 0) {
				continue
			}
			m.drawAlignment(pos[i], pos[j])
		}
	}

	m.drawFormatBits(L, 0) // reserve the area; the real bits come after masking
	m.drawVersion()
}

// drawFinder draws a finder pattern centred on (x, y), with its separator.
func (m *matrix) drawFinder(x, y int) {
	for dy := -4; dy <= 4; dy++ {
		for dx := -4; dx <= 4; dx++ {
			xx, yy := x+dx, y+dy
			if xx < 0 || yy < 0 || xx >= m.size || yy >= m.size {
				continue
			}
			d := max(abs(dx), abs(dy))
			m.setFunction(xx, yy, d != 2 && d != 4)
		}
	}
}

// drawAlignment draws an alignment pattern centred on (x, y).
func (m *matrix) drawAlignment(x, y int) {
	for dy := -2; dy <= 2; dy++ {
		for dx := -2; dx <= 2; dx++ {
			m.setFunction(x+dx, y+dy, max(abs(dx), abs(dy)) != 1)
		}
	}
}

// drawFormatBits writes both copies of the format information: the level and
// mask, protected by a BCH code.
func (m *matrix) drawFormatBits(level Level, mask int) {
	bits := formatInfo(level, mask)

	for i := 0; i <= 5; i++ {
		m.setFunction(8, i, bit(bits, i))
	}
	m.setFunction(8, 7, bit(bits, 6))
	m.setFunction(8, 8, bit(bits, 7))
	m.setFunction(7, 8, bit(bits, 8))
	for i := 9; i < 15; i++ {
		m.setFunction(14-i, 8, bit(bits, i))
	}

	for i := 0; i < 8; i++ {
		m.setFunction(m.size-1-i, 8, bit(bits, i))
	}
	for i := 8; i < 15; i++ {
		m.setFunction(8, m.size-15+i, bit(bits, i))
	}
	m.setFunction(8, m.size-8, true) // the dark module, always set
}

// drawVersion writes the two copies of the version information, which only
// versions 7 and up carry.
func (m *matrix) drawVersion() {
	v := m.version()
	if v < 7 {
		return
	}
	bits := versionInfo(v)
	for i := 0; i < 18; i++ {
		a, b := m.size-11+i%3, i/3
		m.setFunction(a, b, bit(bits, i))
		m.setFunction(b, a, bit(bits, i))
	}
}

// formatInfo is the 15-bit format information: the level and mask, a 10-bit
// BCH code, and the standard's fixed XOR pattern.
func formatInfo(level Level, mask int) int {
	data := level.formatBits()<<3 | mask
	rem := data
	for i := 0; i < 10; i++ {
		rem = (rem << 1) ^ ((rem >> 9) * 0x537)
	}
	return (data<<10 | rem) ^ 0x5412
}

// versionInfo is the 18-bit version information: the version and a 12-bit
// BCH code.
func versionInfo(version int) int {
	rem := version
	for i := 0; i < 12; i++ {
		rem = (rem << 1) ^ ((rem >> 11) * 0x1F25)
	}
	return version<<12 | rem
}

// drawCodewords places the codewords in the zigzag order the standard
// defines: two-column strips from the right, alternating up and down.
func (m *matrix) drawCodewords(data []byte) {
	i := 0
	for right := m.size - 1; right >= 1; right -= 2 {
		if right == 6 {
			right = 5 // the vertical timing pattern sits in column 6
		}
		upward := ((right + 1) & 2) == 0
		for vert := 0; vert < m.size; vert++ {
			y := vert
			if upward {
				y = m.size - 1 - vert
			}
			for j := 0; j < 2; j++ {
				x := right - j
				if m.function[y*m.size+x] || i >= len(data)*8 {
					continue
				}
				m.modules[y*m.size+x] = data[i>>3]>>(7-uint(i&7))&1 == 1
				i++
			}
		}
	}
}

// applyMask flips the data modules selected by one of the eight masks.
func (m *matrix) applyMask(mask int) {
	for y := 0; y < m.size; y++ {
		for x := 0; x < m.size; x++ {
			if !m.function[y*m.size+x] && maskHit(mask, x, y) {
				m.modules[y*m.size+x] = !m.modules[y*m.size+x]
			}
		}
	}
}

func maskHit(mask, x, y int) bool {
	switch mask {
	case 0:
		return (x+y)%2 == 0
	case 1:
		return y%2 == 0
	case 2:
		return x%3 == 0
	case 3:
		return (x+y)%3 == 0
	case 4:
		return (x/3+y/2)%2 == 0
	case 5:
		return x*y%2+x*y%3 == 0
	case 6:
		return (x*y%2+x*y%3)%2 == 0
	default:
		return ((x+y)%2+x*y%3)%2 == 0
	}
}

// Penalty weights from the standard.
const (
	penaltyRun     = 3  // a row or column run of five or more same-colour modules
	penaltyBlock   = 3  // each 2x2 block of one colour
	penaltyFinder  = 40 // each pattern that could be mistaken for a finder
	penaltyBalance = 10 // each 5% the dark share strays from 50%
)

// penalty scores how hard the symbol would be to scan; the mask with the
// lowest score wins.
func (m *matrix) penalty() int {
	n := m.size
	score := 0

	line := make([]bool, n)
	for y := 0; y < n; y++ {
		copy(line, m.modules[y*n:(y+1)*n])
		score += linePenalty(line)
	}
	for x := 0; x < n; x++ {
		for y := 0; y < n; y++ {
			line[y] = m.modules[y*n+x]
		}
		score += linePenalty(line)
	}

	for y := 0; y+1 < n; y++ {
		row, next := m.modules[y*n:], m.modules[(y+1)*n:]
		for x := 0; x+1 < n; x++ {
			c := row[x]
			if c == row[x+1] && c == next[x] && c == next[x+1] {
				score += penaltyBlock
			}
		}
	}

	dark := 0
	for _, d := range m.modules {
		if d {
			dark++
		}
	}
	total := n * n // always odd, so the share is never exactly 50% and k >= 0
	k := (abs(dark*20-total*10)+total-1)/total - 1
	return score + k*penaltyBalance
}

// Finder-like patterns, read left to right as the bits of an 11-module window:
// dark-light-dark-dark-dark-light-dark with four light modules on one side.
const (
	finderLikeAfter  = 0b10111010000
	finderLikeBefore = 0b00001011101
)

// linePenalty scores one row or column for long runs and finder-like
// patterns, in a single pass.
func linePenalty(line []bool) int {
	score, run, window := 0, 0, 0
	for k, dark := range line {
		if k > 0 && dark == line[k-1] {
			run++
		} else {
			if run >= 5 {
				score += penaltyRun + run - 5
			}
			run = 1
		}

		window = (window << 1) & 0x7FF
		if dark {
			window |= 1
		}
		if k >= 10 && (window == finderLikeAfter || window == finderLikeBefore) {
			score += penaltyFinder
		}
	}
	if run >= 5 {
		score += penaltyRun + run - 5
	}
	return score
}

// bitBuffer accumulates a big-endian bit string.
type bitBuffer []bool

func (b *bitBuffer) append(value, n int) {
	for i := n - 1; i >= 0; i-- {
		*b = append(*b, (value>>uint(i))&1 == 1)
	}
}

func (b *bitBuffer) len() int { return len(*b) }

func (b *bitBuffer) bytes() []byte {
	out := make([]byte, (len(*b)+7)/8)
	for i, set := range *b {
		if set {
			out[i>>3] |= 1 << (7 - uint(i&7))
		}
	}
	return out
}

func bit(x, i int) bool { return (x>>uint(i))&1 == 1 }

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
