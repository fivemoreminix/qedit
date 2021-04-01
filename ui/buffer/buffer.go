package buffer

import (
	"io"
)

// A Buffer is wrapper around any buffer data structure like ropes or a gap buffer
// that can be used for text editors. One way this interface helps is by making
// all API function parameters line and column indexes, so it is simple and easy
// to index and use like a text editor. All lines and columns start at zero, and
// all "end" ranges are inclusive.
//
// Any bounds out of range are panics! If you are unsure your position or range
// may be out of bounds, use ClampLineCol() or compare with Lines() or ColsInLine().
type Buffer interface {
	// Line returns a slice of the data at the given line, including the ending line-
	// delimiter. line starts from zero. Data returned may or may not be a copy: do not
	// write to it.
	Line(line int) []byte

	// Returns a slice of the buffer from startLine, startCol, to endLine, endCol,
	// inclusive bounds. The returned value may or may not be a copy of the data,
	// so do not write to it.
	Slice(startLine, startCol, endLine, endCol int) []byte

	// Bytes returns all of the bytes in the buffer. This function is very likely
	// to copy all of the data in the buffer. Use sparingly. Try using other methods,
	// where possible.
	Bytes() []byte

	// Insert copies a byte slice (inserting it) into the position at line, col.
	Insert(line, col int, value []byte)

	// Remove deletes any characters between startLine, startCol, and endLine,
	// endCol, inclusive bounds.
	Remove(startLine, startCol, endLine, endCol int)

	// Returns the number of occurrences of 'sequence' in the buffer, within the range
	// of start line and col, to end line and col. [start, end) (exclusive end).
	Count(startLine, startCol, endLine, endCol int, sequence []byte) int

	// Len returns the number of bytes in the buffer.
	Len() int

	// Lines returns the number of lines in the buffer. If the buffer is empty,
	// 1 is returned, because there is always at least one line. This function
	// basically counts the number of newline ('\n') characters in a buffer.
	Lines() int

	// RunesInLine returns the number of runes in the given line. That is, the
	// number of Utf-8 codepoints in the line, not bytes. Includes the line delimiter
	// in the count. If that line delimiter is CRLF ('\r\n'), then it adds two.
	RunesInLineWithDelim(line int) int

	// RunesInLine returns the number of runes in the given line. That is, the
	// number of Utf-8 codepoints in the line, not bytes. Excludes line delimiters.
	RunesInLine(line int) int

	// ClampLineCol is a utility function to clamp any provided line and col to
	// only possible values within the buffer, pointing to runes. It first clamps
	// the line, then clamps the column. The column is clamped between zero and
	// the last rune before the line delimiter.
	ClampLineCol(line, col int) (int, int)


	// LineColToPos returns the index of the byte at line, col. If line is less than
	// zero, or more than the number of available lines, the function will panic. If
	// col is less than zero, the function will panic. If col is greater than the
	// length of the line, the position of the last byte of the line is returned,
	// instead.
	LineColToPos(line, col int) int

	// PosToLineCol converts a byte offset (position) of the buffer's bytes, into
	// a line and column. Unless you are working with the Bytes() function, this
	// is unlikely to be useful to you. Position will be clamped.
	PosToLineCol(pos int) (int, int)

	WriteTo(w io.Writer) (int64, error)
}
