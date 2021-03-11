package ui

import "github.com/gdamore/tcell/v2"

// An InputField is a single-line input box.
type InputField struct {
	Text string

	cursorPos     int
	scrollPos     int
	x, y          int
	width, height int
	focused       bool
	screen        *tcell.Screen

	Theme *Theme
}

func NewInputField(screen *tcell.Screen, placeholder string, theme *Theme) *InputField {
	return &InputField{
		Text:  placeholder,
		screen: screen,
		Theme: theme,
	}
}

func (f *InputField) GetCursorPos() int {
	return f.cursorPos
}

// SetCursorPos sets the cursor position offset. Offset is clamped to possible values.
// The InputField is scrolled to show the new cursor position.
func (f *InputField) SetCursorPos(offset int) {
	// Clamping
	if offset < 0 {
		offset = 0
	} else if offset > len(f.Text) {
		offset = len(f.Text)
	}

	// Scrolling
	if offset >= f.scrollPos+f.width-2 { // If cursor position is out of view to the right...
		f.scrollPos = offset - f.width+2 // Scroll just enough to view that column
	} else if offset < f.scrollPos { // If cursor position is out of view to the left...
		f.scrollPos = offset
	}

	f.cursorPos = offset
	if f.focused {
		(*f.screen).ShowCursor(f.x+offset-f.scrollPos+1, f.y)
	}
}

func (f *InputField) Delete(forward bool) {
	if forward {
		//if f.cursorPos 
	} else {

	}
}

func (f *InputField) Draw(s tcell.Screen) {
	style := f.Theme.GetOrDefault("InputField")

	DrawRect(s, f.x, f.y, f.width, f.height, ' ', style) // Draw background
	s.SetContent(f.x, f.y, '[', nil, style)
	s.SetContent(f.x+f.width-1, f.y, ']', nil, style)

	if len(f.Text) > 0 {
		endPos := f.scrollPos + Min(len(f.Text)-f.scrollPos, f.width-2)
		DrawStr(s, f.x+1, f.y, f.Text[f.scrollPos:endPos], style) // Draw text
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

func (f *InputField) SetTheme(theme *Theme) {
	f.Theme = theme
}

func (f *InputField) GetPos() (int, int) {
	return f.x, f.y
}

func (f *InputField) SetPos(x, y int) {
	f.x, f.y = x, y
}

func (f *InputField) GetMinSize() (int, int) {
	return 0, 0
}

func (f *InputField) GetSize() (int, int) {
	return f.width, f.height
}

func (f *InputField) SetSize(width, height int) {
	f.width, f.height = width, height
}

func (f *InputField) HandleEvent(event tcell.Event) bool {
	switch ev := event.(type) {
	case *tcell.EventKey:
		switch ev.Key() {
		case tcell.KeyLeft:
			f.SetCursorPos(f.cursorPos - 1)
		case tcell.KeyRight:
			f.SetCursorPos(f.cursorPos + 1)
		case tcell.KeyRune:
			ch := ev.Rune()
			f.Text += string(ch)
			f.SetCursorPos(f.cursorPos + 1)
		default:
			return false
		}
		return true
	}
	return false
}
