package ui

import (
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
)

// DrawRect renders a filled box at `x` and `y`, of size `width` and `height`.
// Will not call `Show()`.
func DrawRect(s tcell.Screen, x, y, width, height int, char rune, style tcell.Style) {
	for col := x; col < x+width; col++ {
		for row := y; row < y+height; row++ {
			s.SetContent(col, row, char, nil, style)
		}
	}
}

// DrawStr will render each character of a string at `x` and `y`.
func DrawStr(s tcell.Screen, x, y int, str string, style tcell.Style) {
	runes := []rune(str)
	for idx, r := range runes {
		s.SetContent(x+idx, y, r, nil, style)
	}
}

// DrawQuickCharStr renders a string very similar to how DrawStr works, but stylizes the
// quick char (the rune at `quickCharIdx`) with an underline. Returned is the number of
// columns that were drawn to the screen.
func DrawQuickCharStr(s tcell.Screen, x, y int, str string, quickCharIdx int, style tcell.Style) int {
	var col int
	var runeIdx int

	bytes := []byte(str)
	for i := 0; i < len(bytes); runeIdx++ { // i is a byte index
		r, size := utf8.DecodeRune(bytes[i:])

		sty := style
		if runeIdx == quickCharIdx {
			sty = style.Underline(true)						
		}
		s.SetContent(x+col, y, r, nil, sty)

		i += size
		col++
	}
	return col // TODO: use mattn/runewidth
}

// DrawRectOutline draws only the outline of a rectangle, using `ul`, `ur`, `bl`, and `br`
// for the corner runes, and `hor` and `vert` for the horizontal and vertical runes, respectively.
func DrawRectOutline(s tcell.Screen, x, y, _width, _height int, ul, ur, bl, br, hor, vert rune, style tcell.Style) {
	width := x + _width - 1   // Length across
	height := y + _height - 1 // Length top-to-bottom

	// Horizontals and verticals
	for col := x + 1; col < width; col++ {
		s.SetContent(col, y, hor, nil, style)      // Top line
		s.SetContent(col, height, hor, nil, style) // Bottom line
	}
	for row := y + 1; row < height; row++ {
		s.SetContent(x, row, vert, nil, style)     // Left line
		s.SetContent(width, row, vert, nil, style) // Right line
	}
	// Corners
	s.SetContent(x, y, ul, nil, style)
	s.SetContent(width, y, ur, nil, style)
	s.SetContent(x, height, bl, nil, style)
	s.SetContent(width, height, br, nil, style)
}

// DrawRectOutlineDefault calls DrawRectOutline with the default edge runes.
func DrawRectOutlineDefault(s tcell.Screen, x, y, width, height int, style tcell.Style) {
	DrawRectOutline(s, x, y, width, height, '┌', '┐', '└', '┘', '─', '│', style)
}

// DrawWindow draws a window-like object at x and y as the top-left corner. This window
// has an optional title. The Theme values "WindowHeader" and "Window" are used.
func DrawWindow(s tcell.Screen, x, y, width, height int, title string, theme *Theme) {
	headerStyle := theme.GetOrDefault("WindowHeader")

	DrawRect(s, x, y, width, 1, ' ', headerStyle) // Draw header background
	DrawStr(s, x + width/2 - len(title)/2, y, title, headerStyle) // Draw header title

	DrawRect(s, x, y+1, width, height-1, ' ', theme.GetOrDefault("Window")) // Draw body
}

// TODO: add DrawShadow(x, y, width, height int)
