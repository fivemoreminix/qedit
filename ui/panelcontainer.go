package ui

import "github.com/gdamore/tcell/v2"

type SplitKind uint8

const (
	SplitVertical SplitKind = SplitKind(PanelKindSplitVert) + iota
	SplitHorizontal
)

type PanelContainer struct {
	root                    *Panel
	floating                []*Panel
	selected                **Panel // Only Panels with PanelKindSingle
	lastNonFloatingSelected **Panel // Used only when focused on floating Panels
	floatingMode            bool    // True if 'selected' is part of a floating Panel
	focused                 bool
	theme                   *Theme
}

func NewPanelContainer(theme *Theme) *PanelContainer {
	root := &Panel{Kind: PanelKindEmpty}
	return &PanelContainer{
		root:            root,
		floating:        make([]*Panel, 0, 3),
		selected:        &root,
		theme:           theme,
	}
}

// ClearSelected makes the selected Panel empty, but does not delete it from
// the tree.
func (c *PanelContainer) ClearSelected() Component {
	item := (**c.selected).Left
	(**c.selected).Left = nil
	(**c.selected).Kind = PanelKindEmpty
	if p := (**c.selected).Parent; p != nil {
		p.UpdateSplits()
	}
	return item
}

// DeleteSelected deletes the selected Panel and returns its child Component.
// If the selected Panel is the root Panel, ClearSelected() is called, instead.
func (c *PanelContainer) DeleteSelected() Component {
	if !(*c.selected).IsLeaf() {
		panic("selected is not leaf")
	}

	// If selected is the root, just make it empty
	if *c.selected == c.root {
		return c.ClearSelected()
	} else {
		item := (**c.selected).Left
		p := (**c.selected).Parent

		if c.focused {
			(*c.selected).SetFocused(false) // Unfocus item
		}

		if p != nil {
			if *c.selected == (*p).Left { // If we're deleting the parent's Left
				(*p).Left = (*p).Right
				(*p).Right = nil
			} else { // Deleting parent's Right
				(*p).Right = nil
			}

			if (*p).Left != nil {
				panel := (*p).Left.(*Panel)
				(*p).Left = (*panel).Left
				(*p).Right = (*panel).Right
				(*p).Kind = (*panel).Kind
			} else {
				(*p).Kind = PanelKindEmpty
			}
			c.selected = &p
			(*p).UpdateSplits()
		} else if c.floatingMode { // Deleting a floating Panel without a parent
			c.floating[0] = nil
			copy(c.floating, c.floating[1:]) // Shift items to front
			c.floating = c.floating[:len(c.floating)-1] // Shrink slice's len by one

			if len(c.floating) <= 0 {
				c.SetFloatingFocused(false)
			} else {
				c.selected = &c.floating[0]
			}
		} else {
			panic("Panel does not have parent and is not floating")
		}
		
		if c.focused {
			(*c.selected).SetFocused(c.focused)
		}

		return item
	}
}

// SwapNeighborsSelected swaps two Left and Right child Panels of a vertical or
// horizontally split Panel. This is necessary to achieve a "split top" or
// "split left" effect, as Panels only split open to the bottom or right.
func (c *PanelContainer) SwapNeighborsSelected() {
	parent := (**c.selected).Parent
	if parent != nil {
		left := (*parent).Left
		(*parent).Left = parent.Right
		(*parent).Right = left
		parent.UpdateSplits() // Updates position and size of reordered children
	}
}

// Turns the selected Panel into a split panel, moving its contents to its Left field,
// and putting the given Panel at the Right field. `panel` cannot be nil.
func (c *PanelContainer) splitSelectedWithPanel(kind SplitKind, panel *Panel) {
	(**c.selected).Left = &Panel{Parent: *c.selected, Left: (**c.selected).Left, Kind: (**c.selected).Kind}
	(**c.selected).Right = panel
	(**c.selected).Right.(*Panel).Parent = *c.selected

	// Update parent's split information
	(**c.selected).Kind = PanelKind(kind)
	if kind == SplitVertical {
		(**c.selected).SplitAt = (**c.selected).height / 2
	} else {
		(**c.selected).SplitAt = (**c.selected).width / 2
	}
	(*c.selected).UpdateSplits()

	// Change selected from parent to the previously selected Panel on the Left
	if c.focused {
		(*c.selected).SetFocused(false)
	}
	panel = (**c.selected).Left.(*Panel)
	c.selected = &panel
	if c.focused {
		(*c.selected).SetFocused(c.focused)
	}
}

