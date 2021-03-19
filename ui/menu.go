package ui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
)

// Item is an interface implemented by ItemEntry and ItemMenu to be listed in Menus.
type Item interface {
	isItem()
	GetName() string
}

// An ItemSeparator is like a blank Item that cannot actually be selected. It is useful
// for separating items in a Menu.
type ItemSeparator struct{}

func (i *ItemSeparator) isItem() {}

// GetName returns an empty string.
func (i *ItemSeparator) GetName() string {
	return ""
}

// ItemEntry is a listing in a Menu with a name and callback.
type ItemEntry struct {
	Name     string
	Callback func()
}

func (i *ItemEntry) isItem() {}

// GetName returns the name of the ItemEntry.
func (i *ItemEntry) GetName() string {
	return i.Name
}

func (m *Menu) isItem() {}

// GetName returns the name of the Menu.
func (m *Menu) GetName() string {
	return m.Name
}

// A MenuBar is a horizontal list of menus.
type MenuBar struct {
	Menus []*Menu

	x, y          int
	width, height int
	focused       bool
	selected      int   // Index of selection in MenuBar
	menusVisible   bool // Whether to draw the selected menu

	Theme *Theme
}

func NewMenuBar(theme *Theme) *MenuBar {
	return &MenuBar{
		Menus: make([]*Menu, 0, 6),
		Theme: theme,
	}
}

func (b *MenuBar) AddMenu(menu *Menu) {
	menu.itemSelectedCallback = func() {
		b.menusVisible = false
		menu.SetFocused(false)
	}
	b.Menus = append(b.Menus, menu)
}

// GetMenuXPos returns the X position of the name of Menu at `idx` visually.
func (b *MenuBar) GetMenuXPos(idx int) int {
	x := 1
	for i := 0; i < idx; i++ {
		x += len(b.Menus[i].Name) + 2 // two for padding
	}
	return x
}

// Draw renders the MenuBar and its sub-menus.
func (b *MenuBar) Draw(s tcell.Screen) {
	normalStyle := b.Theme.GetOrDefault("MenuBar")

	// Draw menus based on whether b.focused and which is selected
	DrawRect(s, b.x, b.y, b.width, 1, ' ', normalStyle)
	col := b.x + 1
	for i, item := range b.Menus {
		str := fmt.Sprintf(" %s ", item.Name) // Surround the name in spaces
		var sty tcell.Style
		if b.selected == i && b.focused { // If we are drawing the selected item ...
			sty = b.Theme.GetOrDefault("MenuBarSelected") // Use style for selected items
		} else {
			sty = normalStyle
		}
		DrawStr(s, col, b.y, str, sty)
		col += len(str)
	}

	if b.menusVisible {
		menu := b.Menus[b.selected]
		menu.Draw(s) // Draw menu when it is expanded / visible
	}
}

// SetFocused highlights the MenuBar and focuses any sub-menus.
func (b *MenuBar) SetFocused(v bool) {
	b.focused = v
	b.Menus[b.selected].SetFocused(v)
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
		switch ev.Key() {
		case tcell.KeyEnter:
			if !b.menusVisible { // If menus are not visible...
				b.menusVisible = true

				menu := &b.Menus[b.selected]
				(*menu).SetPos(b.GetMenuXPos(b.selected), b.y+1)
				(*menu).SetFocused(true) // Makes .Visible true for the Menu
			} else { // The selected Menu is visible, send the event to it
				return b.Menus[b.selected].HandleEvent(event)				
			}
		case tcell.KeyLeft:
			b.Menus[b.selected].SetFocused(false) // Unfocus current menu
			if b.selected <= 0 {
				b.selected = len(b.Menus) - 1 // Wrap to end
			} else {
				b.selected--
			}
			// Update position of new menu after changing menu selection
			b.Menus[b.selected].SetPos(b.GetMenuXPos(b.selected), b.y+1)
			b.Menus[b.selected].SetFocused(true) // Focus new menu
		case tcell.KeyRight:
			b.Menus[b.selected].SetFocused(false)
			if b.selected >= len(b.Menus)-1 {
				b.selected = 0 // Wrap to beginning
			} else {
				b.selected++
			}
			// Update position of new menu after changing menu selection
			b.Menus[b.selected].SetPos(b.GetMenuXPos(b.selected), b.y+1)
			b.Menus[b.selected].SetFocused(true) // Focus new menu

		case tcell.KeyRune: // Search for the matching quick char in menu names
			if !b.menusVisible { // If the selected Menu is not open/visible
				for _, m := range b.Menus {
					found, r := QuickCharInString(m.Name)
					if found && r == ev.Rune() {
						b.menusVisible = true

						menu := &b.Menus[b.selected]
						(*menu).SetPos(b.GetMenuXPos(b.selected), b.y+1)
						(*menu).SetFocused(true)
					}
				}
			} else {
				return b.Menus[b.selected].HandleEvent(event) // Have menu handle quick char event
			}

		default:
			if b.menusVisible {
				return b.Menus[b.selected].HandleEvent(event)
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
	Name    string
	Items   []Item

	x, y                 int
	width, height        int    // Size may not be settable
	selected             int    // Index of selected Item
	itemSelectedCallback func() // Used internally to hide menus on selection

	Theme *Theme
}

// New creates a new Menu. `items` can be `nil`.
func NewMenu(name string, theme *Theme) *Menu {
	return &Menu{
		Name:  name,
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
			itemName := item.GetName()
			str := fmt.Sprintf("%s%s", itemName, strings.Repeat(" ", m.width-2-len(itemName)))
			DrawStr(s, m.x+1, m.y+1+i, str, sty)
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

// GetSize returns the size of the Menu.
func (m *Menu) GetSize() (int, int) {
	// TODO: no, pls don't do this
	maxLen := 0
	for _, item := range m.Items {
		len := len(item.GetName())
		if len > maxLen {
			maxLen = len
		}
	}
	m.width = maxLen + 2        // Add two for padding
	m.height = len(m.Items) + 2 // And another two for the same reason ...
	return m.width, m.height
}

// SetSize sets the size of the Menu.
func (m *Menu) SetSize(width, height int) {
	// Cannot set the size of a Menu
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
		case tcell.KeyDown:
			m.CursorDown()

		case tcell.KeyRune:
			// TODO: support quick chars for sub-menus
			for i, item := range m.Items {
				found, r := QuickCharInString(item.GetName())
				if found && r == ev.Rune() {
					m.selected = i
					m.ActivateItemUnderCursor()
				}
			}

		default:
			return false
		}
		return true
	}
	return false
}
