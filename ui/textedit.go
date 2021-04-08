package ui

import (
	"bytes"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/fivemoreminix/qedit/ui/buffer"
	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
)

// A Selection represents a region of the buffer to be selected for text editing
// purposes. It is asserted that the start position is less than the end position.
// The start and end are inclusive. If the EndCol of a Region is one more than the
// last column of a line, then it points to the line delimiter at the end of that
// line. It is understood that as a Region spans multiple lines, those connecting
// line-delimiters are included, as well.
type Region struct {
	StartLine, StartCol int
	EndLine, EndCol     int
}

// TextEdit is a field for line-based editing. It features syntax highlighting
// tools, is autocomplete ready, and contains the various information about
// content being edited.
type TextEdit struct {
	Buffer      buffer.Buffer
	Highlighter *buffer.Highlighter
	LineNumbers bool   // Whether to render line numbers (and therefore the column)
	Dirty       bool   // Whether the buffer has been edited
	UseHardTabs bool   // When true, tabs are '\t'
	TabSize     int    // How many spaces to indent by
	IsCRLF      bool   // Whether the file's line endings are CRLF (\r\n) or LF (\n)
	FilePath    string // Will be empty if the file has not been saved yet

	screen           *tcell.Screen // We keep our own reference to the screen for cursor purposes.
	curx, cury       int // Zero-based: cursor points before the character at that position.
	prevCurCol       int // Previous maximum column the cursor was at, when the user pressed left or right
	scrollx, scrolly int // X and Y offset of view, known as scroll

	selection  Region // Selection: selectMode determines if it should be used
	selectMode bool   // Whether the user is actively selecting text

	baseComponent
}

// New will initialize the buffer using the given 'contents'. If the 'filePath' or 'FilePath' is empty,
// it can be assumed that the TextEdit has no file association, or it is unsaved.
func NewTextEdit(screen *tcell.Screen, filePath string, contents []byte, theme *Theme) *TextEdit {
	te := &TextEdit{
		Buffer:      nil, // Set in SetContents
		Highlighter: nil, // Set in SetContents
		LineNumbers: true,
		UseHardTabs: true,
		TabSize:     4,
		FilePath:    filePath,

		screen:        screen,
		baseComponent: baseComponent{theme: theme},
	}
	te.SetContents(contents)
	return te
}

