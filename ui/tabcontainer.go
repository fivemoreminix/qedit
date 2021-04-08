package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
)

// A Tab is a child of a TabContainer; has a name and child Component.
type Tab struct {
	Name  string
	Child Component
}

// A TabContainer organizes children by showing only one of them at a time.
type TabContainer struct {
	children      []Tab
	selected      int

	baseComponent
}

func NewTabContainer(theme *Theme) *TabContainer {
	return &TabContainer{
		children: make([]Tab, 0, 4),
		baseComponent: baseComponent{theme: theme},
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
		if c.selected == idx {
			c.children[idx].Child.SetFocused(false)
		}

		copy(c.children[idx:], c.children[idx+1:])  // Shift all items after idx to the left
		c.children = c.children[:len(c.children)-1] // Shrink slice by one

		if c.selected >= idx && idx > 0 {
			c.selected-- // Keep the cursor within the bounds of available tabs
		}

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
	var styFocused tcell.Style
	if c.focused {
		styFocused = c.theme.GetOrDefault("TabContainerFocused")
	} else {
		styFocused = c.theme.GetOrDefault("TabContainer")
	}

	// Draw outline
	DrawRectOutlineDefault(s, c.x, c.y, c.width, c.height, styFocused)

	combinedTabLength := 0
	for i := range c.children {
		combinedTabLength += len(c.children[i].Name) + 2 // 2 for padding
	}
	combinedTabLength += len(c.children) - 1 // add for spacing between tabs

	// Draw tabs
	col := c.x + c.width/2 - combinedTabLength/2 // Starting column
	for i, tab := range c.children {
		sty := styFocused
		if c.selected == i {
			fg, bg, attr := styFocused.Decompose()
			sty = tcell.Style{}.Foreground(bg).Background(fg).Attributes(attr)
		}

		var dirty bool
		switch typ := tab.Child.(type) {
		case *TextEdit:
			dirty = typ.Dirty
		}

		name := tab.Name
		if dirty {
			name = "*" + name
		}

		str := fmt.Sprintf(" %s ", name)

		DrawStr(s, col, c.y, str, sty)
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
	if len(c.children) > 0 {
		c.children[c.selected].Child.SetFocused(v)
	}
}

// SetTheme sets the theme.
func (c *TabContainer) SetTheme(theme *Theme) {
	c.theme = theme
	for _, tab := range c.children {
		tab.Child.SetTheme(theme) // Update the theme for all children
	}
}

// SetPos sets the position of the container and updates the child Component.
func (c *TabContainer) SetPos(x, y int) {
	c.x, c.y = x, y
	if c.selected < len(c.children) {
		c.children[c.selected].Child.SetPos(x+1, y+1)
	}
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
