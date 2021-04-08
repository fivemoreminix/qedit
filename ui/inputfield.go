package ui

import (
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
)

// An InputField is a single-line input box.
type InputField struct {
	Buffer []byte

	cursorPos     int
	scrollPos     int
	screen        *tcell.Screen
	style         tcell.Style

	baseComponent
}

func NewInputField(screen *tcell.Screen, placeholder []byte, style tcell.Style) *InputField {
	return &InputField{
		Buffer: append(make([]byte, 0, Max(len(placeholder), 32)), placeholder...),
		screen: screen,
		style:  style,
	}
}

func (f *InputField) String() string {
	return string(f.Buffer)
}

func (f *InputField) GetCursorPos() int {
	return f.cursorPos
}

// SetCursorPos sets the cursor position offset. Offset is clamped to possible values.
// The InputField is scrolled to show the new cursor position. The offset is in runes.
func (f *InputField) SetCursorPos(offset int) {
	// Clamping
	if offset < 0 {
		offset = 0
	} else if runes := utf8.RuneCount(f.Buffer); offset > runes {
		offset = runes
	}

	// Scrolling
	if offset >= f.scrollPos+f.width-2 { // If cursor position is out of view to the right...
		f.scrollPos = offset - f.width + 2 // Scroll just enough to view that column
	} else if offset < f.scrollPos { // If cursor position is out of view to the left...
		f.scrollPos = offset
	}

	f.cursorPos = offset
	if f.focused {
		(*f.screen).ShowCursor(f.x+offset-f.scrollPos+1, f.y)
	}
}

func (f *InputField) runeIdxToByteIdx(idx int) int {
	var i int
	for idx > 0 {
		_, size := utf8.DecodeRune(f.Buffer[i:])
		i += size
		idx--
	}
	return i
}

func (f *InputField) Insert(contents []byte) {
	f.Buffer = f.insert(f.Buffer, f.runeIdxToByteIdx(f.cursorPos), contents...)
	f.SetCursorPos(f.cursorPos + utf8.RuneCount(contents))
}

// Efficient slice inserting from Slice Tricks.
func (f *InputField) insert(dst []byte, at int, src ...byte) []byte {
	if n := len(dst) + len(src); n <= cap(dst) {
		dstn := dst[:n]
		copy(dstn[at+len(src):], dst[at:])
		copy(dstn[at:], src)
		return dstn
	}
	dstn := make([]byte, len(dst)+len(src))
	copy(dstn, dst[:at])
	copy(dstn[at:], src)
	copy(dstn[at+len(src):], dst[at:])
	return dstn
}

func (f *InputField) Delete(forward bool) {
	if forward {
		if f.cursorPos < utf8.RuneCount(f.Buffer) { // If the cursor is not at the end...
			f.Buffer = f.delete(f.Buffer, f.runeIdxToByteIdx(f.cursorPos))
		}
	} else {
		if f.cursorPos > 0 { // If the cursor is not at the beginning...
			f.SetCursorPos(f.cursorPos - 1)
			f.Buffer = f.delete(f.Buffer, f.runeIdxToByteIdx(f.cursorPos))
		}
	}
}

func (f *InputField) delete(dst []byte, at int) []byte {
	copy(dst[at:], dst[at+1:])
	dst[len(dst)-1] = 0
	dst = dst[:len(dst)-1]
	return dst
}

func (f *InputField) Draw(s tcell.Screen) {
	s.SetContent(f.x, f.y, '[', nil, f.style)
	s.SetContent(f.x+f.width-1, f.y, ']', nil, f.style)

	fg, bg, attr := f.style.Decompose()
	invertedStyle := tcell.Style{}.Foreground(bg).Background(fg).Attributes(attr)

	var byteIdx int
	var runeIdx int

	// Scrolling
	for byteIdx < len(f.Buffer) && runeIdx < f.scrollPos {
		_, size := utf8.DecodeRune(f.Buffer[byteIdx:])
		byteIdx += size
		runeIdx++
	}

	for i := 0; i < f.width-2; i++ { // For each column between [ and ]
		if byteIdx < len(f.Buffer) {
			// Draw the rune
			r, size := utf8.DecodeRune(f.Buffer[byteIdx:])

			s.SetContent(f.x+1+i, f.y, r, nil, invertedStyle)

			byteIdx += size
			runeIdx++
		} else {
			// Draw a '.'
			s.SetContent(f.x+1+i, f.y, '.', nil, f.style)
		}
	}

	// Update cursor
	f.SetCursorPos(f.cursorPos)
}

func (f *InputField) SetFocused(v bool) {
	f.focused = v
	if v {
		f.SetCursorPos(f.cursorPos)
	} else {
		(*f.screen).HideCursor()
	}
}

func (f *InputField) SetStyle(style tcell.Style) {
	f.style = style
}

func (f *InputField) HandleEvent(event tcell.Event) bool {
	switch ev := event.(type) {
	case *tcell.EventKey:
		switch ev.Key() {
		// Cursor movement
		case tcell.KeyLeft:
			f.SetCursorPos(f.cursorPos - 1)
		case tcell.KeyRight:
			f.SetCursorPos(f.cursorPos + 1)

		// Deleting
		case tcell.KeyBackspace:
			fallthrough
		case tcell.KeyBackspace2:
			f.Delete(false)
		case tcell.KeyDelete:
			f.Delete(true)

		// Inserting
		case tcell.KeyRune:
			ch := ev.Rune()
			if bytesLen := utf8.RuneLen(ch); bytesLen > 0 {
				bytes := make([]byte, bytesLen)
				utf8.EncodeRune(bytes, ch)
				f.Insert(bytes)
			}
		default:
			return false
		}
		return true
	}
	return false
}