// SetContents applies the string to the internal buffer of the TextEdit component.
// The string is determined to be either CRLF or LF based on line-endings.
func (t *TextEdit) SetContents(contents []byte) {
	var i int
loop:
	for i < len(contents) {
		switch contents[i] {
		case '\n':
			t.IsCRLF = false
			break loop
		case '\r':
			// We could check for a \n after, but what's the point?
			t.IsCRLF = true
			break loop
		}
		_, size := utf8.DecodeRune(contents[i:])
		i += size
	}

	t.Buffer = buffer.NewRopeBuffer(contents)

	// TODO: replace with automatic determination of language via filetype
	lang := &buffer.Language{
		Name:      "Go",
		Filetypes: []string{".go"},
		Rules: map[*buffer.RegexpRegion]buffer.Syntax{
			&buffer.RegexpRegion{Start: regexp.MustCompile("\\/\\/.*")}: buffer.Comment,
			&buffer.RegexpRegion{Start: regexp.MustCompile("\".*?\"")}:  buffer.String,
			&buffer.RegexpRegion{
				Start: regexp.MustCompile("\\b(var|const|if|else|range|for|switch|fallthrough|case|default|break|continue|go|func|return|defer|import|type|package)\\b"),
			}: buffer.Keyword,
			&buffer.RegexpRegion{
				Start: regexp.MustCompile("\\b(u?int(8|16|32|64)?|rune|byte|string|bool|struct)\\b"),
			}: buffer.Type,
			&buffer.RegexpRegion{
				Start: regexp.MustCompile("\\b([1-9][0-9]*|0[0-7]*|0[Xx][0-9A-Fa-f]+|0[Bb][01]+)\\b"),
			}: buffer.Number,
			&buffer.RegexpRegion{
				Start: regexp.MustCompile("\\b(len|cap|panic|make|copy|append)\\b"),
			}: buffer.Builtin,
			&buffer.RegexpRegion{
				Start: regexp.MustCompile("\\b(nil|true|false)\\b"),
			}: buffer.Special,
		},
	}

	colorscheme := &buffer.Colorscheme{
		buffer.Default: tcell.Style{}.Foreground(tcell.ColorLightGray).Background(tcell.ColorBlack),
		buffer.Comment: tcell.Style{}.Foreground(tcell.ColorGray).Background(tcell.ColorBlack),
		buffer.String:  tcell.Style{}.Foreground(tcell.ColorOlive).Background(tcell.ColorBlack),
		buffer.Keyword: tcell.Style{}.Foreground(tcell.ColorNavy).Background(tcell.ColorBlack),
		buffer.Type:    tcell.Style{}.Foreground(tcell.ColorPurple).Background(tcell.ColorBlack),
		buffer.Number:  tcell.Style{}.Foreground(tcell.ColorFuchsia).Background(tcell.ColorBlack),
		buffer.Builtin: tcell.Style{}.Foreground(tcell.ColorBlue).Background(tcell.ColorBlack),
		buffer.Special: tcell.Style{}.Foreground(tcell.ColorFuchsia).Background(tcell.ColorBlack),
	}

	t.Highlighter = buffer.NewHighlighter(t.Buffer, lang, colorscheme)
}

// GetLineDelimiter returns "\r\n" for a CRLF buffer, or "\n" for an LF buffer.
func (t *TextEdit) GetLineDelimiter() string {
	if t.IsCRLF {
		return "\r\n"
	} else {
		return "\n"
	}
}

// Changes a file's line delimiters. If `crlf` is true, then line delimiters are replaced
// with Windows CRLF (\r\n). If `crlf` is false, then line delimtiers are replaced with Unix
// LF (\n). The TextEdit `IsCRLF` variable is updated with the new value.
func (t *TextEdit) ChangeLineDelimiters(crlf bool) {
	t.IsCRLF = crlf
	t.Dirty = true
	// line delimiters are constructed with String() function
	// TODO: ^ not true anymore ^
	panic("Cannot ChangeLineDelimiters")
}

// Delete with `forwards` false will backspace, destroying the character before the cursor,
// while Delete with `forwards` true will delete the character after (or on) the cursor.
// In insert mode, forwards is always true.
func (t *TextEdit) Delete(forwards bool) {
	t.Dirty = true

	var deletedLine bool // Whether any whole line has been deleted (changing the # of lines)
	startingLine := t.cury

	if t.selectMode { // If text is selected, delete the whole selection
		t.selectMode = false // Disable selection and prevent infinite loop

		// Delete the region
		t.Buffer.Remove(t.selection.StartLine, t.selection.StartCol, t.selection.EndLine, t.selection.EndCol)
		t.SetLineCol(t.selection.StartLine, t.selection.StartCol) // Set cursor to start of region

		deletedLine = t.selection.StartLine != t.selection.EndLine
	} else { // Not deleting selection
		if forwards { // Delete the character after the cursor
			// If the cursor is not at the end of the last line...
			if t.cury < t.Buffer.Lines()-1 || t.curx < t.Buffer.RunesInLine(t.cury) {
				bytes := t.Buffer.Slice(t.cury, t.curx, t.cury, t.curx) // Get the character at cursor
				deletedLine = bytes[0] == '\n'

				t.Buffer.Remove(t.cury, t.curx, t.cury, t.curx) // Remove character at cursor
			}
		} else { // Delete the character before the cursor
			// If the cursor is not at the first column of the first line...
			if t.cury > 0 || t.curx > 0 {
				t.CursorLeft() // Back up to that character

				bytes := t.Buffer.Slice(t.cury, t.curx, t.cury, t.curx) // Get the char at cursor
				deletedLine = bytes[0] == '\n'

				t.Buffer.Remove(t.cury, t.curx, t.cury, t.curx) // Remove character at cursor
			}
		}
	}

	if deletedLine {
		t.Highlighter.InvalidateLines(startingLine, t.Buffer.Lines()-1)
	} else {
		t.Highlighter.InvalidateLines(startingLine, startingLine)
	}
}

