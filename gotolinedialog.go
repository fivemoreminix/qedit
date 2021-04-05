package main

import (
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/fivemoreminix/qedit/ui"
)

type GotoLineDialog struct {
	LineChosenCallback func(int)

	x, y          int
	width, height int
	focused       bool
	screen        *tcell.Screen
	theme         *ui.Theme

	tabOrder    []ui.Component
	tabOrderIdx int

	inputField   *ui.InputField
	acceptButton *ui.Button
	cancelButton *ui.Button
}

func NewGotoLineDialog(s *tcell.Screen, theme *ui.Theme, lineChosenCallback func(int), cancelCallback func()) *GotoLineDialog {
	dialog := &GotoLineDialog{
		LineChosenCallback: lineChosenCallback,
		screen:       s,
		theme:        theme,
	}

	dialog.inputField = ui.NewInputField(s, nil, theme.GetOrDefault("Window"))
	dialog.acceptButton = ui.NewButton("Go", theme, dialog.onConfirm) // TODO: callback
	dialog.cancelButton = ui.NewButton("Cancel", theme, cancelCallback)
	dialog.tabOrder = []ui.Component{dialog.inputField, dialog.cancelButton, dialog.acceptButton}

	return dialog
}

func (d *GotoLineDialog) onConfirm() {
	if d.LineChosenCallback != nil {
		if len(d.inputField.Buffer) > 0 {
			num, err := strconv.Atoi(strings.TrimSpace(string(d.inputField.Buffer)))
			if err == nil {
				d.LineChosenCallback(num)
			}
		}
	}
}

func (d *GotoLineDialog) Draw(s tcell.Screen) {
	ui.DrawWindow(s, d.x, d.y, d.width, d.height, "Go to line", d.theme)

	btnWidth, _ := d.acceptButton.GetSize()
	d.acceptButton.SetPos(d.x+d.width-btnWidth-1, d.y+4) // Place "Ok" button on right, bottom

	d.inputField.Draw(s)
	d.acceptButton.Draw(s)
	d.cancelButton.Draw(s)
}

func (d *GotoLineDialog) SetFocused(v bool) {
	d.focused = v
	d.tabOrder[d.tabOrderIdx].SetFocused(v)
}

func (d *GotoLineDialog) SetTheme(theme *ui.Theme) {
	d.theme = theme
	d.inputField.SetStyle(theme.GetOrDefault("Window"))
	d.acceptButton.SetTheme(theme)
	d.cancelButton.SetTheme(theme)
}

func (d *GotoLineDialog) GetPos() (int, int) {
	return d.x, d.y
}

func (d *GotoLineDialog) SetPos(x, y int) {
	d.x, d.y = x, y
	d.inputField.SetPos(d.x+1, d.y+2)   // Center input field
	d.cancelButton.SetPos(d.x+1, d.y+4) // Place "Cancel" button on left, bottom
}

func (d *GotoLineDialog) GetMinSize() (int, int) {
	return 20, 6
}

func (d *GotoLineDialog) GetSize() (int, int) {
	return d.width, d.height
}

func (d *GotoLineDialog) SetSize(width, height int) {
	minX, minY := d.GetMinSize()
	d.width, d.height = ui.Max(width, minX), ui.Max(height, minY)

	d.inputField.SetSize(d.width-2, 1)
	d.cancelButton.SetSize(d.cancelButton.GetMinSize())
	d.acceptButton.SetSize(d.acceptButton.GetMinSize())
}

func (d *GotoLineDialog) HandleEvent(event tcell.Event) bool {
	switch ev := event.(type) {
	case *tcell.EventKey:
		switch ev.Key() {
		case tcell.KeyTab:
			d.tabOrder[d.tabOrderIdx].SetFocused(false)

			d.tabOrderIdx++
			if d.tabOrderIdx >= len(d.tabOrder) {
				d.tabOrderIdx = 0
			}

			d.tabOrder[d.tabOrderIdx].SetFocused(true)

			return true
		case tcell.KeyEsc:
			if d.cancelButton.Callback != nil {
				d.cancelButton.Callback()
			}
			return true
		case tcell.KeyEnter:
			if d.tabOrder[d.tabOrderIdx] == d.inputField {
				d.onConfirm()
				return true
			}
		}
	}
	return d.tabOrder[d.tabOrderIdx].HandleEvent(event)
}
