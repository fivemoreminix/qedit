package main

import (
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"

	"github.com/fivemoreminix/diesel/ui"
	"github.com/gdamore/tcell/v2"
)

var theme = ui.Theme{
	"StatusBar": tcell.Style{}.Foreground(tcell.ColorBlack).Background(tcell.ColorSilver),
}

var (
	menuBar      *ui.MenuBar
	tabContainer *ui.TabContainer
	dialog       ui.Component // nil if not present (has exclusive focus)

	focusedComponent ui.Component = nil
)

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

	tabContainer = ui.NewTabContainer(&theme)
	tabContainer.SetPos(0, 1)
	tabContainer.SetSize(sizex, sizey-2)

	_, err := ClipInitialize(ClipExternal)
	if err != nil {
		panic(err)
	}

	// Open files from command-line arguments
	if len(os.Args) > 1 {
		for i := 1; i < len(os.Args); i++ {
			_, err := os.Stat(os.Args[i])

			var dirty bool
			var bytes []byte

			if errors.Is(err, os.ErrNotExist) { // If the file does not exist...
				dirty = true
			} else { // If the file exists...
				file, err := os.Open(os.Args[i])
				if err != nil {
					panic("File could not be opened at path " + os.Args[i])
				}
				defer file.Close()

				bytes, err = ioutil.ReadAll(file)
				if err != nil {
					panic("Could not read all of " + os.Args[i])
				}
			}

			textEdit := ui.NewTextEdit(&s, os.Args[i], string(bytes), &theme)
			textEdit.Dirty = dirty
			tabContainer.AddTab(os.Args[i], textEdit)
		}
	}

	barFocused := false

	menuBar = ui.NewMenuBar(&theme)

	fileMenu := ui.NewMenu("_File", &theme)

	fileMenu.AddItems([]ui.Item{&ui.ItemEntry{Name: "_New File", Shortcut: 'n', Callback: func() {
		textEdit := ui.NewTextEdit(&s, "", "", &theme) // No file path, no contents
		tabContainer.AddTab("noname", textEdit)
	}}, &ui.ItemEntry{Name: "_Open...", Shortcut: 'o', Callback: func() {
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
			// TODO: free the dialog instead?
			dialog = nil // Hide the file selector

			changeFocus(tabContainer)
			barFocused = false
		}
		dialog = ui.NewFileSelectorDialog(
			&s,
			"Comma-separated files or a directory",
			true,
			&theme,
			callback,
			func() { // Dialog is canceled
				dialog = nil
				changeFocus(tabContainer)
				barFocused = false
			},
		)
		changeFocus(dialog)
	}}, &ui.ItemEntry{Name: "_Save", Shortcut: 's', Callback: func() {
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

				te.Dirty = false
			}
			changeFocus(tabContainer)
			barFocused = false
		}
	}}, &ui.ItemEntry{Name: "Save _As...", Callback: func() {
		// TODO: implement a "Save as" dialog system, and show that when trying to save noname files
		callback := func(filePaths []string) {
			dialog = nil // Hide the file selector
			changeFocus(tabContainer)
			barFocused = false
		}

		dialog = ui.NewFileSelectorDialog(
			&s,
			"Select a file to overwrite",
			false,
			&theme,
			callback,
			func() { // Dialog canceled
				dialog = nil
				changeFocus(tabContainer)
				barFocused = false
			},
		)
		changeFocus(dialog)
	}}, &ui.ItemSeparator{}, &ui.ItemEntry{Name: "_Close", Shortcut: 'q', Callback: func() {
		if tabContainer.GetTabCount() > 0 {
			tabContainer.RemoveTab(tabContainer.GetSelectedTabIdx())
		} else { // No tabs open; close the editor
			s.Fini()
			os.Exit(0)
		}
	}}})

	panelMenu := ui.NewMenu("_Panel", &theme)

	panelMenu.AddItems([]ui.Item{&ui.ItemEntry{Name: "_Up", Callback: func() {

	}}, &ui.ItemEntry{Name: "_Down", Callback: func() {

	}}, &ui.ItemEntry{Name: "_Left", Callback: func() {

	}}, &ui.ItemEntry{Name: "_Right", Callback: func() {

	}}, &ui.ItemSeparator{}, &ui.ItemEntry{Name: "Split _Left", Callback: func() {

	}}, &ui.ItemEntry{Name: "Split _Right", Callback: func() {

	}}, &ui.ItemSeparator{}, &ui.ItemEntry{Name: "_Move", Shortcut: 'm', Callback: func() {

	}}, &ui.ItemEntry{Name: "_Resize", Shortcut: 'r', Callback: func() {

	}}, &ui.ItemEntry{Name: "_Float", Callback: func() {

	}}})

	editMenu := ui.NewMenu("_Edit", &theme)

	editMenu.AddItems([]ui.Item{&ui.ItemEntry{Name: "_Cut", Shortcut: 'x', Callback: func() {
		if tabContainer.GetTabCount() > 0 {
			tab := tabContainer.GetTab(tabContainer.GetSelectedTabIdx())
			te := tab.Child.(*ui.TextEdit)
			selectedStr := te.GetSelectedString()
			if selectedStr != "" { // If something is selected...
				te.Delete(false) // Delete the selection
				// TODO: better error handling within editor
				_ = ClipWrite(selectedStr) // Add the selectedStr to clipboard
			}
			changeFocus(tabContainer)
			barFocused = false
		}
	}}, &ui.ItemEntry{Name: "_Copy", Shortcut: 'c', Callback: func() {
		if tabContainer.GetTabCount() > 0 {
			tab := tabContainer.GetTab(tabContainer.GetSelectedTabIdx())
			te := tab.Child.(*ui.TextEdit)
			selectedStr := te.GetSelectedString()
			if selectedStr != "" { // If there is something selected...
				_ = ClipWrite(selectedStr) // Add selectedStr to clipboard
			}
			changeFocus(tabContainer)
			barFocused = false
		}
	}}, &ui.ItemEntry{Name: "_Paste", Shortcut: 'v', Callback: func() {
		if tabContainer.GetTabCount() > 0 {
			tab := tabContainer.GetTab(tabContainer.GetSelectedTabIdx())
			te := tab.Child.(*ui.TextEdit)

			contents, err := ClipRead()
			if err != nil {
				panic(err)
			}
			te.Insert(contents)

			changeFocus(tabContainer)
			barFocused = false
		}
	}}, &ui.ItemSeparator{}, &ui.ItemEntry{Name: "Select _All", Shortcut: 'a', Callback: func() {

	}}, &ui.ItemEntry{Name: "Select _Line", Callback: func() {

	}}})

	searchMenu := ui.NewMenu("_Search", &theme)

	searchMenu.AddItems([]ui.Item{&ui.ItemEntry{Name: "New", Callback: func() {
		s.Beep()
	}}})

	menuBar.AddMenu(fileMenu)
	menuBar.AddMenu(panelMenu)
	menuBar.AddMenu(editMenu)
	menuBar.AddMenu(searchMenu)

	changeFocus(tabContainer) // TabContainer is focused by default

	for {
		s.Clear()

		// Draw background (grey and black checkerboard)
		ui.DrawRect(s, 0, 0, sizex, sizey, '▚', tcell.Style{}.Foreground(tcell.ColorGrey).Background(tcell.ColorBlack))

		if tabContainer.GetTabCount() > 0 { // Draw the tab container only if a tab is open
			tabContainer.Draw(s)
		}
		menuBar.Draw(s) // Always draw the menu bar

		if dialog != nil {
			// Update fileSelector dialog pos and size
			diagMinX, diagMinY := dialog.GetMinSize()
			dialog.SetSize(diagMinX, diagMinY)
			dialog.SetPos(sizex/2-diagMinX/2, sizey/2-diagMinY/2) // Center

			dialog.Draw(s)
		}

		// Draw statusbar
		ui.DrawRect(s, 0, sizey-1, sizex, 1, ' ', theme["StatusBar"])
		if tabContainer.GetTabCount() > 0 {
			focusedTab := tabContainer.GetTab(tabContainer.GetSelectedTabIdx())
			te := focusedTab.Child.(*ui.TextEdit)

			var delim string
			if te.IsCRLF {
				delim = "CRLF"
			} else {
				delim = "LF"
			}

			line, col := te.GetLineCol()

			var tabs string
			if te.UseHardTabs {
				tabs = "Tabs: Hard"
			} else {
				tabs = "Tabs: Spaces"
			}

			str := fmt.Sprintf(" Filetype: %s  %d, %d  %s  %s", "None", line+1, col+1, delim, tabs)
			ui.DrawStr(s, 0, sizey-1, str, theme["StatusBar"])
		}

		s.Show()

		switch ev := s.PollEvent().(type) {
		case *tcell.EventResize:
			sizex, sizey = s.Size()

			menuBar.SetSize(sizex, 1)
			tabContainer.SetSize(sizex, sizey-2)

			s.Sync() // Redraw everything
		case *tcell.EventKey:
			// On Escape, we change focus between editor and the MenuBar.
			if dialog == nil { // While no dialog is present...
				if ev.Key() == tcell.KeyEscape {
					barFocused = !barFocused
					if barFocused {
						changeFocus(menuBar)
					} else {
						changeFocus(tabContainer)
					}
				}

				if ev.Modifiers() & tcell.ModCtrl != 0 {
					handled := menuBar.HandleEvent(ev)
					if handled {
						continue // Avoid passing the event to the focusedComponent
					}
				}
			}

			focusedComponent.HandleEvent(ev)
		}
	}
}