// Writes `contents` at the cursor position. Line delimiters and tab character supported.
// Any other control characters will be printed. Overwrites any active selection.
func (t *TextEdit) Insert(contents string) {
	t.Dirty = true

	if t.selectMode { // If there is a selection...
		// Go to and delete the selection
		t.Delete(true) // The parameter doesn't matter with selection
	}

	var lineInserted bool // True if contents contains a '\n'
	startingLine := t.cury

	runes := []rune(contents)
	for i := 0; i < len(runes); i++ {
		ch := runes[i]
		switch ch {
		case '\r':
			// If the character after is a \n, then it is a CRLF
			if i+1 < len(runes) && runes[i+1] == '\n' {
				i++ // Consume '\n' after
				t.Buffer.Insert(t.cury, t.curx, []byte{'\n'})
				t.SetLineCol(t.cury+1, 0) // Go to the start of that new line
				lineInserted = true
			}
		case '\n':
			t.Buffer.Insert(t.cury, t.curx, []byte{'\n'})
			t.SetLineCol(t.cury+1, 0) // Go to the start of that new line
			lineInserted = true
		case '\b':
			t.Delete(false) // Delete the character before the cursor
		case '\t':
			if !t.UseHardTabs { // If this file does not use hard tabs...
				// Insert spaces
				spaces := strings.Repeat(" ", t.TabSize)
				t.Buffer.Insert(t.cury, t.curx, []byte(spaces))
				t.SetLineCol(t.cury, t.curx+len(spaces)) // Advance the cursor
				break
			}
			fallthrough // Append the \t character
		default:
			// Insert character into line
			t.Buffer.Insert(t.cury, t.curx, []byte(string(ch)))
			t.SetLineCol(t.cury, t.curx+1) // Advance the cursor
		}
	}
	t.prevCurCol = t.curx

	if lineInserted {
		t.Highlighter.InvalidateLines(startingLine, t.Buffer.Lines()-1)
	} else {
		t.Highlighter.InvalidateLines(startingLine, startingLine)
	}
}

// getTabCountInLineAtCol returns tabs in the given line, before the column position,
// if hard tabs are enabled. If hard tabs are not enabled, the function returns zero.
// Multiply returned tab count by TabSize to get the offset produced by tabs.
// Col must be a valid column position in the given line. Maybe call clampLineCol before
// this function.
func (t *TextEdit) getTabCountInLineAtCol(line, col int) int {
	if t.UseHardTabs {
		return t.Buffer.Count(line, 0, line, col, []byte{'\t'})
	}
	return 0
}

// GetLineCol returns (line, col) of the cursor. Zero is origin for both.
func (t *TextEdit) GetLineCol() (int, int) {
	return t.cury, t.curx
}

