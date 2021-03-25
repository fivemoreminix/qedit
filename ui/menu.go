package ui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	runewidth "github.com/mattn/go-runewidth"
)

// Item is an interface implemented by ItemEntry and ItemMenu to be listed in Menus.
type Item interface {
	GetName() string
	// Returns a character/rune index of the name of the item.
	GetQuickCharIdx() int
	// A Shortcut is a string of the modifiers+key name of the action that must be pressed
	// to trigger the shortcut. For example: "Ctrl+Alt+X". The order of the modifiers is
	// very important. Letters are case-sensitive. See the KeyEvent.Name() function of tcell
	// for information. An empty string implies no shortcut.
	GetShortcut() string
}

// An ItemSeparator is like a blank Item that cannot actually be selected. It is useful
// for separating items in a Menu.
type ItemSeparator struct{}

// GetName returns an empty string.
func (i *ItemSeparator) GetName() string {
	return ""
}

func (i *ItemSeparator) GetQuickCharIdx() int {
	return 0
}

func (i *ItemSeparator) GetShortcut() string {
	return ""
}

// ItemEntry is a listing in a Menu with a name and callback.
type ItemEntry struct {
	Name      string
	QuickChar int // Character/rune index of Name
	Shortcut  string
	Callback  func()
}

// GetName returns the name of the ItemEntry.
func (i *ItemEntry) GetName() string {
	return i.Name
}

func (i *ItemEntry) GetQuickCharIdx() int {
	return i.QuickChar
}

func (i *ItemEntry) GetShortcut() string {
	return i.Shortcut
}

// GetName returns the name of the Menu.
func (m *Menu) GetName() string {
	return m.Name
}

func (m *Menu) GetQuickCharIdx() int {
	return m.QuickChar
}

func (m *Menu) GetShortcut() string {
	return ""
}

// A MenuBar is a horizontal list of menus.
type MenuBar struct {
	menus         []*Menu
	x, y          int
	width, height int
	focused       bool
	selected      int   // Index of selection in MenuBar
	menusVisible  bool // Whether to draw the selected menu

	Theme *Theme
}

func NewMenuBar(theme *Theme) *MenuBar {
	return &MenuBar{
		menus: make([]*Menu, 0, 6),
		Theme: theme,
	}
}

func (b *MenuBar) AddMenu(menu *Menu) {
	menu.itemSelectedCallback = func() {
		b.menusVisible = false
		menu.SetFocused(false)
	}
	b.menus = append(b.menus, menu)
}

// GetMenuXPos returns the X position of the name of Menu at `idx` visually.
func (b *MenuBar) GetMenuXPos(idx int) int {
	x := 1
	for i := 0; i < idx; i++ {
		x += len(b.menus[i].Name) + 2 // two for padding
	}
	return x
}

func (b *MenuBar) ActivateMenuUnderCursor() {
	b.menusVisible = true // Show menus
	menu := &b.menus[b.selected]
	(*menu).SetPos(b.GetMenuXPos(b.selected), b.y+1)
	(*menu).SetFocused(true)
}

func (b *MenuBar) CursorLeft() {
	if b.menusVisible {	
		b.menus[b.selected].SetFocused(false) // Unfocus current menu
	}

	if b.selected <= 0 {
		b.selected = len(b.menus) - 1 // Wrap to end
	} else {
		b.selected--
	}

	if b.menusVisible {
		// Update position of new menu after changing menu selection
		b.menus[b.selected].SetPos(b.GetMenuXPos(b.selected), b.y+1)
		b.menus[b.selected].SetFocused(true) // Focus new menu
	}
}

func (b *MenuBar) CursorRight() {
	if b.menusVisible {
		b.menus[b.selected].SetFocused(false)
	}

	if b.selected >= len(b.menus)-1 {
		b.selected = 0 // Wrap to beginning
	} else {
		b.selected++
	}

	if b.menusVisible {
		// Update position of new menu after changing menu selection
		b.menus[b.selected].SetPos(b.GetMenuXPos(b.selected), b.y+1)
		b.menus[b.selected].SetFocused(true) // Focus new menu
	}
}

// Draw renders the MenuBar and its sub-menus.
func (b *MenuBar) Draw(s tcell.Screen) {
	normalStyle := b.Theme.GetOrDefault("MenuBar")

	// Draw menus based on whether b.focused and which is selected
	DrawRect(s, b.x, b.y, b.width, 1, ' ', normalStyle)
	col := b.x + 1
	for i, item := range b.menus {
		sty := normalStyle		
		if b.focused && b.selected == i {
			sty = b.Theme.GetOrDefault("MenuBarSelected") // Use special style for selected item
		}

		str := fmt.Sprintf(" %s ", item.Name)
		cols := DrawQuickCharStr(s, col, b.y, str, item.QuickChar+1, sty)

		col += cols
	}

	if b.menusVisible {
		menu := b.menus[b.selected]
		menu.Draw(s) // Draw menu when it is expanded / visible
	}
}

