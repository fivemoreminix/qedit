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
	screen tcell.Screen

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

func showErrorDialog(title string, message string, callback func()) {
	dialog = ui.NewMessageDialog(title, message, ui.MessageKindError, nil, &theme, func(string) {
		if callback != nil {
			callback()
		} else {
			dialog = nil
			changeFocus(tabContainer) // Default behavior: focus tabContainer
		}
	})
	changeFocus(dialog)
}

// Shows the Save As... dialog for saving unnamed files
func saveAs() {
	callback := func(filePaths []string) {
		tab := tabContainer.GetTab(tabContainer.GetSelectedTabIdx())
		te := tab.Child.(*ui.TextEdit)

		// If we got the callback, it is safe to assume there are one or more files
		f, err := os.OpenFile(filePaths[0], os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fs.ModePerm)
		if err != nil {
			showErrorDialog("Could not open file for writing", fmt.Sprintf("File at %#v could not be opened with write permissions. Maybe another program has it open? %v", filePaths[0], err), nil)
			return
		}
		defer f.Close()

		_, err = te.Buffer.WriteTo(f)
		if err != nil {
			showErrorDialog("Failed to write to file", fmt.Sprintf("File at %#v was opened for writing, but an error occurred while writing the buffer. %v", filePaths[0], err), nil)
			return
		}
		te.Dirty = false

		dialog = nil // Hide the file selector
		changeFocus(tabContainer)
		tab.Name = filePaths[0]
	}

	dialog = ui.NewFileSelectorDialog(
		&screen,
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

	screen, err := tcell.NewScreen()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	if err := screen.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	defer screen.Fini() // Useful for handling panics

	var closing bool
	sizex, sizey := screen.Size()

	tabContainer = ui.NewTabContainer(&theme)
	tabContainer.SetPos(0, 1)
	tabContainer.SetSize(sizex, sizey-2)

	changeFocus(tabContainer) // tabContainer focused by default

	// Open files from command-line arguments
	if flag.NArg() > 0 {
		for i := 0; i < flag.NArg(); i++ {
			arg := flag.Arg(i)
			_, err := os.Stat(arg)

			var dirty bool
			var bytes []byte

			if errors.Is(err, os.ErrNotExist) { // If the file does not exist...
				dirty = true
			} else { // If the file exists...
				file, err := os.Open(arg)
				if err != nil {
					showErrorDialog("File could not be opened", fmt.Sprintf("File at %#v could not be opened. %v", arg, err), nil)
					continue
				}
				defer file.Close()

				bytes, err = ioutil.ReadAll(file)
				if err != nil {
					showErrorDialog("Could not read file", fmt.Sprintf("File at %#v was opened, but could not be read. %v", arg, err), nil)
					continue
				}
			}

			textEdit := ui.NewTextEdit(&screen, arg, bytes, &theme)
			textEdit.Dirty = dirty
			tabContainer.AddTab(arg, textEdit)
		}
		tabContainer.SetFocused(true) // Lets any opened TextEdit component know to be focused
	}

	_, err = ClipInitialize(ClipExternal)
	if err != nil {
		showErrorDialog("Error Initializing Clipboard", fmt.Sprintf("%v\n\nAn internal clipboard will be used, instead.", err), nil)
	}

	menuBar = ui.NewMenuBar(&theme)

	fileMenu := ui.NewMenu("File", 0, &theme)

	fileMenu.AddItems([]ui.Item{&ui.ItemEntry{Name: "New File", Shortcut: "Ctrl+N", Callback: func() {
		textEdit := ui.NewTextEdit(&screen, "", []byte{}, &theme) // No file path, no contents
		tabContainer.AddTab("noname", textEdit)

		changeFocus(tabContainer)
		tabContainer.FocusTab(tabContainer.GetTabCount()-1)
	}}, &ui.ItemEntry{Name: "Open...", Shortcut: "Ctrl+O", Callback: func() {
		callback := func(filePaths []string) {
			var errOccurred bool
			for _, path := range filePaths {
				file, err := os.Open(path)
				if err != nil {
					showErrorDialog("File could not be opened", fmt.Sprintf("File at %#v could not be opened. %v", path, err), nil)
					errOccurred = true
					continue
				}
				defer file.Close()

				bytes, err := ioutil.ReadAll(file)
				if err != nil {
					showErrorDialog("Could not read file", fmt.Sprintf("File at %#v was opened, but could not be read. %v", path, err), nil)
					errOccurred = true
					continue
				}

				textEdit := ui.NewTextEdit(&screen, path, bytes, &theme)
				tabContainer.AddTab(path, textEdit)
			}

			if !errOccurred { // Prevent hiding the error dialog
				dialog = nil // Hide the file selector
				changeFocus(tabContainer)
				if tabContainer.GetTabCount() > 0 {
					tabContainer.FocusTab(tabContainer.GetTabCount()-1)
				}
			}
		}
		dialog = ui.NewFileSelectorDialog(
			&screen,
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
			if te.FilePath != "" {
				f, err := os.OpenFile(te.FilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fs.ModePerm)
				if err != nil {
					showErrorDialog("Could not open file for writing", fmt.Sprintf("File at %#v could not be opened with write permissions. Maybe another program has it open? %v", te.FilePath, err), nil)
					return
				}
				defer f.Close()

				_, err = te.Buffer.WriteTo(f) // TODO: check count
				if err != nil {
					showErrorDialog("Failed to write to file", fmt.Sprintf("File at %#v was opened for writing, but an error occurred while writing the buffer. %v", te.FilePath, err), nil)
					return
				}
				te.Dirty = false

				changeFocus(tabContainer)
			} else {
				saveAs()
			}
		}
	}}, &ui.ItemEntry{Name: "Save As...", QuickChar: 5, Callback: saveAs}, &ui.ItemSeparator{},
	&ui.ItemEntry{Name: "Close", Shortcut: "Ctrl+Q", Callback: func() {
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
			var err error
			if len(bytes) > 0 { // If something is selected...
				err = ClipWrite(string(bytes)) // Add the selectedStr to clipboard
				if err != nil {
					showErrorDialog("Clipboard Failure", fmt.Sprintf("%v", err), nil)
				}
				te.Delete(false) // Delete selection
			}
			if err == nil { // Prevent hiding error dialog
				changeFocus(tabContainer)
			}
		}
	}}, &ui.ItemEntry{Name: "Copy", Shortcut: "Ctrl+C", Callback: func() {
		if tabContainer.GetTabCount() > 0 {
			tab := tabContainer.GetTab(tabContainer.GetSelectedTabIdx())
			te := tab.Child.(*ui.TextEdit)
			bytes := te.GetSelectedBytes()
			var err error
			if len(bytes) > 0 { // If there is something selected...
				err = ClipWrite(string(bytes)) // Add selectedStr to clipboard
				if err != nil {
					showErrorDialog("Clipboard Failure", fmt.Sprintf("%v", err), nil)
				}
			}
			if err == nil {
				changeFocus(tabContainer)
			}
		}
	}}, &ui.ItemEntry{Name: "Paste", Shortcut: "Ctrl+V", Callback: func() {
		if tabContainer.GetTabCount() > 0 {
			tab := tabContainer.GetTab(tabContainer.GetSelectedTabIdx())
			te := tab.Child.(*ui.TextEdit)

			contents, err := ClipRead()
			if err != nil {
				showErrorDialog("Clipboard Failure", fmt.Sprintf("%v", err), nil)
			} else {
				te.Insert(contents)
				changeFocus(tabContainer)
			}
		}
	}}, &ui.ItemSeparator{}, &ui.ItemEntry{Name: "Select All", QuickChar: 7, Shortcut: "Ctrl+A", Callback: func() {

	}}, &ui.ItemEntry{Name: "Select Line", QuickChar: 7, Callback: func() {

	}}})

	searchMenu := ui.NewMenu("Search", 0, &theme)

	searchMenu.AddItems([]ui.Item{&ui.ItemEntry{Name: "Find and Replace...", Shortcut: "Ctrl+F", Callback: func() {
		screen.Beep()
	}}, &ui.ItemEntry{Name: "Find in Directory...", QuickChar: 8, Callback: func() {

	}}, &ui.ItemSeparator{}, &ui.ItemEntry{Name: "Go to line...", Shortcut: "Ctrl+G", Callback: func() {

	}}})

	menuBar.AddMenu(fileMenu)
	menuBar.AddMenu(panelMenu)
	menuBar.AddMenu(editMenu)
	menuBar.AddMenu(searchMenu)

	for !closing {
		screen.Clear()

		// Draw background (grey and black checkerboard)
		ui.DrawRect(screen, 0, 0, sizex, sizey, '▚', tcell.Style{}.Foreground(tcell.ColorGrey).Background(tcell.ColorBlack))

		if tabContainer.GetTabCount() > 0 { // Draw the tab container only if a tab is open
			tabContainer.Draw(screen)
		}
		menuBar.Draw(screen) // Always draw the menu bar

		if dialog != nil {
			// Update fileSelector dialog pos and size
			diagMinX, diagMinY := dialog.GetMinSize()
			dialog.SetSize(diagMinX, diagMinY)
			dialog.SetPos(sizex/2-diagMinX/2, sizey/2-diagMinY/2) // Center

			dialog.Draw(screen)
		}

		// Draw statusbar
		ui.DrawRect(screen, 0, sizey-1, sizex, 1, ' ', theme["StatusBar"])
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
			ui.DrawStr(screen, 0, sizey-1, str, theme["StatusBar"])
		}

		screen.Show()

		switch ev := screen.PollEvent().(type) {
		case *tcell.EventResize:
			sizex, sizey = screen.Size()

			menuBar.SetSize(sizex, 1)
			tabContainer.SetSize(sizex, sizey-2)

			screen.Sync() // Redraw everything
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
