package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"io/fs"
	"io/ioutil"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/fivemoreminix/qedit/ui"
	"github.com/gdamore/tcell/v2"
)

var (
	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
	memprofile = flag.String("memprofile", "", "write memory profile to `file`")
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
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("Could not create CPU profile: ", err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("Could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}

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

	var closing bool
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
			if os.Args[i] == "-cpuprofile" || os.Args[i] == "-memprofile" {
				i++
				continue
			}

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

			textEdit := ui.NewTextEdit(&s, os.Args[i], bytes, &theme)
			textEdit.Dirty = dirty
			tabContainer.AddTab(os.Args[i], textEdit)
		}
	}

	menuBar = ui.NewMenuBar(&theme)

	fileMenu := ui.NewMenu("File", 0, &theme)

	fileMenu.AddItems([]ui.Item{&ui.ItemEntry{Name: "New File", Shortcut: "Ctrl+N", Callback: func() {
		textEdit := ui.NewTextEdit(&s, "", []byte{}, &theme) // No file path, no contents
		tabContainer.AddTab("noname", textEdit)
	}}, &ui.ItemEntry{Name: "Open...", Shortcut: "Ctrl+O", Callback: func() {
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

				textEdit := ui.NewTextEdit(&s, path, bytes, &theme)
				tabContainer.AddTab(path, textEdit)
			}
			dialog = nil // Hide the file selector

			changeFocus(tabContainer)
			tabContainer.FocusTab(tabContainer.GetTabCount()-1)
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
			},
		)
		changeFocus(dialog)
	}}, &ui.ItemEntry{Name: "Save", Shortcut: "Ctrl+S", Callback: func() {
		if tabContainer.GetTabCount() > 0 {
			tab := tabContainer.GetTab(tabContainer.GetSelectedTabIdx())
			te := tab.Child.(*ui.TextEdit)
			if len(te.FilePath) > 0 {
				f, err := os.OpenFile(te.FilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fs.ModePerm)
				if err != nil {
					panic(err)
				}
				defer f.Close()

				_, err = te.Buffer.WriteTo(f) // TODO: check count
				if err != nil {
					panic(fmt.Sprintf("Error occurred while writing buffer to file: %v", err))
				}

				te.Dirty = false
			}
			changeFocus(tabContainer)
		}
	}}, &ui.ItemEntry{Name: "Save As...", QuickChar: 5, Callback: func() {
		// TODO: implement a "Save as" dialog system, and show that when trying to save noname files
		callback := func(filePaths []string) {
			dialog = nil // Hide the file selector
			changeFocus(tabContainer)
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
			},
		)
		changeFocus(dialog)
	}}, &ui.ItemSeparator{}, &ui.ItemEntry{Name: "Close", Shortcut: "Ctrl+Q", Callback: func() {
		if tabContainer.GetTabCount() > 0 {
			tabContainer.RemoveTab(tabContainer.GetSelectedTabIdx())
		} else { // No tabs open; close the editor
			closing = true
		}
	}}})

	panelMenu := ui.NewMenu("Panel", 0, &theme)

	panelMenu.AddItems([]ui.Item{&ui.ItemEntry{Name: "Focus Up", QuickChar: -1, Shortcut: "Alt+Up", Callback: func() {

	}}, &ui.ItemEntry{Name: "Focus Down", QuickChar: -1, Shortcut: "Alt+Down", Callback: func() {

	}}, &ui.ItemEntry{Name: "Focus Left", QuickChar: -1, Shortcut: "Alt+Left", Callback: func() {

	}}, &ui.ItemEntry{Name: "Focus Right", QuickChar: -1, Shortcut: "Alt+Right", Callback: func() {

	}}, &ui.ItemSeparator{}, &ui.ItemEntry{Name: "Split Top", QuickChar: 6, Callback: func() {

	}}, &ui.ItemEntry{Name: "Split Bottom", QuickChar: 6, Callback: func() {

	}}, &ui.ItemEntry{Name: "Split Left", QuickChar: 6, Callback: func() {

	}}, &ui.ItemEntry{Name: "Split Right", QuickChar: 6, Callback: func() {

	}}, &ui.ItemSeparator{}, &ui.ItemEntry{Name: "Move", Shortcut: "Ctrl+M", Callback: func() {

	}}, &ui.ItemEntry{Name: "Resize", Shortcut: "Ctrl+R", Callback: func() {

	}}, &ui.ItemEntry{Name: "Float", Callback: func() {

	}}})

	editMenu := ui.NewMenu("Edit", 0, &theme)

	editMenu.AddItems([]ui.Item{&ui.ItemEntry{Name: "Cut", Shortcut: "Ctrl+X", Callback: func() {
		if tabContainer.GetTabCount() > 0 {
			tab := tabContainer.GetTab(tabContainer.GetSelectedTabIdx())
			te := tab.Child.(*ui.TextEdit)
			bytes := te.GetSelectedBytes()
			if len(bytes) > 0 { // If something is selected...
				te.Delete(false) // Delete the selection
				// TODO: better error handling within editor
				_ = ClipWrite(string(bytes)) // Add the selectedStr to clipboard
			}
			changeFocus(tabContainer)
		}
	}}, &ui.ItemEntry{Name: "Copy", Shortcut: "Ctrl+C", Callback: func() {
		if tabContainer.GetTabCount() > 0 {
			tab := tabContainer.GetTab(tabContainer.GetSelectedTabIdx())
			te := tab.Child.(*ui.TextEdit)
			bytes := te.GetSelectedBytes()
			if len(bytes) > 0 { // If there is something selected...
				_ = ClipWrite(string(bytes)) // Add selectedStr to clipboard
			}
			changeFocus(tabContainer)
		}
	}}, &ui.ItemEntry{Name: "Paste", Shortcut: "Ctrl+V", Callback: func() {
		if tabContainer.GetTabCount() > 0 {
			tab := tabContainer.GetTab(tabContainer.GetSelectedTabIdx())
			te := tab.Child.(*ui.TextEdit)

			contents, err := ClipRead()
			if err != nil {
				panic(err)
			}
			te.Insert(contents)

			changeFocus(tabContainer)
		}
	}}, &ui.ItemSeparator{}, &ui.ItemEntry{Name: "Select All", QuickChar: 7, Shortcut: "Ctrl+A", Callback: func() {

	}}, &ui.ItemEntry{Name: "Select Line", QuickChar: 7, Callback: func() {

	}}})

	searchMenu := ui.NewMenu("Search", 0, &theme)

	searchMenu.AddItems([]ui.Item{&ui.ItemEntry{Name: "Find and Replace...", Shortcut: "Ctrl+F", Callback: func() {
		s.Beep()
	}}, &ui.ItemEntry{Name: "Find in Directory...", QuickChar: 8, Callback: func() {

	}}, &ui.ItemSeparator{}, &ui.ItemEntry{Name: "Go to line...", Shortcut: "Ctrl+G", Callback: func() {

	}}})

	menuBar.AddMenu(fileMenu)
	menuBar.AddMenu(panelMenu)
	menuBar.AddMenu(editMenu)
	menuBar.AddMenu(searchMenu)

	changeFocus(tabContainer) // TabContainer is focused by default

	for !closing {
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
			if dialog == nil {
				if ev.Key() == tcell.KeyEscape {
					if focusedComponent == tabContainer {
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

	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal("Could not create memory profile: ", err)
		}
		defer f.Close()
		runtime.GC() // Get updated statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal("Could not write memory profile: ", err)
		}
	}
}
