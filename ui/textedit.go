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
	cursor           buffer.Cursor
	scrollx, scrolly int // X and Y offset of view, known as scroll
	theme            *Theme

	selection  buffer.Region // Selection: selectMode determines if it should be used
	selectMode bool          // Whether the user is actively selecting text

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
	t.cursor = buffer.NewCursor(&t.Buffer)

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
		buffer.Column:  tcell.Style{}.Foreground(tcell.ColorDarkGray).Background(tcell.ColorBlack),
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
	cursLine, cursCol := t.cursor.GetLineCol()
	startingLine := cursLine

	if t.selectMode { // If text is selected, delete the whole selection
		t.selectMode = false // Disable selection and prevent infinite loop

		startLine, startCol := t.selection.Start()
		endLine, endCol := t.selection.End()

		// Delete the region
		t.Buffer.Remove(startLine, startCol, endLine, endCol)
		t.cursor.SetLineCol(startLine, startCol) // Set cursor to start of region

		deletedLine = startLine != endLine
	} else { // Not deleting selection
		if forwards { // Delete the character after the cursor
			// If the cursor is not at the end of the last line...
			if cursLine < t.Buffer.Lines()-1 || cursCol < t.Buffer.RunesInLine(cursLine) {
				bytes := t.Buffer.Slice(cursLine, cursCol, cursLine, cursCol) // Get the character at cursor
				deletedLine = bytes[0] == '\n'

				t.Buffer.Remove(cursLine, cursCol, cursLine, cursCol) // Remove character at cursor
			}
		} else { // Delete the character before the cursor
			// If the cursor is not at the first column of the first line...
			if cursLine > 0 || cursCol > 0 {
				t.cursor = t.cursor.Left() // Back up to that character

				bytes := t.Buffer.Slice(cursLine, cursCol, cursLine, cursCol) // Get the char at cursor
				deletedLine = bytes[0] == '\n'

				t.Buffer.Remove(cursLine, cursCol, cursLine, cursCol) // Remove character at cursor
			}
		}
	}

	t.ScrollToCursor()
	t.updateCursorVisibility()

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
	cursLine, cursCol := t.cursor.GetLineCol()
	startingLine := cursLine

	runes := []rune(contents)
	for i := 0; i < len(runes); i++ {
		ch := runes[i]
		switch ch {
		case '\r':
			// If the character after is a \n, then it is a CRLF
			if i+1 < len(runes) && runes[i+1] == '\n' {
				i++ // Consume '\n' after
				t.Buffer.Insert(cursLine, cursCol, []byte{'\n'})
				lineInserted = true
			}
		case '\n':
			t.Buffer.Insert(cursLine, cursCol, []byte{'\n'})
			lineInserted = true
		case '\b':
			t.Delete(false) // Delete the character before the cursor
		case '\t':
			if !t.UseHardTabs { // If this file does not use hard tabs...
				// Insert spaces
				spaces := strings.Repeat(" ", t.TabSize)
				t.Buffer.Insert(cursLine, cursCol, []byte(spaces))
				break
			}
			fallthrough // Append the \t character
		default:
			// Insert character into line
			t.Buffer.Insert(cursLine, cursCol, []byte(string(ch)))
			// t.SetLineCol(t.cury, t.curx+1) // Advance the cursor
		}
	}

	t.ScrollToCursor()
	t.updateCursorVisibility()

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

// updateCursorVisibility sets the position of the terminal's cursor with the
// cursor of the TextEdit. Sends a signal to show the cursor if the TextEdit
// is focused and not in select mode.
func (t *TextEdit) updateCursorVisibility() {
	if t.focused && !t.selectMode {
		columnWidth := t.getColumnWidth()
		line, col := t.cursor.GetLineCol()
		tabOffset := t.getTabCountInLineAtCol(line, col) * (t.TabSize - 1)
		(*t.screen).ShowCursor(t.x+columnWidth+col+tabOffset-t.scrollx, t.y+line-t.scrolly)
	}
}