// SetLineCol sets the cursor line and column position. Zero is origin for both.
// If `line` is out of bounds, `line` will be clamped to the closest available line.
// If `col` is out of bounds, `col` will be clamped to the closest column available for the line.
// Will scroll the TextEdit just enough to see the line the cursor is at.
func (t *TextEdit) SetLineCol(line, col int) {
	line, col = t.Buffer.ClampLineCol(line, col)

	// Handle hard tabs
	tabOffset := t.getTabCountInLineAtCol(line, col) * (t.TabSize - 1) // Offset for the current line from hard tabs (temporary; purely visual)

	// Scroll the screen when going to lines out of view
	if line >= t.scrolly+t.height-1 { // If the new line is below view...
		t.scrolly = line - t.height + 1 // Scroll just enough to view that line
	} else if line < t.scrolly { // If the new line is above view
		t.scrolly = line
	}

	columnWidth := t.getColumnWidth()

	// Scroll the screen horizontally when going to columns out of view
	if col+tabOffset >= t.scrollx+(t.width-columnWidth-1) { // If the new column is right of view
		t.scrollx = (col + tabOffset) - (t.width - columnWidth) + 1 // Scroll just enough to view that column
	} else if col+tabOffset < t.scrollx { // If the new column is left of view
		t.scrollx = col + tabOffset // Scroll left enough to view that column
	}

	if t.scrollx < 0 {
		panic("oops")
	}

	t.cury, t.curx = line, col
	if t.focused && !t.selectMode {
		(*t.screen).ShowCursor(t.x+columnWidth+col+tabOffset-t.scrollx, t.y+line-t.scrolly)
	} else {
		(*t.screen).HideCursor()
	}
}

// CursorUp moves the cursor up a line.
func (t *TextEdit) CursorUp() {
	if t.cury <= 0 { // If the cursor is at the first line...
		t.SetLineCol(t.cury, 0) // Go to beginning
	} else {
		line, col := t.Buffer.ClampLineCol(t.cury-1, t.prevCurCol)
		if t.UseHardTabs { // When using hard tabs, subtract offsets produced by tabs
			tabOffset := t.getTabCountInLineAtCol(line, col) * (t.TabSize - 1)
			col -= tabOffset // We still count each \t in the col
		}
		t.SetLineCol(line, col)
	}
}

// CursorDown moves the cursor down a line.
func (t *TextEdit) CursorDown() {
	if t.cury >= t.Buffer.Lines()-1 { // If the cursor is at the last line...
		t.SetLineCol(t.cury, math.MaxInt32) // Go to end of current line
	} else {
		line, col := t.Buffer.ClampLineCol(t.cury+1, t.prevCurCol)
		if t.UseHardTabs {
			tabOffset := t.getTabCountInLineAtCol(line, col) * (t.TabSize - 1)
			col -= tabOffset // We still count each \t in the col
		}
		t.SetLineCol(line, col) // Go to line below
	}
}

// CursorLeft moves the cursor left a column.
func (t *TextEdit) CursorLeft() {
	if t.curx <= 0 && t.cury != 0 { // If we are at the beginning of the current line...
		t.SetLineCol(t.cury-1, math.MaxInt32) // Go to end of line above
	} else {
		t.SetLineCol(t.cury, t.curx-1)
	}
	tabOffset := t.getTabCountInLineAtCol(t.cury, t.curx) * (t.TabSize - 1)
	t.prevCurCol = t.curx + tabOffset
}

// CursorRight moves the cursor right a column.
func (t *TextEdit) CursorRight() {
	// If we are at the end of the current line,
	// and not at the last line...
	if t.curx >= t.Buffer.RunesInLine(t.cury) && t.cury < t.Buffer.Lines()-1 {
		t.SetLineCol(t.cury+1, 0) // Go to beginning of line below
	} else {
		t.SetLineCol(t.cury, t.curx+1)
	}
	tabOffset := t.getTabCountInLineAtCol(t.cury, t.curx) * (t.TabSize - 1)
	t.prevCurCol = t.curx + tabOffset
}

// getColumnWidth returns the width of the line numbers column if it is present.
func (t *TextEdit) getColumnWidth() int {
	columnWidth := 0
	if t.LineNumbers {
		// Set columnWidth to max count of line number digits
		columnWidth = Max(2, len(strconv.Itoa(t.Buffer.Lines()))) // Column has minimum width of 2
	}
	return columnWidth
}

