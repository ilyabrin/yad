package qr

// Tables from ISO/IEC 18004, indexed [level][version]. Index 0 is unused so
// that versions can be used as indices directly.

// eccPerBlock is the number of error correction codewords in each block.
var eccPerBlock = [4][41]int{
	L: {-1, 7, 10, 15, 20, 26, 18, 20, 24, 30, 18, 20, 24, 26, 30, 22, 24, 28, 30, 28, 28, 28, 28, 30, 30, 26, 28, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30},
	M: {-1, 10, 16, 26, 18, 24, 16, 18, 22, 22, 26, 30, 22, 22, 24, 24, 28, 28, 26, 26, 26, 26, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28},
	Q: {-1, 13, 22, 18, 26, 18, 24, 18, 22, 20, 24, 28, 26, 24, 20, 30, 24, 28, 28, 26, 30, 28, 30, 30, 30, 30, 28, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30},
	H: {-1, 17, 28, 22, 16, 22, 28, 26, 26, 24, 28, 24, 28, 22, 24, 24, 30, 28, 28, 26, 28, 30, 24, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30},
}

// numBlocks is the number of error correction blocks the data is split into.
var numBlocks = [4][41]int{
	L: {-1, 1, 1, 1, 1, 1, 2, 2, 2, 2, 4, 4, 4, 4, 4, 6, 6, 6, 6, 7, 8, 8, 9, 9, 10, 12, 12, 12, 13, 14, 15, 16, 17, 18, 19, 19, 20, 21, 22, 24, 25},
	M: {-1, 1, 1, 1, 2, 2, 4, 4, 4, 5, 5, 5, 8, 9, 9, 10, 10, 11, 13, 14, 16, 17, 17, 18, 20, 21, 23, 25, 26, 28, 29, 31, 33, 35, 37, 38, 40, 43, 45, 47, 49},
	Q: {-1, 1, 1, 2, 2, 4, 4, 6, 6, 8, 8, 8, 10, 12, 16, 12, 17, 16, 18, 21, 20, 23, 23, 25, 27, 29, 34, 34, 35, 38, 40, 43, 45, 48, 51, 53, 56, 59, 62, 65, 68},
	H: {-1, 1, 1, 2, 4, 4, 4, 5, 6, 8, 8, 11, 11, 16, 16, 18, 16, 19, 21, 25, 25, 25, 34, 30, 32, 35, 37, 40, 42, 45, 48, 51, 54, 57, 60, 63, 66, 70, 74, 77, 81},
}

// rawDataModules is the number of modules left for data and error correction
// once the function patterns of a version are drawn.
func rawDataModules(version int) int {
	n := (16*version+128)*version + 64
	if version >= 2 {
		align := version/7 + 2
		n -= (25*align-10)*align - 55
		if version >= 7 {
			n -= 36
		}
	}
	return n
}

// dataCodewords is how many 8-bit data codewords a version holds at a level.
func dataCodewords(version int, level Level) int {
	return rawDataModules(version)/8 - eccPerBlock[level][version]*numBlocks[level][version]
}

// charCountBits is the width of the length field in byte mode.
func charCountBits(version int) int {
	if version <= 9 {
		return 8
	}
	return 16
}

// capacity is the most bytes that fit in a version at a level, in byte mode.
func capacity(version int, level Level) int {
	return (dataCodewords(version, level)*8 - 4 - charCountBits(version)) / 8
}

// alignmentPositions lists the row and column centres of the alignment
// patterns, ascending.
func alignmentPositions(version int) []int {
	if version == 1 {
		return nil
	}
	count := version/7 + 2
	step := (version*8 + count*3 + 5) / (count*4 - 4) * 2
	pos := make([]int, count)
	pos[0] = 6
	for i, p := count-1, version*4+17-7; i >= 1; i, p = i-1, p-step {
		pos[i] = p
	}
	return pos
}