// SetFocused highlights the MenuBar and focuses any sub-menus.
func (b *MenuBar) SetFocused(v bool) {
	b.focused = v
	b.menus[b.selected].SetFocused(v)
	if !v {
		b.selected = 0 // Reset cursor position every time component is unfocused
		if b.menusVisible {
			b.menusVisible = false
		}
	}
}

func (b *MenuBar) SetTheme(theme *Theme) {
	b.Theme = theme
}

// GetPos returns the position of the MenuBar.
func (b *MenuBar) GetPos() (int, int) {
	return b.x, b.y
}

// SetPos sets the position of the MenuBar.
func (b *MenuBar) SetPos(x, y int) {
	b.x, b.y = x, y
}

func (b *MenuBar) GetMinSize() (int, int) {
	return 0, 1
}

// GetSize returns the size of the MenuBar.
func (b *MenuBar) GetSize() (int, int) {
	return b.width, b.height
}

// SetSize sets the size of the MenuBar.
func (b *MenuBar) SetSize(width, height int) {
	b.width, b.height = width, height
}

// HandleEvent will propogate events to sub-menus and returns true if
// any of them handled the event.
func (b *MenuBar) HandleEvent(event tcell.Event) bool {
	switch ev := event.(type) {
	case *tcell.EventKey:
		// Shortcuts (Ctrl-s or Ctrl-A, for example)
		if ev.Modifiers() != 0 { // If there is a modifier on the key...
			// tcell names it "Ctrl+(Key)" so we want to remove the "Ctrl+"
			// prefix, and use the remaining part of the string as the shortcut.

			keyName := ev.Name()

			// Find who the shortcut key belongs to
			for i := range b.menus {
				handled := b.menus[i].handleShortcut(keyName)
				if handled {
					return true
				}
			}
			return false // The shortcut key was not handled by any menus
		}

		switch ev.Key() {
		case tcell.KeyEnter:
			if !b.menusVisible { // If menus are not visible...
				b.ActivateMenuUnderCursor()
			} else { // The selected Menu is visible, send the event to it
				return b.menus[b.selected].HandleEvent(event)				
			}
		case tcell.KeyLeft:
			b.CursorLeft()
		case tcell.KeyRight:
			b.CursorRight()
		case tcell.KeyTab:
			if b.menusVisible {
				return b.menus[b.selected].HandleEvent(event)
			} else {
				b.CursorRight()
			}

		// Quick char
		case tcell.KeyRune: // Search for the matching quick char in menu names
			if !b.menusVisible { // If the selected Menu is not open/visible
				for i, m := range b.menus {
					r := QuickCharInString(m.Name, m.QuickChar)
					if r != 0 && r == ev.Rune() {
						b.selected = i // Select menu at i
						b.ActivateMenuUnderCursor() // Show menu
						break
					}
				}
			} else {
				return b.menus[b.selected].HandleEvent(event) // Have menu handle quick char event
			}		

		default:
			if b.menusVisible {
				return b.menus[b.selected].HandleEvent(event)
			} else {
				return false // Nobody to propogate our event to
			}
		}
		return true
	}
	return false
}

// A Menu contains one or more ItemEntry or ItemMenus.
type Menu struct {
	Name      string
	QuickChar int // Character/rune index of Name
	Items     []Item

	x, y                 int
	width, height        int    // Size may not be settable
	selected             int    // Index of selected Item
	itemSelectedCallback func() // Used internally to hide menus on selection

	Theme *Theme
}

// New creates a new Menu. `items` can be `nil`.
func NewMenu(name string, quickChar int, theme *Theme) *Menu {
	return &Menu{
		Name:  name,
		QuickChar: quickChar,
		Items: make([]Item, 0, 6),
		Theme: theme,
	}
}

func (m *Menu) AddItem(item Item) {
	switch typ := item.(type) {
	case *Menu:
		typ.itemSelectedCallback = func() {
			m.itemSelectedCallback()
		}
	}
	m.Items = append(m.Items, item)
}

func (m *Menu) AddItems(items []Item) {
	for _, item := range items {
		m.AddItem(item)
	}
}

func (m *Menu) ActivateItemUnderCursor() {
	switch item := m.Items[m.selected].(type) {
	case *ItemEntry:
		item.Callback()
		m.itemSelectedCallback()
	case *Menu:
		// TODO: implement sub-menus ...
	}	
}

func (m *Menu) CursorUp() {
	if m.selected <= 0 {
		m.selected = len(m.Items) - 1 // Wrap to end
	} else {
		m.selected--
	}
	switch m.Items[m.selected].(type) {
	case *ItemSeparator:
		m.CursorUp() // Recursion; stack overflow if the only item in a Menu is a separator.
	default:
	}
}