// GetSelectedBytes returns a byte slice of the region of the buffer that is currently selected.
// If the returned string is empty, then nothing was selected. The slice returned may or may not
// be a copy of the buffer, so do not write to it.
func (t *TextEdit) GetSelectedBytes() []byte {
	// TODO: there's a bug with copying text
	if t.selectMode {
		return t.Buffer.Slice(t.selection.StartLine, t.selection.StartCol, t.selection.EndLine, t.selection.EndCol)
	}
	return []byte{}
}

// Draw renders the TextEdit component.
func (t *TextEdit) Draw(s tcell.Screen) {
	columnWidth := t.getColumnWidth()
	bufferLines := t.Buffer.Lines()

	selectedStyle := t.theme.GetOrDefault("TextEditSelected")
	columnStyle := t.theme.GetOrDefault("TextEditColumn")

	t.Highlighter.UpdateInvalidatedLines(t.scrolly, t.scrolly+(t.height-1))

	var tabBytes []byte
	if t.UseHardTabs {
		// Only call Repeat once for each draw in hard tab files
		tabBytes = bytes.Repeat([]byte{' '}, t.TabSize)
	}

	defaultStyle := t.Highlighter.Colorscheme.GetStyle(buffer.Default)
	currentStyle := defaultStyle

	for lineY := t.y; lineY < t.y+t.height; lineY++ { // For each line we can draw...
		line := lineY + t.scrolly - t.y // The line number being drawn (starts at zero)

		lineNumStr := "" // Line number as a string

		if line < bufferLines { // Only index buffer if we are within it...
			lineNumStr = strconv.Itoa(line + 1) // Only set for lines within the buffer (not view)

			var origLineBytes []byte = t.Buffer.Line(line)
			var lineBytes []byte = origLineBytes // Line to be drawn

			// When iterating lineTabs: the value at i is
			// the rune index the tab was found at.
			//			var lineTabs  [128]int // Rune index for each hard tab '\t' in lineBytes
			//			var tabs      int // Length of lineTabs (number of hard tabs)
			if t.UseHardTabs {
				//				var ri int // rune index
				//				var i int
				//				for i < len(lineBytes) {
				//					r, size := utf8.DecodeRune(lineBytes[i:])
				//					if r == '\t' {
				//						lineTabs[tabs] = ri
				//						tabs++
				//					}
				//					i += size
				//					ri++
				//				}
				lineBytes = bytes.ReplaceAll(lineBytes, []byte{'\t'}, tabBytes)
			}

			lineHighlightData := t.Highlighter.GetLineMatches(line)
			var lineHighlightDataIdx int

			var byteIdx int // Byte index of lineStr
			// X offset we draw the next rune at (some runes can be 2 cols wide)
			col := t.x + columnWidth
			var runeIdx int // Index into lineStr (as runes) we draw the next character at

			// REWRITE OF SCROLL FUNC:
			for runeIdx < t.scrollx && byteIdx < len(lineBytes) {
				_, size := utf8.DecodeRune(lineBytes[byteIdx:]) // Respect UTF-8
				byteIdx += size
				runeIdx++
			}

			tabOffsetAtRuneIdx := func(idx int) int {
				var count int
				var i int
				for i < len(origLineBytes) {
					r, size := utf8.DecodeRune(origLineBytes[i:])
					if r == '\t' {
						count++
					}
					i += size
				}
				return count * (t.TabSize - 1)
			}

			// origRuneIdx converts a rune index from lineBytes to a runeIndex from origLineBytes
			// not affected by the hard tabs becoming 4 or 8 spaces.
			origRuneIdx := func(idx int) int { // returns the idx that is not mutated by hard tabs
				var ridx int // new rune idx
				var i int    // byte index
				for idx > 0 {
					r, size := utf8.DecodeRune(origLineBytes[i:])
					if r == '\t' {
						idx -= t.TabSize
					} else {
						idx--
					}
					if idx >= 0 { // causes ridx = 0, when idx = 3
						ridx++
					}
					i += size
				}
				return ridx
			}

			for col < t.x+t.width { // For each column in view...
				var r rune = ' '  // Rune to draw this iteration
				var size int = 1  // Size of the rune (in bytes)
				var selected bool // Whether this rune should be styled as selected

				tabOffsetAtRuneIdx := tabOffsetAtRuneIdx(runeIdx)

				if byteIdx < len(lineBytes) { // If we are drawing part of the line contents...
					r, size = utf8.DecodeRune(lineBytes[byteIdx:])

					if r == '\n' {
						r = ' '
					}

					// Determine whether we select the current rune. Also only select runes within
					// the line bytes range.
					if t.selectMode && line >= t.selection.StartLine && line <= t.selection.EndLine { // If we're part of a selection...
						_origRuneIdx := origRuneIdx(runeIdx)
						if line == t.selection.StartLine { // If selection starts at this line...
							if _origRuneIdx >= t.selection.StartCol { // And we're at or past the start col...
								// If the start line is also the end line...
								if line == t.selection.EndLine {
									if _origRuneIdx <= t.selection.EndCol { // And we're before the end of that...
										selected = true
									}
								} else { // Definitely highlight
									selected = true
								}
							}
						} else if line == t.selection.EndLine { // If selection ends at this line...
							if _origRuneIdx <= t.selection.EndCol { // And we're at or before the end col...
								selected = true
							}
						} else { // We're between the start and the end lines, definitely highlight.
							selected = true
						}
					}
				}

				// Determine the style of the rune we draw next:

				if selected {
					currentStyle = selectedStyle
				} else {
					currentStyle = defaultStyle

					if lineHighlightDataIdx < len(lineHighlightData) { // Works for single-line highlights
						data := lineHighlightData[lineHighlightDataIdx]
						if runeIdx-tabOffsetAtRuneIdx >= data.Col {
							if runeIdx-tabOffsetAtRuneIdx > data.EndCol { // Passed that highlight data
								currentStyle = defaultStyle
								lineHighlightDataIdx++ // Go to next one
							} else { // Start coloring as this syntax style
								currentStyle = t.Highlighter.Colorscheme.GetStyle(data.Syntax)
							}
						}
					}
				}

				// Draw the rune
				s.SetContent(col, lineY, r, nil, currentStyle)

				col += runewidth.RuneWidth(r)

				// Understanding the tab simulation is unnecessary; just know that it works.
				byteIdx += size
				runeIdx++
			}
		}

		columnStr := fmt.Sprintf("%s%s", strings.Repeat(" ", columnWidth-len(lineNumStr)), lineNumStr) // Right align line number

		DrawStr(s, t.x, lineY, columnStr, columnStyle) // Draw column
	}

	// Update cursor
	t.SetLineCol(t.cury, t.curx)
}

