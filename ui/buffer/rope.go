package buffer

import (
	"io"
	"unicode/utf8"

	"github.com/zyedidia/rope"
)

type RopeBuffer rope.Node

func NewRopeBuffer(contents []byte) *RopeBuffer {
	return (*RopeBuffer)(rope.New(contents))
}

// Returns the index of the byte at line, col. If line is less than zero, or more
// than the number of available lines, the function will panic. If col is less than
// zero, the function will panic. If col is greater than the length of the line,
// the position of the last byte of the line is returned, instead.
func (b *RopeBuffer) pos(line, col int) int {
	var pos int

	_rope := (*rope.Node)(b)

	pos = b.getLineStartPos(line)

	// Have to do this algorithm for safety. If this function was declared to panic
	// or index out of bounds memory, if col > the given line length, it would be
	// more efficient and simpler. But unfortunately, I believe it is necessary.
	if col > 0 {
		_, r := _rope.SplitAt(pos)
		l, _ := r.SplitAt(_rope.Len() - pos)

		l.EachLeaf(func(n *rope.Node) bool {
			data := n.Value() // Reference; not a copy.
			var i int
			for i < len(data) {
				if col == 0 || data[i] == '\n' {
					return true // Found the position of the column
				}
				pos++
				col--

				// Respect Utf-8 codepoint boundaries
				_, size := utf8.DecodeRune(data[i:])
				i += size
			}
			return false // Have not gotten to the appropriate position, yet
		})
	}

	return pos
}

// Line returns a slice of the data at the given line, including the ending line-
// delimiter. line starts from zero. Data returned may or may not be a copy: do not
// write it.
func (b *RopeBuffer) Line(line int) []byte {
	pos   := b.getLineStartPos(line)
	bytes := 0

	_rope := (*rope.Node)(b)
	_, r := _rope.SplitAt(pos)
	l, _ := r.SplitAt(_rope.Len() - pos)

	l.EachLeaf(func(n *rope.Node) bool {
		data := n.Value() // Reference; not a copy.
		var i int
		for i < len(data) {
			if data[i] == '\n' {
				bytes++ // Add the newline byte
				return true // Read (past-tense) the whole line
			}

			// Respect Utf-8 codepoint boundaries
			_, size := utf8.DecodeRune(data[i:])
			bytes += size
			i += size
		}
		return false // Have not read the whole line, yet
	})

	return _rope.Slice(pos, pos+bytes) // NOTE: may be faster to do it ourselves
}

// Returns a slice of the buffer from startLine, startCol, to endLine, endCol,
// inclusive bounds. The returned value may or may not be a copy of the data,
// so do not write to it.
func (b *RopeBuffer) Slice(startLine, startCol, endLine, endCol int) []byte {
	return (*rope.Node)(b).Slice(b.pos(startLine, startCol), b.pos(endLine, endCol)+1)
}

// Bytes returns all of the bytes in the buffer. This function is very likely
// to copy all of the data in the buffer. Use sparingly. Try using other methods,
// where possible.
func (b *RopeBuffer) Bytes() []byte {
	return (*rope.Node)(b).Value()
}

// Insert copies a byte slice (inserting it) into the position at line, col.
func (b *RopeBuffer) Insert(line, col int, value []byte) {
	(*rope.Node)(b).Insert(b.pos(line, col), value)
}

// Remove deletes any characters between startLine, startCol, and endLine,
// endCol, inclusive bounds.
func (b *RopeBuffer) Remove(startLine, startCol, endLine, endCol int) {
	(*rope.Node)(b).Remove(b.pos(startLine, startCol), b.pos(endLine, endCol)+1)
}

// Len returns the number of bytes in the buffer.
func (b *RopeBuffer) Len() int {
	return (*rope.Node)(b).Len()
}

// Lines returns the number of lines in the buffer. If the buffer is empty,
// 1 is returned, because there is always at least one line. This function
// basically counts the number of newline ('\n') characters in a buffer.
func (b *RopeBuffer) Lines() int {
	rope := (*rope.Node)(b)
	return rope.Count(0, rope.Len(), []byte{'\n'}) + 1
}

// pos is the first byte index in the line we want to count runes/cols. pos can point to
// a newline or be greater than or equal to the number of bytes in the buffer, and will
// return 0 for both cases.
func (b *RopeBuffer) runesInLine(pos int) int {
	_rope := (*rope.Node)(b)
	ropeLen := _rope.Len()

	if pos >= ropeLen {
		return 0
	}

	var count int

	_, r := _rope.SplitAt(pos)
	l, _ := r.SplitAt(ropeLen - pos)

	l.EachLeaf(func(n *rope.Node) bool {
		data := n.Value() // Reference; not a copy.
		var i int
		for i < len(data) {
			if data[i] == '\n' {
				return true // Read (past-tense) the whole line
			}
			count++

			// Respect Utf-8 codepoint boundaries
			_, size := utf8.DecodeRune(data[i:])
			i += size
		}
		return false // Have not read the whole line, yet
	})

	return count
}

// getLineStartPos returns the first byte index of the given line (starting from zero).
// The returned index can be equal to the length of the buffer, not pointing to any byte,
// which means the byte is on the last, and empty, line of the buffer. If line is greater
// than or equal to the number of lines in the buffer, a panic is issued.
func (b *RopeBuffer) getLineStartPos(line int) int {
	_rope := (*rope.Node)(b)
	var pos int

	if line > 0 {
		_rope.IndexAllFunc(0, _rope.Len(), []byte{'\n'}, func(idx int) bool {
			line--
			pos = idx + 1 // idx+1 = start of line after delimiter
			if line <= 0 { // If pos is now the start of the line we're searching for...
				return true // Stop indexing
			}
			return false
		})
	}

	if line > 0 { // If there aren't enough lines to reach line...
		panic("getLineStartPos: not enough lines in buffer to reach position")
	}

	return pos
}

// RunesInLine returns the number of runes in the given line. That is, the
// number of Utf-8 codepoints in the line, not bytes.
func (b *RopeBuffer) RunesInLine(line int) int {
	linePos := b.getLineStartPos(line)
	return b.runesInLine(linePos)
}

// ClampLineCol is a utility function to clamp any provided line and col to
// only possible values within the buffer, pointing to runes. It first clamps
// the line, then clamps the column.
func (b *RopeBuffer) ClampLineCol(line, col int) (int, int) {
	if line < 0 {
		line = 0
	} else if lines := b.Lines()-1; line > lines {
		line = lines		
	}

	if col < 0 {
		col = 0
	} else if cols := b.RunesInLine(line)-1; col > cols {
		col = cols
	}

	return line, col
}

func (b *RopeBuffer) WriteTo(w io.Writer) (int64, error) {
	return (*rope.Node)(b).WriteTo(w)
}
