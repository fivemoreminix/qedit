package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/fivemoreminix/qedit/internal/clipboard"
	internal_ui "github.com/fivemoreminix/qedit/internal/ui"
	"github.com/fivemoreminix/qedit/pkg/ui"
	"github.com/gdamore/tcell/v2"
)

var (
	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
	memprofile = flag.String("memprofile", "", "write memory profile to `file`")
)

var theme = ui.Theme{
	"StatusBar": tcell.Style{}.Foreground(tcell.ColorBlack).Background(tcell.ColorLightGray),
}

var (
	screen *tcell.Screen

	menuBar        *ui.MenuBar
	panelContainer *ui.PanelContainer
	dialog         ui.Component // nil if not present (has exclusive focus)

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
			changeFocus(panelContainer) // Default behavior: focus panelContainer
		}
	})
	changeFocus(dialog)
}

func getActiveTabContainer() *ui.TabContainer {
	if panelContainer.GetSelected() != nil {
		return panelContainer.GetSelected().(*ui.TabContainer)
	}
	return nil
}

// returns nil if no TextEdit is visible
func getActiveTextEdit() *ui.TextEdit {
	tabContainer := getActiveTabContainer()
	if tabContainer != nil && tabContainer.GetTabCount() > 0 {
		tab := tabContainer.GetTab(tabContainer.GetSelectedTabIdx())
		te := tab.Child.(*ui.TextEdit)
		return te
	}
	return nil
}