// SetFocused sets whether the TextEdit is focused. When focused, the cursor is set visible
// and its position is updated on every event.
func (t *TextEdit) SetFocused(v bool) {
	t.focused = v
	if v {
		t.SetLineCol(t.cury, t.curx)
	} else {
		(*t.screen).HideCursor()
	}
}

// HandleEvent allows the TextEdit to handle `event` if it chooses, returns
// whether the TextEdit handled the event.
func (t *TextEdit) HandleEvent(event tcell.Event) bool {
	switch ev := event.(type) {
	case *tcell.EventKey:
		switch ev.Key() {
		// Cursor movement
		case tcell.KeyUp:
			if ev.Modifiers() == tcell.ModShift {
				if !t.selectMode {
					t.selection.StartLine, t.selection.StartCol = t.cury, t.curx
					t.selection.EndLine, t.selection.EndCol = t.cury, t.curx
					t.selectMode = true
				} else {
					prevCurX, prevCurY := t.curx, t.cury
					t.CursorUp()
					// Grow the selection in the correct direction
					if prevCurY <= t.selection.StartLine && prevCurX <= t.selection.StartCol {
						t.selection.StartLine, t.selection.StartCol = t.cury, t.curx
					} else {
						t.selection.EndLine, t.selection.EndCol = t.cury, t.curx
					}
				}
			} else {
				t.selectMode = false
				t.CursorUp()
			}
		case tcell.KeyDown:
			if ev.Modifiers() == tcell.ModShift {
				if !t.selectMode {
					t.selection.StartLine, t.selection.StartCol = t.cury, t.curx
					t.selection.EndLine, t.selection.EndCol = t.cury, t.curx
					t.selectMode = true
				} else {
					prevCurX, prevCurY := t.curx, t.cury
					t.CursorDown()
					if prevCurY >= t.selection.EndLine && prevCurX >= t.selection.EndCol {
						t.selection.EndLine, t.selection.EndCol = t.cury, t.curx
					} else {
						t.selection.StartLine, t.selection.StartCol = t.cury, t.curx
					}
				}
			} else {
				t.selectMode = false
				t.CursorDown()
			}
		case tcell.KeyLeft:
			if ev.Modifiers() == tcell.ModShift {
				if !t.selectMode {
					t.CursorLeft() // We want the character to the left to be selected only (think insert)
					t.selection.StartLine, t.selection.StartCol = t.cury, t.curx
					t.selection.EndLine, t.selection.EndCol = t.cury, t.curx
					t.selectMode = true
				} else {
					prevCurX, prevCurY := t.curx, t.cury
					t.CursorLeft()
					if prevCurY == t.selection.StartLine && prevCurX == t.selection.StartCol { // We are moving the start...
						t.selection.StartLine, t.selection.StartCol = t.cury, t.curx
					} else {
						t.selection.EndLine, t.selection.EndCol = t.cury, t.curx
					}
				}
			} else {
				t.selectMode = false
				t.CursorLeft()
			}
		case tcell.KeyRight:
			if ev.Modifiers() == tcell.ModShift {
				if !t.selectMode { // If we are not already selecting...
					// Reset the selection to cursor pos
					t.selection.StartLine, t.selection.StartCol = t.cury, t.curx
					t.selection.EndLine, t.selection.EndCol = t.cury, t.curx
					t.selectMode = true
				} else {
					prevCurX, prevCurY := t.curx, t.cury
					t.CursorRight() // Advance the cursor
					if prevCurY == t.selection.EndLine && prevCurX == t.selection.EndCol {
						t.selection.EndLine, t.selection.EndCol = t.cury, t.curx
					} else {
						t.selection.StartLine, t.selection.StartCol = t.cury, t.curx
					}
				}
			} else {
				t.selectMode = false
				t.CursorRight()
			}
		case tcell.KeyHome:
			t.SetLineCol(t.cury, 0)
			t.prevCurCol = t.curx
		case tcell.KeyEnd:
			t.SetLineCol(t.cury, math.MaxInt32) // Max column
			t.prevCurCol = t.curx
		case tcell.KeyPgUp:
			t.SetLineCol(t.scrolly-t.height, t.curx) // Go a page up
			t.prevCurCol = t.curx
		case tcell.KeyPgDn:
			t.SetLineCol(t.scrolly+t.height*2-1, t.curx) // Go a page down
			t.prevCurCol = t.curx

		// Deleting
		case tcell.KeyBackspace:
			fallthrough
		case tcell.KeyBackspace2:
			t.Delete(false)
		case tcell.KeyDelete:
			t.Delete(true)

		// Other control
		case tcell.KeyTab:
			t.Insert("\t") // (can translate to four spaces)
		case tcell.KeyEnter:
			t.Insert("\n")

		// Inserting
		case tcell.KeyRune:
			t.Insert(string(ev.Rune())) // Insert rune
		default:
			return false
		}
		return true
	}
	return false
}