// Scroll the screen if the cursor is out of view.
func (t *TextEdit) ScrollToCursor() {
	line, col := t.cursor.GetLineCol()

	// Handle hard tabs
	tabOffset := t.getTabCountInLineAtCol(line, col) * (t.TabSize - 1) // Offset for the current line from hard tabs

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
}

func (t *TextEdit) GetCursor() buffer.Cursor {
	return t.cursor
}

func (t *TextEdit) SetCursor(newCursor buffer.Cursor) {
	t.cursor = newCursor
	t.updateCursorVisibility()
}

// getColumnWidth returns the width of the line numbers column if it is present.
func (t *TextEdit) getColumnWidth() int {
	var columnWidth int
	if t.LineNumbers {
		// Set columnWidth to max count of line number digits
		columnWidth = Max(3, 1+len(strconv.Itoa(t.Buffer.Lines()))) // Column has minimum width of 2
	}
	return columnWidth
}

// GetSelectedBytes returns a byte slice of the region of the buffer that is currently selected.
// If the returned string is empty, then nothing was selected. The slice returned may or may not
// be a copy of the buffer, so do not write to it.
func (t *TextEdit) GetSelectedBytes() []byte {
	// TODO: there's a bug with copying text
	if t.selectMode {
		startLine, startCol := t.selection.Start()
		endLine, endCol := t.selection.End()
		return t.Buffer.Slice(startLine, startCol, endLine, endCol)
	}
	return []byte{}
}

