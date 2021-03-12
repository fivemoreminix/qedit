package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
)

// A Container has zero or more Components. Containers decide how Components are
// laid out in view, and may draw decorations like bounding boxes.
type Container interface {
	Component
}

// A BoxContainer draws an outline using the `Character` with `Style` attributes
// around the `Child` Component.
type BoxContainer struct {
	Child Component

	x, y          int
	width, height int
	ULRune        rune // Rune for upper-left
	URRune        rune // Rune for upper-right
	BLRune        rune // Rune for bottom-left
	BRRune        rune // Rune for bottom-right
	HorRune       rune // Rune for horizontals
	VertRune      rune // Rune for verticals

	Style tcell.Style
}

// New constructs a default BoxContainer using the terminal default style.
func NewBoxContainer(child Component, style tcell.Style) *BoxContainer {
	return &BoxContainer{
		Child:    child,
		ULRune:   '╭',
		URRune:   '╮',
		BLRune:   '╰',
		BRRune:   '╯',
		HorRune:  '—',
		VertRune: '│',
		Style:    style,
	}
}

// Draw will draws the border of the BoxContainer, then it draws its child component.
func (c *BoxContainer) Draw(s tcell.Screen) {
	DrawRectOutline(s, c.x, c.y, c.width, c.height, c.ULRune, c.URRune, c.BLRune, c.BRRune, c.HorRune, c.VertRune, c.Style)

	if c.Child != nil {
		c.Child.Draw(s)
	}
}

// SetFocused calls SetFocused on the child Component.
func (c *BoxContainer) SetFocused(v bool) {
	if c.Child != nil {
		c.Child.SetFocused(v)
	}
}

func (c *BoxContainer) SetTheme(theme *Theme) {}

// GetPos returns the position of the container.
func (c *BoxContainer) GetPos() (int, int) {
	return c.x, c.y
}

// SetPos sets the position of the container and updates the child Component.
func (c *BoxContainer) SetPos(x, y int) {
	c.x, c.y = x, y
	if c.Child != nil {
		c.Child.SetPos(x+1, y+1)
	}
}

func (c *BoxContainer) GetMinSize() (int, int) {
	return 0, 0
}

// GetSize gets the size of the container.
func (c *BoxContainer) GetSize() (int, int) {
	return c.width, c.height
}

// SetSize sets the size of the container and updates the size of the child Component.
func (c *BoxContainer) SetSize(width, height int) {
	c.width, c.height = width, height
	if c.Child != nil {
		c.Child.SetSize(width-2, height-2)
	}
}

// HandleEvent forwards the event to the child Component and returns whether it was handled.
func (c *BoxContainer) HandleEvent(event tcell.Event) bool {
	if c.Child != nil {
		return c.Child.HandleEvent(event)
	}
	return false
}

// A Tab is a child of a TabContainer; has a name and child Component.
type Tab struct {
	Name  string
	Child Component
}

// A TabContainer organizes children by showing only one of them at a time.
type TabContainer struct {
	children      []Tab
	x, y          int
	width, height int
	focused       bool
	selected      int

	Theme *Theme
}

func NewTabContainer(theme *Theme) *TabContainer {
	return &TabContainer{
		children: make([]Tab, 0, 4),
		Theme:    theme,
	}
}

func (c *TabContainer) AddTab(name string, child Component) {
	c.children = append(c.children, Tab{Name: name, Child: child})
	// Update new child's size and position
	child.SetPos(c.x+1, c.y+1)
	child.SetSize(c.width-2, c.height-2)
}

// RemoveTab deletes the tab at `idx`. Returns true if the tab was found,
// false otherwise.
func (c *TabContainer) RemoveTab(idx int) bool {
	if idx >= 0 && idx < len(c.children) {
		copy(c.children[idx:], c.children[idx+1:])  // Shift all items after idx to the left
		c.children = c.children[:len(c.children)-1] // Shrink slice by one

		return true
	}
	return false
}

// FocusTab sets the visible tab to the one at `idx`. FocusTab clamps `idx`
// between 0 and tab_count - 1. If no tabs are present, the function does nothing.
func (c *TabContainer) FocusTab(idx int) {
	if len(c.children) < 1 {
		return
	}

	if idx < 0 {
		idx = 0
	} else if idx >= len(c.children) {
		idx = len(c.children) - 1
	}

	c.children[c.selected].Child.SetFocused(false) // Unfocus old tab
	c.children[idx].Child.SetFocused(true)         // Focus new tab
	c.selected = idx
}

func (c *TabContainer) GetSelectedTabIdx() int {
	return c.selected
}

func (c *TabContainer) GetTabCount() int {
	return len(c.children)
}

func (c *TabContainer) GetTab(idx int) *Tab {
	return &c.children[idx]
}