// Shows the Save As... dialog for saving unnamed files
func saveAs() {
	callback := func(filePaths []string) {
		te := getActiveTextEdit() // te should have value if we are here
		tabContainer := getActiveTabContainer()
		tab := tabContainer.GetTab(tabContainer.GetSelectedTabIdx())

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

		te.FilePath = filePaths[0]
		tab.Name = filePaths[0]

		dialog = nil // Hide the file selector
		changeFocus(panelContainer)
		tab.Name = filePaths[0]
	}

	dialog = ui.NewFileSelectorDialog(
		screen,
		"Select a file to overwrite",
		false,
		&theme,
		callback,
		func() { // Dialog canceled
			dialog = nil
			changeFocus(panelContainer)
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

	s, err := tcell.NewScreen()
	screen = &s
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	if err := s.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	defer s.Fini() // Useful for handling panics

	var closing bool
	sizex, sizey := s.Size()

	panelContainer = ui.NewPanelContainer(&theme)
	panelContainer.SetPos(0, 1)
	panelContainer.SetSize(sizex, sizey-2)

	panelContainer.SetSelected(ui.NewTabContainer(&theme))

	changeFocus(panelContainer) // panelContainer focused by default

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

			textEdit := ui.NewTextEdit(screen, arg, bytes, &theme)
			textEdit.Dirty = dirty
			getActiveTabContainer().AddTab(arg, textEdit)
		}
		panelContainer.SetFocused(true) // Lets any opened TextEdit component know to be focused
	}

	_, err = clipboard.ClipInitialize(clipboard.ClipExternal)
	if err != nil {
		showErrorDialog("Error Initializing Clipboard", fmt.Sprintf("%v\n\nAn internal clipboard will be used, instead.", err), nil)
	}

	menuBar = ui.NewMenuBar(&theme)

	fileMenu := ui.NewMenu("File", 0, &theme)

	fileMenu.AddItems([]ui.Item{&ui.ItemEntry{Name: "New File", Shortcut: "Ctrl+N", Callback: func() {
		textEdit := ui.NewTextEdit(screen, "", []byte{}, &theme) // No file path, no contents
		tabContainer := getActiveTabContainer()
		if tabContainer == nil {
			tabContainer = ui.NewTabContainer(&theme)
			panelContainer.SetSelected(tabContainer)
		}
		tabContainer.AddTab("noname", textEdit)
		tabContainer.FocusTab(tabContainer.GetTabCount() - 1)
		changeFocus(panelContainer)
	}}, &ui.ItemEntry{Name: "Open...", Shortcut: "Ctrl+O", Callback: func() {
		callback := func(filePaths []string) {
			tabContainer := getActiveTabContainer()

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

				textEdit := ui.NewTextEdit(screen, path, bytes, &theme)
				if tabContainer == nil {
					tabContainer = ui.NewTabContainer(&theme)
					panelContainer.SetSelected(tabContainer)
				}
				tabContainer.AddTab(path, textEdit)
			}

			if !errOccurred { // Prevent hiding the error dialog
				dialog = nil // Hide the file selector
				changeFocus(panelContainer)
				if tabContainer.GetTabCount() > 0 {
					tabContainer.FocusTab(tabContainer.GetTabCount() - 1)
				}
			}
		}
		dialog = ui.NewFileSelectorDialog(
			screen,
			"Comma-separated files or a directory",
			true,
			&theme,
			callback,
			func() { // Dialog is canceled
				dialog = nil
				changeFocus(panelContainer)
			},
		)
		changeFocus(dialog)
	}}, &ui.ItemEntry{Name: "Save", Shortcut: "Ctrl+S", Callback: func() {
		te := getActiveTextEdit()
		if te != nil {
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

				changeFocus(panelContainer)
			} else {
				saveAs()
			}
		}
	}}, &ui.ItemEntry{Name: "Save As...", QuickChar: 5, Callback: saveAs}, &ui.ItemSeparator{},
		&ui.ItemEntry{Name: "Close", Shortcut: "Ctrl+Q", Callback: func() {
			tabContainer := getActiveTabContainer()
			if tabContainer != nil && tabContainer.GetTabCount() > 0 {
				tabContainer.RemoveTab(tabContainer.GetSelectedTabIdx())
			} else {
				// if the selected is root: close editor. otherwise close panel
				if panelContainer.IsRootSelected() {
					closing = true
				} else {
					panelContainer.DeleteSelected()
				}
			}
		}}})

	panelMenu := ui.NewMenu("Panel", 0, &theme)

	panelMenu.AddItems([]ui.Item{&ui.ItemEntry{Name: "Focus Next", Shortcut: "Alt+.", Callback: func() {
		panelContainer.SelectNext()
		changeFocus(panelContainer)
	}}, &ui.ItemEntry{Name: "Focus Prev", Shortcut: "Alt+,", Callback: func() {
		panelContainer.SelectPrev()
		changeFocus(panelContainer)
	}}, &ui.ItemEntry{Name: "Focus Up", QuickChar: -1, Shortcut: "Alt+Up", Callback: func() {

	}}, &ui.ItemEntry{Name: "Focus Down", QuickChar: -1, Shortcut: "Alt+Down", Callback: func() {

	}}, &ui.ItemEntry{Name: "Focus Left", QuickChar: -1, Shortcut: "Alt+Left", Callback: func() {

	}}, &ui.ItemEntry{Name: "Focus Right", QuickChar: -1, Shortcut: "Alt+Right", Callback: func() {

	}}, &ui.ItemSeparator{}, &ui.ItemEntry{Name: "Split Top", QuickChar: 6, Callback: func() {
		panelContainer.SplitSelected(ui.SplitVertical, ui.NewTabContainer(&theme))
		panelContainer.SwapNeighborsSelected()
		panelContainer.SelectPrev()
		changeFocus(panelContainer)
	}}, &ui.ItemEntry{Name: "Split Bottom", QuickChar: 6, Callback: func() {
		panelContainer.SplitSelected(ui.SplitVertical, ui.NewTabContainer(&theme))
		panelContainer.SelectNext()
		changeFocus(panelContainer)
	}}, &ui.ItemEntry{Name: "Split Left", QuickChar: 6, Callback: func() {
		panelContainer.SplitSelected(ui.SplitHorizontal, ui.NewTabContainer(&theme))
		panelContainer.SwapNeighborsSelected()
		panelContainer.SelectPrev()
		changeFocus(panelContainer)
	}}, &ui.ItemEntry{Name: "Split Right", QuickChar: 6, Callback: func() {
		panelContainer.SplitSelected(ui.SplitHorizontal, ui.NewTabContainer(&theme))
		panelContainer.SelectNext()
		changeFocus(panelContainer)
	}}, &ui.ItemSeparator{}, &ui.ItemEntry{Name: "Move", Shortcut: "Ctrl+M", Callback: func() {

	}}, &ui.ItemEntry{Name: "Resize", Shortcut: "Ctrl+R", Callback: func() {

	}}, &ui.ItemEntry{Name: "Toggle Floating", Callback: func() {
		panelContainer.FloatSelected()
		if !panelContainer.GetFloatingFocused() {
			panelContainer.SetFloatingFocused(true)
		}
		changeFocus(panelContainer)
	}}})

	editMenu := ui.NewMenu("Edit", 0, &theme)

	editMenu.AddItems([]ui.Item{&ui.ItemEntry{Name: "Cut", Shortcut: "Ctrl+X", Callback: func() {
		te := getActiveTextEdit()
		if te != nil {
			bytes := te.GetSelectedBytes()
			var err error
			if len(bytes) > 0 { // If something is selected...
				err = clipboard.ClipWrite(string(bytes)) // Add the selectedStr to clipboard
				if err != nil {
					showErrorDialog("Clipboard Failure", fmt.Sprintf("%v", err), nil)
				}
				te.Delete(false) // Delete selection
			}
			if err == nil { // Prevent hiding error dialog
				changeFocus(panelContainer)
			}
		}
	}}, &ui.ItemEntry{Name: "Copy", Shortcut: "Ctrl+C", Callback: func() {
		te := getActiveTextEdit()
		if te != nil {
			bytes := te.GetSelectedBytes()
			var err error
			if len(bytes) > 0 { // If there is something selected...
				err = clipboard.ClipWrite(string(bytes)) // Add selectedStr to clipboard
				if err != nil {
					showErrorDialog("Clipboard Failure", fmt.Sprintf("%v", err), nil)
				}
			}
			if err == nil {
				changeFocus(panelContainer)
			}
		}
	}}, &ui.ItemEntry{Name: "Paste", Shortcut: "Ctrl+V", Callback: func() {
		te := getActiveTextEdit()
		if te != nil {
			contents, err := clipboard.ClipRead()
			if err != nil {
				showErrorDialog("Clipboard Failure", fmt.Sprintf("%v", err), nil)
			} else {
				te.Insert(contents)
				changeFocus(panelContainer)
			}
		}
	}}, &ui.ItemSeparator{}, &ui.ItemEntry{Name: "Select All", QuickChar: 7, Shortcut: "Ctrl+A", Callback: func() {

	}}, &ui.ItemEntry{Name: "Select Line", QuickChar: 7, Callback: func() {

	}}})

	searchMenu := ui.NewMenu("Search", 0, &theme)

	searchMenu.AddItems([]ui.Item{&ui.ItemEntry{Name: "Find and Replace...", Shortcut: "Ctrl+F", Callback: func() {
		s.Beep()
	}}, &ui.ItemEntry{Name: "Find in Directory...", QuickChar: 8, Callback: func() {

	}}, &ui.ItemSeparator{}, &ui.ItemEntry{Name: "Go to line...", Shortcut: "Ctrl+G", Callback: func() {
		te := getActiveTextEdit()
		if te != nil {
			callback := func(line int) {
				te := getActiveTextEdit()
				te.SetCursor(te.GetCursor().SetLineCol(line-1, 0))
				// Hide dialog
				dialog = nil
				changeFocus(panelContainer)
			}
			dialog = internal_ui.NewGotoLineDialog(screen, &theme, callback, func() {
				// Dialog canceled
				dialog = nil
				changeFocus(panelContainer)
			})
			changeFocus(dialog)
		}
	}}})

	menuBar.AddMenu(fileMenu)
	menuBar.AddMenu(panelMenu)
	menuBar.AddMenu(editMenu)
	menuBar.AddMenu(searchMenu)

	for !closing {
		s.Clear()

		// Draw background (grey and black checkerboard)
		// TODO: draw checkered background on panics with error dialog
		//ui.DrawRect(screen, 0, 0, sizex, sizey, '▚', tcell.Style{}.Foreground(tcell.ColorGrey).Background(tcell.ColorBlack))
		ui.DrawRect(s, 0, 1, sizex, sizey-1, ' ', tcell.Style{}.Background(tcell.ColorBlack))

		panelContainer.Draw(s)
		menuBar.Draw(s)

		if dialog != nil {
			// Update fileSelector dialog pos and size
			diagMinX, diagMinY := dialog.GetMinSize()
			dialog.SetSize(diagMinX, diagMinY)
			dialog.SetPos(sizex/2-diagMinX/2, sizey/2-diagMinY/2) // Center

			dialog.Draw(s)
		}

		// Draw statusbar
		ui.DrawRect(s, 0, sizey-1, sizex, 1, ' ', theme["StatusBar"])
		if te := getActiveTextEdit(); te != nil {
			var delim string
			if te.IsCRLF {
				delim = "CRLF"
			} else {
				delim = "LF"
			}

			line, col := te.GetCursor().GetLineCol()

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
			panelContainer.SetSize(sizex, sizey-2)

			s.Sync() // Redraw everything
		case *tcell.EventKey:
			// On Escape, we change focus between editor and the MenuBar.
			if dialog == nil {
				if ev.Key() == tcell.KeyEscape {
					if focusedComponent == panelContainer {
						changeFocus(menuBar)
					} else {
						changeFocus(panelContainer)
					}
				}

				if ev.Modifiers()&tcell.ModCtrl != 0 {
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
