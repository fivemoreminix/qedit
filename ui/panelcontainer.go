package ui

import "github.com/gdamore/tcell/v2"

type SplitKind uint8

const (
	SplitVertical SplitKind = SplitKind(PanelKindSplitVert) + iota
	SplitHorizontal
)

type PanelContainer struct {
	root          *Panel
	floating      []*Panel
	selected      **Panel // Only Panels with PanelKindSingle
	focused       bool
	theme         *Theme
}

func NewPanelContainer(theme *Theme) *PanelContainer {
	root := &Panel{Kind: PanelKindEmpty}
	return &PanelContainer{
		root:     root,
		floating: make([]*Panel, 0, 3),
		selected: &root,
		theme:    theme,
	}
}

// ClearSelected makes the selected Panel empty, but does not delete it from
// the tree.
func (c *PanelContainer) ClearSelected() Component {
	item := (**c.selected).Left
	(**c.selected).Left = nil
	(**c.selected).Kind = PanelKindEmpty
	(*c.selected).UpdateSplits()
	return item
}

// DeleteSelected deletes the selected Panel and returns its child Component.
// If the selected Panel is the root Panel, ClearSelected() is called, instead.
func (c *PanelContainer) DeleteSelected() Component {
	if !(*c.selected).IsLeaf() {
		panic("selected is not leaf")
	}

	// If selected is root, just make it empty
	if *c.selected == c.root {
		return c.ClearSelected()
	} else {
		item := (**c.selected).Left
		p := (**c.selected).Parent
		if *c.selected == (*p).Left { // If we're deleting the parent's Left
			(*p).Left = (*p).Right
			(*p).Right = nil
		} else { // Deleting parent's Right
			(*p).Right = nil
		}
		(*p).Kind = PanelKindSingle
		(*c.selected) = nil // Tell garbage collector to come pick up selected (being safe)
		c.selected = &p
		(*c.selected).UpdateSplits()
		return item
	}
}

// SplitSelected splits the selected Panel with the given Component `item`.
// The type of split (vertical or horizontal) is determined with the `kind`.
// If `item` is nil, the new Panel will be of kind empty.
func (c *PanelContainer) SplitSelected(kind SplitKind, item Component) {
	if !(*c.selected).IsLeaf() {
		panic("selected is not leaf")
	}

	// It should be asserted that whatever is selected is either PanelKindEmpty or PanelKindSingle
	if item == nil {
		(**c.selected).Right = &Panel{Parent: *c.selected, Kind: PanelKindEmpty}
	} else {
		(**c.selected).Right = &Panel{Parent: *c.selected, Left: item, Kind: PanelKindSingle}
	}
	(**c.selected).Kind = PanelKind(kind)
	(*c.selected).UpdateSplits()
	panel := (**c.selected).Left.(*Panel) // TODO: watch me... might be a bug lurking in a hidden copy here
	c.selected = &panel
}

func (c *PanelContainer) GetSelected() Component {
	return (**c.selected).Left
}

func (c *PanelContainer) SetSelected(item Component) {
	if !(*c.selected).IsLeaf() {
		panic("selected is not leaf")
	}

	(**c.selected).Left = item
	(**c.selected).Kind = PanelKindSingle
	(*c.selected).UpdateSplits()
}

func (c *PanelContainer) FloatSelected() {

}

func (c *PanelContainer) UnfloatSelected() {

}

func (c *PanelContainer) Draw(s tcell.Screen) {
	c.root.Draw(s)
}

func (c *PanelContainer) SetFocused(v bool) {
	c.focused = v
	// TODO: update focused on selected children
}

func (c *PanelContainer) SetTheme(theme *Theme) {
	c.theme = theme
	c.root.SetTheme(theme)
	for i := range c.floating {
		c.floating[i].SetTheme(theme)
	}
}

func (c *PanelContainer) GetPos() (int, int) {
	return c.root.GetPos()
}

func (c *PanelContainer) SetPos(x, y int) {
	c.root.SetPos(x, y)
	c.root.UpdateSplits()
}

func (c *PanelContainer) GetMinSize() (int, int) {
	return c.root.GetMinSize()
}

func (c *PanelContainer) GetSize() (int, int) {
	return c.root.GetSize()
}

func (c *PanelContainer) SetSize(width, height int) {
	c.root.SetSize(width, height)
	c.root.UpdateSplits()
}

func (c *PanelContainer) HandleEvent(event tcell.Event) bool {
	// Call handle event on selected Panel
	return false
}
