package ui

import "unicode"

// QuickCharInString is used for finding the "quick char" in a string. A quick char
// suffixes a '_' (underscore). So basically, this function returns any rune after
// an underscore. The rune is always made lowercase. The bool returned is whether
// the rune was found.
func QuickCharInString(s string) (bool, rune) {
	runes := []rune(s)
	for i, r := range runes {
		if r == '_' {
			if i+1 < len(runes) {
				return true, unicode.ToLower(runes[i+1])
			} else {
				return false, ' '
			}
		}
	}
	return false, ' '
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