// Draw will draws the border of the BoxContainer, then it draws its child component.
func (c *TabContainer) Draw(s tcell.Screen) {
	// Draw outline
	DrawRectOutlineDefault(s, c.x, c.y, c.width, c.height, c.Theme.GetOrDefault("TabContainer"))

	combinedTabLength := 0
	for _, tab := range c.children {
		combinedTabLength += len(tab.Name) + 2 // 2 for padding
	}
	combinedTabLength += len(c.children) - 1 // add for spacing between tabs

	// Draw tabs
	col := c.x + c.width/2 - combinedTabLength/2 // Starting column
	for i, tab := range c.children {
		var sty tcell.Style
		if c.selected == i {
			sty = c.Theme.GetOrDefault("TabSelected")
		} else {
			sty = c.Theme.GetOrDefault("Tab")
		}
		str := fmt.Sprintf(" %s ", tab.Name)
		//DrawStr(s, c.x+c.width/2-len(str)/2, c.y, str, sty)
		DrawStr(s, c.x+col, c.y, str, sty)
		col += len(str) + 1 // Add one for spacing between tabs
	}

	// Draw selected child in center
	if c.selected < len(c.children) {
		c.children[c.selected].Child.Draw(s)
	}
}

// SetFocused calls SetFocused on the visible child Component.
func (c *TabContainer) SetFocused(v bool) {
	c.focused = v
	if c.selected < len(c.children) {
		c.children[c.selected].Child.SetFocused(v)
	}
}

// SetTheme sets the theme.
func (c *TabContainer) SetTheme(theme *Theme) {
	c.Theme = theme
	for _, tab := range c.children {
		tab.Child.SetTheme(theme) // Update the theme for all children
	}
}

func (c *TabContainer) GetMinSize() (int, int) {
	return 0, 0
}

// GetPos returns the position of the container.
func (c *TabContainer) GetPos() (int, int) {
	return c.x, c.y
}

// SetPos sets the position of the container and updates the child Component.
func (c *TabContainer) SetPos(x, y int) {
	c.x, c.y = x, y
	if c.selected < len(c.children) {
		c.children[c.selected].Child.SetPos(x+1, y+1)
	}
}

// GetSize gets the size of the container.
func (c *TabContainer) GetSize() (int, int) {
	return c.width, c.height
}

// SetSize sets the size of the container and updates the size of the child Component.
func (c *TabContainer) SetSize(width, height int) {
	c.width, c.height = width, height
	if c.selected < len(c.children) {
		c.children[c.selected].Child.SetSize(width-2, height-2)
	}
}

// HandleEvent forwards the event to the child Component and returns whether it was handled.
func (c *TabContainer) HandleEvent(event tcell.Event) bool {
	switch ev := event.(type) {
	case *tcell.EventKey:
		if ev.Key() == tcell.KeyCtrlE {
			newIdx := c.selected + 1
			if newIdx >= len(c.children) {
				newIdx = 0
			}
			c.FocusTab(newIdx)
			return true
		} else if ev.Key() == tcell.KeyCtrlW {
			newIdx := c.selected - 1
			if newIdx < 0 {
				newIdx = len(c.children) - 1
			}
			c.FocusTab(newIdx)
			return true
		}
	}

	if c.selected < len(c.children) {
		return c.children[c.selected].Child.HandleEvent(event)
	}

	return false
}

// TODO: replace window container with draw function
// A WindowContainer has a border, a title, and a button to close the window.
type WindowContainer struct {
	Title string
	Child Component

	x, y          int
	width, height int
	focused       bool

	Theme *Theme
}

// New constructs a default WindowContainer using the terminal default style.
func NewWindowContainer(title string, child Component, theme *Theme) *WindowContainer {
	return &WindowContainer{
		Title: title,
		Child: child,
		Theme: theme,
	}
}

// Draw will draws the border of the WindowContainer, then it draws its child component.
func (w *WindowContainer) Draw(s tcell.Screen) {
	headerStyle := w.Theme.GetOrDefault("WindowHeader")

	DrawRect(s, w.x, w.y, w.width, 1, ' ', headerStyle)                               // Draw header
	DrawStr(s, w.x+w.width/2-len(w.Title)/2, w.y, w.Title, headerStyle)               // Draw title
	DrawRect(s, w.x, w.y+1, w.width, w.height-1, ' ', w.Theme.GetOrDefault("Window")) // Draw body background

	if w.Child != nil {
		w.Child.Draw(s)
	}
}

// SetFocused calls SetFocused on the child Component.
func (w *WindowContainer) SetFocused(v bool) {
	w.focused = v
	if w.Child != nil {
		w.Child.SetFocused(v)
	}
}

// GetPos returns the position of the container.
func (w *WindowContainer) GetPos() (int, int) {
	return w.x, w.y
}

// SetPos sets the position of the container and updates the child Component.
func (w *WindowContainer) SetPos(x, y int) {
	w.x, w.y = x, y
	if w.Child != nil {
		w.Child.SetPos(x, y+1)
	}
}

func (w *WindowContainer) GetMinSize() (int, int) {
	return 0, 0
}

// GetSize gets the size of the container.
func (w *WindowContainer) GetSize() (int, int) {
	return w.width, w.height
}

// SetSize sets the size of the container and updates the size of the child Component.
func (w *WindowContainer) SetSize(width, height int) {
	w.width, w.height = width, height
	if w.Child != nil {
		w.Child.SetSize(width, height-2)
	}
}

// HandleEvent forwards the event to the child Component and returns whether it was handled.
func (w *WindowContainer) HandleEvent(event tcell.Event) bool {
	if w.Child != nil {
		return w.Child.HandleEvent(event)
	}
	return false
}
