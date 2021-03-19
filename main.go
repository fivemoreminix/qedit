package main

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"

	"github.com/fivemoreminix/diesel/ui"
	"github.com/gdamore/tcell/v2"
)

var theme = ui.Theme{}

var focusedComponent ui.Component = nil

func changeFocus(to ui.Component) {
	if focusedComponent != nil {
		focusedComponent.SetFocused(false)
	}
	focusedComponent = to
	to.SetFocused(true)
}

func main() {
	s, e := tcell.NewScreen()
	if e != nil {
		fmt.Fprintf(os.Stderr, "%v\n", e)
		os.Exit(1)
	}
	if e := s.Init(); e != nil {
		fmt.Fprintf(os.Stderr, "%v\n", e)
		os.Exit(1)
	}
	defer s.Fini() // Useful for handling panics

	sizex, sizey := s.Size()

	tabContainer := ui.NewTabContainer(&theme)
	tabContainer.SetPos(0, 1)
	tabContainer.SetSize(sizex, sizey-1)

	_, err := ClipInitialize(ClipExternal)
	if err != nil {
		panic(err)
	}

	// Load files from command-line arguments
	if len(os.Args) > 1 {
		for _, path := range os.Args[1:] {
			file, err := os.Open(path)
			if err != nil {
				panic("File could not be opened at path " + path)
			}
			defer file.Close()

			bytes, err := ioutil.ReadAll(file)
			if err != nil {
				panic("Could not read all of " + path)
			}

			textEdit := ui.NewTextEdit(&s, path, string(bytes), &theme)
			tabContainer.AddTab(path, textEdit)
		}
	}

	var fileSelector *ui.FileSelectorDialog // if nil, we don't draw it

	bar := ui.NewMenuBar(&theme)

	barFocused := false

	fileMenu := ui.NewMenu("File", &theme)

	fileMenu.AddItems([]ui.Item{&ui.ItemEntry{Name: "New File", Callback: func() {
		textEdit := ui.NewTextEdit(&s, "", "", &theme) // No file path, no contents
		tabContainer.AddTab("noname", textEdit)
	}}, &ui.ItemEntry{Name: "Open...", Callback: func() {
		callback := func(filePaths []string) {
			for _, path := range filePaths {
				file, err := os.Open(path)
				if err != nil {
					panic("Could not open file at path " + path)
				}
				defer file.Close()

				bytes, err := ioutil.ReadAll(file)
				if err != nil {
					panic("Could not read all of file")
				}

				textEdit := ui.NewTextEdit(&s, path, string(bytes), &theme)
				tabContainer.AddTab(path, textEdit)
			}
			fileSelector = nil // Hide the file selector
			changeFocus(tabContainer)
			barFocused = false
		}
		fileSelector = ui.NewFileSelectorDialog(
			&s,
			"Comma-separated files or a directory",
			true,
			&theme,
			callback,
			func() { // Dialog is canceled
				fileSelector = nil
				changeFocus(bar)
				barFocused = true
			},
		)
		changeFocus(fileSelector)
	}}, &ui.ItemEntry{Name: "Save", Callback: func() {
		if tabContainer.GetTabCount() > 0 {
			tab := tabContainer.GetTab(tabContainer.GetSelectedTabIdx())
			te := tab.Child.(*ui.TextEdit)
			if len(te.FilePath) > 0 {
				contents := te.String()

				// Write the contents into the file, creating one if it does
				// not exist.
				err := ioutil.WriteFile(te.FilePath, []byte(contents), fs.ModePerm)
				if err != nil {
					panic("Could not write file at path " + te.FilePath)
				}
			}
		}
	}}, &ui.ItemEntry{Name: "Save As...", Callback: func() {
		// TODO: implement a "Save as" dialog system, and show that when trying to save noname files
		callback := func(filePaths []string) {
			fileSelector = nil // Hide the file selector
		}

		fileSelector = ui.NewFileSelectorDialog(
			&s,
			"Select a file to overwrite",
			false,
			&theme,
			callback,
			func() { // Dialog canceled
				fileSelector = nil
				changeFocus(bar)
			},
		)
		changeFocus(fileSelector)
	}}, &ui.ItemSeparator{}, &ui.ItemEntry{Name: "Exit", Callback: func() {
		s.Fini()
		os.Exit(0)
	}}})

	editMenu := ui.NewMenu("Edit", &theme)

	editMenu.AddItems([]ui.Item{&ui.ItemEntry{Name: "Cut", Callback: func() {
		if tabContainer.GetTabCount() > 0 {
			tab := tabContainer.GetTab(tabContainer.GetSelectedTabIdx())
			te := tab.Child.(*ui.TextEdit)
			selectedStr := te.GetSelectedString()
			if selectedStr != "" { // If something is selected...
				te.Delete(false) // Delete the selection
				// TODO: better error handling within editor
				_ = ClipWrite(selectedStr) // Add the selectedStr to clipboard
			}
		}
	}}, &ui.ItemEntry{Name: "Copy", Callback: func() {
		if tabContainer.GetTabCount() > 0 {
			tab := tabContainer.GetTab(tabContainer.GetSelectedTabIdx())
			te := tab.Child.(*ui.TextEdit)
			selectedStr := te.GetSelectedString()
			if selectedStr != "" { // If there is something selected...
				_ = ClipWrite(selectedStr) // Add selectedStr to clipboard
			}
		}
	}}, &ui.ItemEntry{Name: "Paste", Callback: func() {
		if tabContainer.GetTabCount() > 0 {
			tab := tabContainer.GetTab(tabContainer.GetSelectedTabIdx())
			te := tab.Child.(*ui.TextEdit)

			contents, err := ClipRead()
			if err != nil {
				panic(err)
			}
			te.Insert(contents)
		}
	}}})

	searchMenu := ui.NewMenu("Search", &theme)

	searchMenu.AddItems([]ui.Item{&ui.ItemEntry{Name: "New", Callback: func() {
		s.Beep()
	}}})

	bar.AddMenu(fileMenu)
	bar.AddMenu(editMenu)
	bar.AddMenu(searchMenu)

	changeFocus(tabContainer) // TabContainer is focused by default

main_loop:
	for {
		s.Clear()

		// Draw background (grey and black checkerboard)
		ui.DrawRect(s, 0, 0, sizex, sizey, '▚', tcell.Style{}.Foreground(tcell.ColorGrey).Background(tcell.ColorBlack))

		if tabContainer.GetTabCount() > 0 { // Draw the tab container only if a tab is open
			tabContainer.Draw(s)
		}
		bar.Draw(s) // Always draw the menu bar

		if fileSelector != nil {
			// Update fileSelector dialog pos and size
			diagMinX, diagMinY := fileSelector.GetMinSize()
			fileSelector.SetSize(diagMinX, diagMinY)
			fileSelector.SetPos(sizex/2-diagMinX/2, sizey/2-diagMinY/2) // Center

			fileSelector.Draw(s)
		}

		s.Show()

		switch ev := s.PollEvent().(type) {
		case *tcell.EventResize:
			sizex, sizey = s.Size()

			bar.SetSize(sizex, 1)
			tabContainer.SetSize(sizex, sizey-1)

			s.Sync() // Redraw everything
		case *tcell.EventKey:
			// On Escape, we change focus between editor and the MenuBar.
			if fileSelector == nil { // While no dialog is present...
				if ev.Key() == tcell.KeyEscape {
					barFocused = !barFocused
					if barFocused {
						changeFocus(bar)
					} else {
						changeFocus(tabContainer)
					}
				}
				// Ctrl + Q is a shortcut to exit
				if ev.Key() == tcell.KeyCtrlQ { // TODO: replace with shortcut keys in menus
					break main_loop
				}
			}

			focusedComponent.HandleEvent(ev)
		}
	}
}