// SplitSelected splits the selected Panel with the given Component `item`.
// The type of split (vertical or horizontal) is determined with the `kind`.
// If `item` is nil, the new Panel will be of kind empty.
func (c *PanelContainer) SplitSelected(kind SplitKind, item Component) {
	if !(*c.selected).IsLeaf() {
		panic("selected is not leaf")
	}

	if item == nil {
		c.splitSelectedWithPanel(kind, &Panel{Parent: *c.selected, Kind: PanelKindEmpty})
	} else {
		c.splitSelectedWithPanel(kind, &Panel{Parent: *c.selected, Left: item, Kind: PanelKindSingle})
	}
}

func (c *PanelContainer) IsRootSelected() bool {
	return *c.selected == c.root
}

func (c *PanelContainer) GetSelected() Component {
	if !(*c.selected).IsLeaf() {
		panic("selected is not leaf")
	}
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

func (c *PanelContainer) raiseFloating(idx int) {
	item := c.floating[idx]
	copy(c.floating[1:], c.floating[:idx]) // Shift all items before idx right
	c.floating[0] = item
}

// GetFloatingFocused returns true if a floating window is selected or focused.
func (c *PanelContainer) GetFloatingFocused() bool {
	return c.floatingMode
}

// SetFloatingFocused sets whether the floating Panels are focused. When true,
// the current Panel will be unselected and the front floating Panel will become
// the new selected if there any floating windows. If false, the same, but the
// last selected non-floating Panel will become focused.
//
// The returned boolean is whether floating windows were able to be focused. If
// there are no floating windows when trying to focus them, this will inevitably
// return false, for example.
func (c *PanelContainer) SetFloatingFocused(v bool) bool {
	if v {
		if len(c.floating) > 0 {
			if c.focused {
				(*c.selected).SetFocused(false) // Unfocus in-tree window
			}
			c.lastNonFloatingSelected = c.selected
			c.selected = &c.floating[0]
			if c.focused {
				(*c.selected).SetFocused(true)
			}
			c.floatingMode = true
			return true
		}
	} else {
		if c.focused {
			(*c.selected).SetFocused(false) // Unfocus floating window
		}
		c.selected = c.lastNonFloatingSelected
		if c.focused {
			(*c.selected).SetFocused(true) // Focus in-tree window
		}
		c.floatingMode = false
	}
	return false
}

// FloatSelected makes the selected Panel floating. This function does not focus
// the newly floated Panel. To focus the floating panel, call SetFloatingFocused().
func (c *PanelContainer) FloatSelected() {
	if !(*c.selected).IsLeaf() {
		panic("selected is not leaf")
	}

	if c.floatingMode {
		return
	}

	panel := *c.selected
	c.DeleteSelected()
	(*c.selected).UpdateSplits()
	panel.Parent = nil
	panel.UpdateSplits()

	c.floating = append(c.floating, panel)
	c.raiseFloating(len(c.floating)-1)
}

// UnfloatSelected moves any selected floating Panel to the normal tree that is
// accessible in the standard focus mode. This function will cause focus to go to
// the normal tree if there are no remaining floating windows after the operation.
//
// Like SetFloatingFocused(), the boolean returned is whether the PanelContainer
// is focusing floating windows after the operation.
func (c *PanelContainer) UnfloatSelected(kind SplitKind) bool {
	if !(*c.selected).IsLeaf() {
		panic("selected is not leaf")
	}

	if !c.floatingMode {
		return false
	}

	panel := *c.selected
	c.DeleteSelected()
	c.SetFloatingFocused(false)
	c.splitSelectedWithPanel(kind, panel)

	// Try to return to floating focus
	return c.SetFloatingFocused(true)
}

func (c *PanelContainer) Draw(s tcell.Screen) {
	c.root.Draw(s)
	for i := len(c.floating)-1; i >= 0; i-- {
		c.floating[i].Draw(s)
	}
}

func (c *PanelContainer) SetFocused(v bool) {
	c.focused = v
	(*c.selected).SetFocused(v)
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
	return (*c.selected).HandleEvent(event)
}
