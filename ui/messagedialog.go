package ui

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
)

type MessageDialogKind uint8

const (
	MessageKindNormal MessageDialogKind = iota
	MessageKindWarning
	MessageKindError
)

// Index of messageDialogKindTitles is any MessageDialogKind.
var messageDialogKindTitles [3]string = [3]string{
	"Message",
	"Warning!",
	"Error!",
}

type MessageDialog struct {
	Title    string
	Kind     MessageDialogKind
	Callback func(string)

	message        string
	messageWrapped string

	x, y          int
	width, height int
	focused       bool
	theme         *Theme

	buttons     []*Button
	selectedIdx int
}

func NewMessageDialog(title string, message string, kind MessageDialogKind, options []string, theme *Theme, callback func(string)) *MessageDialog {
	if title == "" {
		title = messageDialogKindTitles[kind] // Use default title
	}

	if options == nil || len(options) == 0 {
		options = []string{"OK"}
	}

	dialog := MessageDialog{
		Title:    title,
		Kind:     kind,
		Callback: callback,

		theme: theme,
	}

	dialog.buttons = make([]*Button, len(options))
	for i := range options {
		dialog.buttons[i] = NewButton(options[i], theme, func() {
			if dialog.Callback != nil {
				dialog.Callback(dialog.buttons[dialog.selectedIdx].Text)
			}
		})
	}

	// Set the dialog's size to its minimum size
	dialog.SetSize(0, 0)
	dialog.SetMessage(message)

	return &dialog
}

func (d *MessageDialog) SetMessage(message string) {
	d.message = message
	d.messageWrapped = runewidth.Wrap(message, d.width-2)
	// Update height:
	_, minHeight := d.GetMinSize()
	d.height = Max(d.height, minHeight)
}

func (d *MessageDialog) Draw(s tcell.Screen) {
	DrawWindow(s, d.x, d.y, d.width, d.height, d.Title, d.theme)

	// DrawStr will handle '\n' characters and wrap for us.
	DrawStr(s, d.x+1, d.y+2, d.messageWrapped, d.theme.GetOrDefault("Window"))

	col := d.width // Start from the right side
	for i := range d.buttons {
		width, _ := d.buttons[i].GetSize()
		col -= width + 1 // Move left enough for each button (1 for padding)
		d.buttons[i].SetPos(d.x+col, d.y+d.height-2)
		d.buttons[i].Draw(s)
	}
}

func (d *MessageDialog) SetFocused(v bool) {
	d.focused = v
	d.buttons[d.selectedIdx].SetFocused(v)
}

func (d *MessageDialog) SetTheme(theme *Theme) {
	d.theme = theme
	for i := range d.buttons {
		d.buttons[i].SetTheme(theme)
	}
}

func (d *MessageDialog) GetPos() (int, int) {
	return d.x, d.y
}

func (d *MessageDialog) SetPos(x, y int) {
	d.x, d.y = x, y
}

func (d *MessageDialog) GetMinSize() (int, int) {
	lines := strings.Count(d.messageWrapped, "\n") + 1

	return Max(len(d.Title)+2, 30), 2 + lines + 2
}

func (d *MessageDialog) GetSize() (int, int) {
	return d.width, d.height
}

func (d *MessageDialog) SetSize(width, height int) {
	minWidth, minHeight := d.GetMinSize()
	d.width, d.height = Max(width, minWidth), Max(height, minHeight)
}

func (d *MessageDialog) HandleEvent(event tcell.Event) bool {
	return d.buttons[d.selectedIdx].HandleEvent(event)
}
