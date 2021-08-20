package buffer

import (
	"math"
	"unicode"
)

// So why is the code for moving the cursor in the buffer package, and not in the
// TextEdit component? Well, it used to be, but it sucked that way. The cursor
// needs to have a reference to the buffer to know where lines end and how it can
// move. The buffer is the city, and the Cursor is the car.

type position struct {
	line int
	col  int
}

// A Selection represents a region of the buffer to be selected for text editing
// purposes. It is asserted that the start position is less than the end position.
// The start and end are inclusive. If the EndCol of a Region is one more than the
// last column of a line, then it points to the line delimiter at the end of that
// line. It is understood that as a Region spans multiple lines, those connecting
// line-delimiters are included in the selection, as well.
type Region struct {
	Start Cursor
	End   Cursor
}

func NewRegion(in *Buffer) Region {
	return Region{
		NewCursor(in),
		NewCursor(in),
	}
}

// A Cursor's functions emulate common cursor actions. To have a Cursor be
// automatically updated when the buffer has text prepended or appended -- one
// should register the Cursor with the Buffer's function `RegisterCursor()`
// which makes the Cursor "anchored" to the Buffer.
type Cursor struct {
	buffer  *Buffer
	prevCol int
	position
}

func NewCursor(in *Buffer) Cursor {
	return Cursor{
		buffer: in,
	}
}

func (c Cursor) Left() Cursor {
	if c.col == 0 && c.line != 0 { // If we are at the beginning of the current line...
		// Go to the end of the above line
		c.line--
		c.col = (*c.buffer).RunesInLine(c.line)
	} else {
		c.col = Max(c.col-1, 0)
	}
	return c
}

func (c Cursor) Right() Cursor {
	// If we are at the end of the current line,
	// and not at the last line...
	if c.col >= (*c.buffer).RunesInLine(c.line) && c.line < (*c.buffer).Lines()-1 {
		c.line, c.col = (*c.buffer).ClampLineCol(c.line+1, 0) // Go to beginning of line below
	} else {
		c.line, c.col = (*c.buffer).ClampLineCol(c.line, c.col+1)
	}
	return c
}

func (c Cursor) Up() Cursor {
	if c.line == 0 { // If the cursor is at the first line...
		c.line, c.col = 0, 0 // Go to beginning
	} else {
		c.line, c.col = (*c.buffer).ClampLineCol(c.line-1, c.col)
	}
	return c
}

func (c Cursor) Down() Cursor {
	if c.line == (*c.buffer).Lines()-1 { // If the cursor is at the last line...
		c.line, c.col = (*c.buffer).ClampLineCol(c.line, math.MaxInt32) // Go to end of current line
	} else {
		c.line, c.col = (*c.buffer).ClampLineCol(c.line+1, c.col)
	}
	return c
}

// NextWordBoundaryEnd proceeds to the position after the last character of the
// next word boundary to the right of the Cursor. A word boundary is the
// beginning or end of any sequence of similar or same-classed characters.
// Whitespace is skipped.
func (c Cursor) NextWordBoundaryEnd() Cursor {
	// Get position of cursor in buffer as pos
	// get classification of character at pos or assume none if whitespace
	// for each pos until end of buffer: pos + 1 (at end)
	//	 if pos char is not of previous pos char class:
	//      set cursor position as pos
	//

	// only skip contiguous characters for word characters
	// jump to position *after* any symbols

	pos := (*c.buffer).LineColToPos(c.line, c.col)
	startClass := getRuneCharclass((*c.buffer).RuneAtPos(pos))
	pos++
	(*c.buffer).EachRuneAtPos(pos, func(rpos int, r rune) bool {
		class := getRuneCharclass(r)
		if class != startClass && class != charwhitespace {
			return true
		}
		return false
	})

	c.line, c.col = (*c.buffer).PosToLineCol(pos)

	return c
}

func (c Cursor) PrevWordBoundaryStart() Cursor {
	return c
}

func (c Cursor) GetLineCol() (line, col int) {
	return c.line, c.col
}

// SetLineCol sets the line and col of the Cursor to those provided. `line` is
// clamped within the range (0, lines in buffer). `col` is then clamped within
// the range (0, line length in runes).
func (c Cursor) SetLineCol(line, col int) Cursor {
	c.line, c.col = (*c.buffer).ClampLineCol(line, col)
	return c
}

func (c Cursor) Eq(other Cursor) bool {
	return c.buffer == other.buffer && c.line == other.line && c.col == other.col
}

type charclass uint8

const (
	charwhitespace charclass = iota
	charword
	charsymbol
)

func getRuneCharclass(r rune) charclass {
	if unicode.IsSpace(r) {
		return charwhitespace
	} else if r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) {
		return charword
	} else {
		return charsymbol
	}
}