func (m *Menu) CursorDown() {
	if m.selected >= len(m.Items)-1 {
		m.selected = 0 // Wrap to beginning
	} else {
		m.selected++
	}
	switch m.Items[m.selected].(type) {
	case *ItemSeparator:
		m.CursorDown() // Recursion; stack overflow if the only item in a Menu is a separator.
	default:
	}
}

// Draw renders the Menu at its position.
func (m *Menu) Draw(s tcell.Screen) {
	defaultStyle := m.Theme.GetOrDefault("Menu")

	m.GetSize()                                                          // Call this to update internal width and height
	DrawRect(s, m.x, m.y, m.width, m.height, ' ', defaultStyle)          // Fill background
	DrawRectOutlineDefault(s, m.x, m.y, m.width, m.height, defaultStyle) // Draw outline

	// Draw items based on whether m.focused and which is selected
	for i, item := range m.Items {
		switch item.(type) {
		case *ItemSeparator:
			str := fmt.Sprintf("%s%s%s", "├", strings.Repeat("─", m.width-2), "┤")
			DrawStr(s, m.x, m.y+1+i, str, defaultStyle)
		default: // Handle sub-menus and item entries the same
			var sty tcell.Style
			if m.selected == i {
				sty = m.Theme.GetOrDefault("MenuSelected")
			} else {
				sty = defaultStyle
			}
			
			nameCols := DrawQuickCharStr(s, m.x+1, m.y+1+i, item.GetName(), item.GetQuickCharIdx(), sty)

			str := strings.Repeat(" ", m.width-2-nameCols) // Fill space after menu names to border
			DrawStr(s, m.x+1+nameCols, m.y+1+i, str, sty)

			if shortcut := item.GetShortcut(); len(shortcut) > 0 { // If the item has a shortcut...
				str := " " + shortcut + " "
				DrawStr(s, m.x+m.width-1-runewidth.StringWidth(str), m.y+1+i, str, sty)
			}
		}
	}
}

// SetFocused does not do anything for a Menu.
func (m *Menu) SetFocused(v bool) {
	// TODO: when adding sub-menus, set all focus to v
	if !v {
		m.selected = 0
	}
}

// GetPos returns the position of the Menu.
func (m *Menu) GetPos() (int, int) {
	return m.x, m.y
}

// SetPos sets the position of the Menu.
func (m *Menu) SetPos(x, y int) {
	m.x, m.y = x, y
}

func (m *Menu) GetMinSize() (int, int) {
	return m.GetSize()
}

// GetSize returns the size of the Menu.
func (m *Menu) GetSize() (int, int) {
	// TODO: no, pls don't do this
	maxNameLen := 0
	var widestShortcut int = 0 // Will contribute to the width
	for i := range m.Items {
		nameLen := len(m.Items[i].GetName())
		if nameLen > maxNameLen {
			maxNameLen = nameLen
		}

		if key := m.Items[i].GetShortcut(); runewidth.StringWidth(key) > widestShortcut {
			widestShortcut = runewidth.StringWidth(key) // For the sake of good unicode
		}
	}

	shortcutsWidth := 0
	if widestShortcut > 0 {
		shortcutsWidth = 1 + widestShortcut + 1 // " Ctrl+X "  (with one cell padding surrounding)
	}

	m.width = 1 + maxNameLen + shortcutsWidth + 1 // Add two for padding
	m.height = 1 + len(m.Items) + 1           // And another two for the same reason ...
	return m.width, m.height
}

// SetSize sets the size of the Menu.
func (m *Menu) SetSize(width, height int) {
	// Cannot set the size of a Menu
}

func (m *Menu) handleShortcut(key string) bool {
	for i := range m.Items {
		switch typ := m.Items[i].(type) {
		case *ItemSeparator:
			continue
		case *Menu:
			return typ.handleShortcut(key) // Have the sub-menu handle the shortcut
		case *ItemEntry:
			if typ.Shortcut == key { // If this item matches the shortcut we're finding...
				m.selected = i
				m.ActivateItemUnderCursor() // Activate it
				return true
			}
		}
	}
	return false
}

// HandleEvent will handle events for a Menu and may propogate them
// to sub-menus. Returns true if the event was handled.
func (m *Menu) HandleEvent(event tcell.Event) bool {
	// TODO: simplify this function
	switch ev := event.(type) {
	case *tcell.EventKey:
		switch ev.Key() {
		case tcell.KeyEnter:
			m.ActivateItemUnderCursor()
		case tcell.KeyUp:
			m.CursorUp()
		case tcell.KeyTab:
			fallthrough
		case tcell.KeyDown:
			m.CursorDown()

		case tcell.KeyRune:
			// TODO: support quick chars for sub-menus
			for i, item := range m.Items {
				if m.selected == i {
					continue // Skip the item we're on
				}
				r := QuickCharInString(item.GetName(), item.GetQuickCharIdx())
				if r != 0 && r == ev.Rune() {
					m.selected = i
					break
				}
			}

		default:
			return false
		}
		return true
	}
	return false
}
