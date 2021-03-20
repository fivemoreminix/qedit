package ui

import "github.com/gdamore/tcell/v2"

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
// quick char (any rune after an underscore) with an underline. Returned is the number of
// columns that were drawn to the screen. This is useful to know the length of the string
// drawn, minus the underscore.
func DrawQuickCharStr(s tcell.Screen, x, y int, str string, style tcell.Style) int {
	runes := []rune(str)
	col := 0
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r == '_' && i+1 < len(runes) {
			i++
			sty := style.Underline(true)
						
			s.SetContent(x+col, y, runes[i], nil, sty)
		} else {
			s.SetContent(x+col, y, r, nil, style)
		}
		col++
	}
	return col
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

// TODO: add DrawShadow(x, y, width, height int)
// TODO: add DrawWindow(x, y, width, height int, style tcell.Style)
