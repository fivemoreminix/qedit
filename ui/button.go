package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
)

type Button struct {
	Text string
	Callback func()

	x, y int
	width, height int
	focused bool

	Theme *Theme
}

func NewButton(text string, theme *Theme, callback func()) *Button {
	return &Button{
		Text: text,
		Callback: callback,
		Theme: theme,
	}
}

func (b *Button) Draw(s tcell.Screen) {
	var str string
	if b.focused {
		str = fmt.Sprintf("🭬 %s 🭮", b.Text)
	} else {
		str = fmt.Sprintf("  %s  ", b.Text)
	}
	DrawStr(s, b.x, b.y, str, b.Theme.GetOrDefault("Button"))
}

func (b *Button) SetFocused(v bool) {
	b.focused = v
}

func (b *Button) SetTheme(theme *Theme) {
	b.Theme = theme
}

func (b *Button) GetPos() (int, int) {
	return b.x, b.y
}

func (b *Button) SetPos(x, y int) {
	b.x, b.y = x, y
}

func (b *Button) GetMinSize() (int, int) {
	return len(b.Text) + 4, 1
}

func (b *Button) GetSize() (int, int) {
	return b.GetMinSize()
}

func (b *Button) SetSize(width, height int) {}

func (b *Button) HandleEvent(event tcell.Event) bool {
	if b.focused {
		switch ev := event.(type) {
		case *tcell.EventKey:
			if ev.Key() == tcell.KeyEnter {
				if b.Callback != nil {
					b.Callback()
				}
			}
		default:
			return false
		}
		return true
	}
	return false
}
