package ui

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
