package ui

import (
	"strings"

	"github.com/gdamore/tcell/v2"
)

// A FileSelectorDialog is a WindowContainer with an input and buttons for selecting files.
// It can be used to open zero or more existing files, or select one non-existant file (for saving).
type FileSelectorDialog struct {
	MustExist           bool           // Whether the dialog should have a user select an existing file.
	FilesChosenCallback func([]string) // Returns slice of filenames selected. nil if user canceled.
	Theme *Theme

	title         string
	x, y          int
	width, height int
	focused       bool

	tabOrder    []Component
	tabOrderIdx int

	inputField    *InputField
	confirmButton *Button
	cancelButton  *Button
}

func NewFileSelectorDialog(screen *tcell.Screen, title string, mustExist bool, theme *Theme, filesChosenCallback func([]string), cancelCallback func()) *FileSelectorDialog {
	dialog := &FileSelectorDialog{
		MustExist:           mustExist,
		FilesChosenCallback: filesChosenCallback,
		Theme:               theme,
		title: title,
	}

	dialog.inputField = NewInputField(screen, []byte{}, theme.GetOrDefault("Window")) // Use window's theme for InputField
	dialog.confirmButton = NewButton("Confirm", theme, dialog.onConfirm)
	dialog.cancelButton = NewButton("Cancel", theme, cancelCallback)
	dialog.tabOrder = []Component{dialog.inputField, dialog.cancelButton, dialog.confirmButton}

	return dialog
}

// onConfirm is a callback called by the confirm button.
func (d *FileSelectorDialog) onConfirm() {
	if d.FilesChosenCallback != nil {
		files := strings.Split(string(d.inputField.Buffer), ",") // Split input by commas
		for i := range files {
			files[i] = strings.TrimSpace(files[i]) // Trim all strings in slice
		}
		d.FilesChosenCallback(files)
	}
}

func (d *FileSelectorDialog) SetCancelCallback(callback func()) {
	d.cancelButton.Callback = callback
}

func (d *FileSelectorDialog) SetTitle(title string) {
	d.title = title
}

func (d *FileSelectorDialog) Draw(s tcell.Screen) {
	DrawWindow(s, d.x, d.y, d.width, d.height, d.title, d.Theme)

	// Update positions of child components (dependent on size information that may not be available at SetPos() )
	btnWidth, _ := d.confirmButton.GetSize()
	d.confirmButton.SetPos(d.x+d.width-btnWidth-1, d.y+4) // Place "Ok" button on right, bottom

	d.inputField.Draw(s)
	d.confirmButton.Draw(s)
	d.cancelButton.Draw(s)
}

func (d *FileSelectorDialog) SetFocused(v bool) {
	d.focused = v
	d.tabOrder[d.tabOrderIdx].SetFocused(v)
}

func (d *FileSelectorDialog) SetTheme(theme *Theme) {
	d.Theme = theme
	d.inputField.SetStyle(theme.GetOrDefault("Window"))
	d.confirmButton.SetTheme(theme)
	d.cancelButton.SetTheme(theme)
}

func (d *FileSelectorDialog) GetPos() (int, int) {
	return d.x, d.y
}

func (d *FileSelectorDialog) SetPos(x, y int) {
	d.x, d.y = x, y
	d.inputField.SetPos(d.x+1, d.y+2)   // Center input field
	d.cancelButton.SetPos(d.x+1, d.y+4) // Place "Cancel" button on left, bottom
}

func (d *FileSelectorDialog) GetMinSize() (int, int) {
	return Max(len(d.title), 8) + 2, 6
}

func (d *FileSelectorDialog) GetSize() (int, int) {
	return d.width, d.height
}

func (d *FileSelectorDialog) SetSize(width, height int) {
	minX, minY := d.GetMinSize()
	d.width, d.height = Max(width, minX), Max(height, minY)

	d.inputField.SetSize(d.width-2, 1)
	d.cancelButton.SetSize(d.cancelButton.GetMinSize())
	d.confirmButton.SetSize(d.confirmButton.GetMinSize())
}

func (d *FileSelectorDialog) HandleEvent(event tcell.Event) bool {
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
