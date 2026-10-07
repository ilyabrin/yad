package qr

import (
	"bytes"
	"errors"
	"slices"
	"strings"
	"testing"
)

// The values below come from ISO/IEC 18004 and its published examples, not
// from this package, so they check the implementation against the standard
// rather than against itself.

func TestCapacityMatchesTheStandard(t *testing.T) {
	// Byte-mode capacities from the standard's capacity table.
	want := map[int][4]int{
		1:  {17, 14, 11, 7},
		10: {271, 213, 151, 119},
		21: {929, 711, 509, 403},
		40: {2953, 2331, 1663, 1273},
	}
	for version, perLevel := range want {
		for level, n := range perLevel {
			if got := capacity(version, Level(level)); got != n {
				t.Errorf("capacity(v%d, %c) = %d, want %d", version, "LMQH"[level], got, n)
			}
		}
	}
}

func TestReedSolomonKnownAnswer(t *testing.T) {
	// "HELLO WORLD" at 1-M, the worked example used throughout QR literature.
	data := []byte{32, 91, 11, 120, 209, 114, 220, 77, 67, 64, 236, 17, 236, 17, 236, 17}
	want := []byte{196, 35, 39, 119, 235, 215, 231, 226, 93, 23}
	if got := rsRemainder(data, rsDivisor(len(want))); !bytes.Equal(got, want) {
		t.Errorf("error correction = %v, want %v", got, want)
	}
}

func TestReedSolomonCodewordsAreDivisible(t *testing.T) {
	// A valid codeword, data followed by its remainder, has every syndrome
	// zero: it evaluates to zero at each root 2^0 .. 2^(degree-1).
	for _, degree := range []int{7, 10, 18, 26, 30} {
		data := []byte(strings.Repeat("yad sign-in ", 8))
		word := append(append([]byte(nil), data...), rsRemainder(data, rsDivisor(degree))...)
		root := byte(1)
		for i := 0; i < degree; i++ {
			var acc byte
			for _, c := range word {
				acc = gfMul(acc, root) ^ c
			}
			if acc != 0 {
				t.Errorf("degree %d: syndrome %d is %d, want 0", degree, i, acc)
			}
			root = gfMul(root, 2)
		}
	}
}

func TestFormatInfoMatchesTheStandard(t *testing.T) {
	// Mask 0 for each level, from the standard's format information table.
	want := map[Level]int{
		L: 0b111011111000100,
		M: 0b101010000010010,
		Q: 0b011010101011111,
		H: 0b001011010001001,
	}
	for level, bits := range want {
		if got := formatInfo(level, 0); got != bits {
			t.Errorf("formatInfo(%c, 0) = %015b, want %015b", "LMQH"[level], got, bits)
		}
	}
}

func TestVersionInfoMatchesTheStandard(t *testing.T) {
	want := map[int]int{7: 0x07C94, 21: 0x15683, 40: 0x28C69}
	for version, bits := range want {
		if got := versionInfo(version); got != bits {
			t.Errorf("versionInfo(%d) = %#05x, want %#05x", version, got, bits)
		}
	}
}

func TestAlignmentPositionsMatchTheStandard(t *testing.T) {
	want := map[int][]int{
		1:  nil,
		2:  {6, 18},
		7:  {6, 22, 38},
		10: {6, 28, 50},
		32: {6, 34, 60, 86, 112, 138},
		36: {6, 24, 50, 76, 102, 128, 154},
		40: {6, 30, 58, 86, 114, 142, 170},
	}
	for version, pos := range want {
		if got := alignmentPositions(version); !slices.Equal(got, pos) {
			t.Errorf("alignmentPositions(%d) = %v, want %v", version, got, pos)
		}
	}
}

// signInURL has the shape and length of the URL yad encodes on its sign-in
// screen, with a fixed stand-in for the per-attempt PKCE challenge.
const signInURL = "https://oauth.yandex.ru/authorize?response_type=code" +
	"&client_id=d82340a941d148c4890a3d70297d639f" +
	"&redirect_uri=https%3A%2F%2Foauth.yandex.ru%2Fverification_code" +
	"&code_challenge=E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM" +
	"&code_challenge_method=S256"

func TestSignInURLFitsVersion10(t *testing.T) {
	c, err := Encode(signInURL, L)
	if err != nil {
		t.Fatal(err)
	}
	if c.Version() != 10 || c.Size() != 57 {
		t.Errorf("got version %d, size %d; want version 10, size 57", c.Version(), c.Size())
	}
}