// Draw renders the TextEdit component.
func (t *TextEdit) Draw(s tcell.Screen) {
	columnWidth := t.getColumnWidth()
	bufferLines := t.Buffer.Lines()

	selectedStyle := t.theme.GetOrDefault("TextEditSelected")
	columnStyle := t.Highlighter.Colorscheme.GetStyle(buffer.Column)

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

			if t.UseHardTabs {
				lineBytes = bytes.ReplaceAll(lineBytes, []byte{'\t'}, tabBytes)
			}

			lineHighlightData := t.Highlighter.GetLineMatches(line)
			var lineHighlightDataIdx int

			var byteIdx int // Byte index of lineStr
			// X offset we draw the next rune at (some runes can be 2 cols wide)
			col := t.x + columnWidth
			var runeIdx int // Index into lineStr (as runes) we draw the next character at

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

					startLine, startCol := t.selection.Start()
					endLine, endCol := t.selection.End()

					// Determine whether we select the current rune. Also only select runes within
					// the line bytes range.
					if t.selectMode && line >= startLine && line <= endLine { // If we're part of a selection...
						_origRuneIdx := origRuneIdx(runeIdx)
						if line == startLine { // If selection starts at this line...
							if _origRuneIdx >= startCol { // And we're at or past the start col...
								// If the start line is also the end line...
								if line == endLine {
									if _origRuneIdx <= endCol { // And we're before the end of that...
										selected = true
									}
								} else { // Definitely highlight
									selected = true
								}
							}
						} else if line == endLine { // If selection ends at this line...
							if _origRuneIdx <= endCol { // And we're at or before the end col...
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

		columnStr := fmt.Sprintf("%s%s│", strings.Repeat(" ", columnWidth-len(lineNumStr)-1), lineNumStr) // Right align line number

		DrawStr(s, t.x, lineY, columnStr, columnStyle) // Draw column
	}

	t.updateCursorVisibility()
}

// SetFocused sets whether the TextEdit is focused. When focused, the cursor is set visible
// and its position is updated on every event.
func (t *TextEdit) SetFocused(v bool) {
	t.focused = v
	if v {
		t.updateCursorVisibility()
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
				// if !t.selectMode {
				// 	t.selection.StartLine, t.selection.StartCol = t.cury, t.curx
				// 	t.selection.EndLine, t.selection.EndCol = t.cury, t.curx
				// 	t.selectMode = true
				// } else {
				// 	prevCurX, prevCurY := t.curx, t.cury
				// 	t.CursorUp()
				// 	// Grow the selection in the correct direction
				// 	if prevCurY <= t.selection.StartLine && prevCurX <= t.selection.StartCol {
				// 		t.selection.StartLine, t.selection.StartCol = t.cury, t.curx
				// 	} else {
				// 		t.selection.EndLine, t.selection.EndCol = t.cury, t.curx
				// 	}
				// }
			} else {
				t.selectMode = false
				t.SetCursor(t.cursor.Up())
				t.ScrollToCursor()
			}
		case tcell.KeyDown:
			if ev.Modifiers() == tcell.ModShift {
				// if !t.selectMode {
				// 	t.selection.StartLine, t.selection.StartCol = t.cury, t.curx
				// 	t.selection.EndLine, t.selection.EndCol = t.cury, t.curx
				// 	t.selectMode = true
				// } else {
				// 	prevCurX, prevCurY := t.curx, t.cury
				// 	t.CursorDown()
				// 	if prevCurY >= t.selection.EndLine && prevCurX >= t.selection.EndCol {
				// 		t.selection.EndLine, t.selection.EndCol = t.cury, t.curx
				// 	} else {
				// 		t.selection.StartLine, t.selection.StartCol = t.cury, t.curx
				// 	}
				// }
			} else {
				t.selectMode = false
				t.SetCursor(t.cursor.Down())
				t.ScrollToCursor()
			}
		case tcell.KeyLeft:
			if ev.Modifiers() == tcell.ModShift {
				// if !t.selectMode {
				// 	t.CursorLeft() // We want the character to the left to be selected only (think insert)
				// 	t.selection.StartLine, t.selection.StartCol = t.cury, t.curx
				// 	t.selection.EndLine, t.selection.EndCol = t.cury, t.curx
				// 	t.selectMode = true
				// } else {
				// 	prevCurX, prevCurY := t.curx, t.cury
				// 	t.CursorLeft()
				// 	if prevCurY == t.selection.StartLine && prevCurX == t.selection.StartCol { // We are moving the start...
				// 		t.selection.StartLine, t.selection.StartCol = t.cury, t.curx
				// 	} else {
				// 		t.selection.EndLine, t.selection.EndCol = t.cury, t.curx
				// 	}
				// }
			} else {
				t.selectMode = false
				t.SetCursor(t.cursor.Left())
				t.ScrollToCursor()
			}
		case tcell.KeyRight:
			if ev.Modifiers() == tcell.ModShift {
				// if !t.selectMode { // If we are not already selecting...
				// 	// Reset the selection to cursor pos
				// 	t.selection.StartLine, t.selection.StartCol = t.cury, t.curx
				// 	t.selection.EndLine, t.selection.EndCol = t.cury, t.curx
				// 	t.selectMode = true
				// } else {
				// 	prevCurX, prevCurY := t.curx, t.cury
				// 	t.CursorRight() // Advance the cursor
				// 	if prevCurY == t.selection.EndLine && prevCurX == t.selection.EndCol {
				// 		t.selection.EndLine, t.selection.EndCol = t.cury, t.curx
				// 	} else {
				// 		t.selection.StartLine, t.selection.StartCol = t.cury, t.curx
				// 	}
				// }
			} else {
				t.selectMode = false
				t.SetCursor(t.cursor.Right())
				t.ScrollToCursor()
			}
		case tcell.KeyHome:
			cursLine, _ := t.cursor.GetLineCol()
			// TODO: go to first (non-whitespace) character on current line, if we are not already there
			// otherwise actually go to first (0) character of the line
			t.SetCursor(t.cursor.SetLineCol(cursLine, 0))
			t.ScrollToCursor()
		case tcell.KeyEnd:
			cursLine, _ := t.cursor.GetLineCol()
			t.SetCursor(t.cursor.SetLineCol(cursLine, math.MaxInt32)) // Max column
			t.ScrollToCursor()
		case tcell.KeyPgUp:
			_, cursCol := t.cursor.GetLineCol()
			t.SetCursor(t.cursor.SetLineCol(t.scrolly-t.height, cursCol)) // Go a page up
			t.ScrollToCursor()
		case tcell.KeyPgDn:
			_, cursCol := t.cursor.GetLineCol()
			t.SetCursor(t.cursor.SetLineCol(t.scrolly+t.height*2-1, cursCol)) // Go a page down
			t.ScrollToCursor()

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
