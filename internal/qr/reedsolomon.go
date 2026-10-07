package qr

// QR Codes use Reed-Solomon codes over GF(2^8) with the reducing polynomial
// x^8 + x^4 + x^3 + x^2 + 1 (0x11D) and generator element 2.

// gfMul multiplies two field elements.
func gfMul(x, y byte) byte {
	z := 0
	for i := 7; i >= 0; i-- {
		z = (z << 1) ^ ((z >> 7) * 0x11D)
		z ^= int((y>>uint(i))&1) * int(x)
	}
	return byte(z)
}

// rsDivisor returns the generator polynomial of the given degree, with its
// leading coefficient of 1 omitted and coefficients from highest to lowest
// power. For degree 2 that is (x - 1)(x - 2) = x^2 + 3x + 2, i.e. {3, 2}.
func rsDivisor(degree int) []byte {
	result := make([]byte, degree)
	result[degree-1] = 1
	root := byte(1)
	for i := 0; i < degree; i++ {
		for j := range result {
			result[j] = gfMul(result[j], root)
			if j+1 < len(result) {
				result[j] ^= result[j+1]
			}
		}
		root = gfMul(root, 2)
	}
	return result
}

// rsRemainder returns the error correction codewords for data: the remainder
// of data * x^len(divisor) divided by the generator polynomial.
func rsRemainder(data, divisor []byte) []byte {
	result := make([]byte, len(divisor))
	for _, b := range data {
		factor := b ^ result[0]
		copy(result, result[1:])
		result[len(result)-1] = 0
		for i, coef := range divisor {
			result[i] ^= gfMul(coef, factor)
		}
	}
	return result
}
