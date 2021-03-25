package ui

import (
	"unicode"
	"unicode/utf8"
)

// QuickCharInString is used for finding the "quick char" in a string. The rune
// is always made lowercase. A rune of value zero is returned if the index was
// less than zero, or greater or equal to, the number of runes in s.
func QuickCharInString(s string, idx int) rune {
	if idx < 0 {
		return 0
	}

	var runeIdx int

	bytes := []byte(s)
	for i := 0; i < len(bytes); runeIdx++ { // i is a byte index
		r, size := utf8.DecodeRune(bytes[i:])
		if runeIdx == idx {
			return unicode.ToLower(r)
		}
		i += size
	}
	return 0
}

// Max returns the larger integer.
func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Min returns the smaller integer.
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Clamp keeps `v` within `a` and `b` numerically. `a` must be smaller than `b`.
// Returns clamped `v`.
func Clamp(v, a, b int) int {
	return Max(a, Min(v, b))
}