func TestEncodePicksTheSmallestVersion(t *testing.T) {
	// The versions where a rule changes: the length field widens after 9 and
	// 26, version information appears at 7, plus both ends of the range.
	// Every version and level was checked once against two independent
	// decoders; repeating all 160 here would only slow CI down.
	for _, level := range []Level{L, M, Q, H} {
		for _, version := range []int{1, 2, 6, 7, 9, 10, 26, 27, 40} {
			n := capacity(version, level)
			c, err := Encode(strings.Repeat("a", n), level)
			if err != nil {
				t.Fatalf("%c v%d: %v", "LMQH"[level], version, err)
			}
			if c.Version() != version {
				t.Errorf("%d bytes at %c: version %d, want %d", n, "LMQH"[level], c.Version(), version)
			}
		}
	}
}

func TestEncodeRejectsWhatDoesNotFit(t *testing.T) {
	if _, err := Encode(strings.Repeat("a", capacity(40, L)+1), L); !errors.Is(err, ErrTooLong) {
		t.Errorf("err = %v, want ErrTooLong", err)
	}
	if _, err := Encode("x", Level(7)); err == nil {
		t.Error("an unknown level should be rejected")
	}
}

func TestEncodeIsDeterministic(t *testing.T) {
	a, _ := Encode(signInURL, L)
	b, _ := Encode(signInURL, L)
	if !slices.Equal(a.modules, b.modules) {
		t.Error("the same input produced different symbols")
	}
}

func TestFunctionPatternsAreInPlace(t *testing.T) {
	c, err := Encode(signInURL, L)
	if err != nil {
		t.Fatal(err)
	}
	n := c.Size()

	// Each finder: dark 7x7 outline, light ring, dark 3x3 core.
	for _, corner := range [][2]int{{0, 0}, {n - 7, 0}, {0, n - 7}} {
		for dy := 0; dy < 7; dy++ {
			for dx := 0; dx < 7; dx++ {
				d := max(abs(dx-3), abs(dy-3))
				if want := d != 2; c.Dark(corner[0]+dx, corner[1]+dy) != want {
					t.Fatalf("finder at %v wrong at (%d,%d)", corner, dx, dy)
				}
			}
		}
	}

	// Timing patterns alternate between the finders.
	for i := 8; i < n-8; i++ {
		if c.Dark(i, 6) != (i%2 == 0) || c.Dark(6, i) != (i%2 == 0) {
			t.Fatalf("timing pattern wrong at %d", i)
		}
	}

	if !c.Dark(8, n-8) {
		t.Error("the dark module next to the lower-left finder is missing")
	}
}

func TestDarkIsLightOutsideTheSymbol(t *testing.T) {
	c, _ := Encode("yad", L)
	for _, p := range [][2]int{{-1, 0}, {0, -1}, {c.Size(), 0}, {0, c.Size()}} {
		if c.Dark(p[0], p[1]) {
			t.Errorf("Dark%v should be light: it is quiet zone", p)
		}
	}
}

func TestHalfBlocksRoundTrip(t *testing.T) {
	c, err := Encode(signInURL, L)
	if err != nil {
		t.Fatal(err)
	}
	for _, quiet := range []int{0, 2, 4} {
		lines := c.HalfBlocks(quiet)
		width := c.Size() + 2*quiet
		if want := (width + 1) / 2; len(lines) != want {
			t.Fatalf("quiet %d: %d lines, want %d", quiet, len(lines), want)
		}
		for row, line := range lines {
			cells := []rune(line)
			if len(cells) != width {
				t.Fatalf("quiet %d: line %d has %d cells, want %d", quiet, row, len(cells), width)
			}
			for col, r := range cells {
				x, top := col-quiet, 2*row-quiet
				wantTop, wantBottom := c.Dark(x, top), c.Dark(x, top+1)
				gotTop := r == '█' || r == '▀'
				gotBottom := r == '█' || r == '▄'
				if gotTop != wantTop || gotBottom != wantBottom {
					t.Fatalf("quiet %d: cell (%d,%d) is %q, modules say top=%v bottom=%v",
						quiet, col, row, r, wantTop, wantBottom)
				}
			}
		}
	}
}

func BenchmarkEncodeSignInURL(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, err := Encode(signInURL, L); err != nil {
			b.Fatal(err)
		}
	}
}
