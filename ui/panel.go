package ui

import "github.com/gdamore/tcell/v2"

// A PanelKind describes how to interpret the fields of a Panel.
type PanelKind uint8

const (
	PanelKindEmpty     PanelKind = iota
	PanelKindSingle              // Single item. Takes up all available space
	PanelKindSplitVert           // Items are above or below eachother
	PanelKindSplitHor            // Items are left or right of eachother
)

// A Panel represents a container for a split view between two items. The Kind
// tells how to interpret the Left and Right fields. The SplitAt is the column
// between 0 and width or height, representing the position of the split between
// the Left and Right, respectively.
//
// If the Kind is equal to PanelKindEmpty, then both Left and Right are nil.
// If the Kind is equal to PanelKindSingle, then only Left has value,
//   and its value will NOT be of type Panel. The SplitAt will not be used,
//   as the Left will take up the whole space.
// If the Kind is equal to PanelKindSplitVert, then both Left and Right will
//   have value, and they will both have to be of type Panel. The split will
//   be represented vertically, and the SplitAt spans 0 to height; top to bottom,
//   respectively.
// If the Kind is equal to PanelKindSplitHor, then both Left and Right will
//   have value, and they will both have to be of type Panel. The split will
//   be represented horizontally, and the SplitAt spans 0 to width; left to right.
type Panel struct {
	Parent  *Panel
	Left    Component
	Right   Component
	SplitAt int
	Kind    PanelKind
	Focused bool

	x, y   int
	width  int
	height int
}

// UpdateSplits uses the position and size of the Panel, along with its Weight
// and Kind, to appropriately size and place its children. It calls UpdateSplits()
// on its child Panels.
func (p *Panel) UpdateSplits() {
	switch p.Kind {
	case PanelKindSingle:
		p.Left.SetPos(p.x, p.y)
		p.Left.SetSize(p.width, p.height)
	case PanelKindSplitVert:
		p.Left.SetPos(p.x, p.y)
		p.Left.SetSize(p.width, p.SplitAt)
		p.Right.SetPos(p.x, p.y+p.SplitAt)
		p.Right.SetSize(p.width, p.height-p.SplitAt)

		p.Left.(*Panel).UpdateSplits()
		p.Right.(*Panel).UpdateSplits()
	case PanelKindSplitHor:
		p.Left.SetPos(p.x, p.y)
		p.Left.SetSize(p.SplitAt, p.height)
		p.Right.SetPos(p.x+p.SplitAt, p.y)
		p.Right.SetSize(p.width-p.SplitAt, p.height)

		p.Left.(*Panel).UpdateSplits()
		p.Right.(*Panel).UpdateSplits()
	}
}

// Same as EachLeaf, but returns true if any call to `f` returned true.
func (p *Panel) eachLeaf(rightMost bool, f func(*Panel) bool) bool {
	switch p.Kind {
	case PanelKindEmpty:
		fallthrough
	case PanelKindSingle:
		return f(p)

	case PanelKindSplitVert:
		fallthrough
	case PanelKindSplitHor:
		if rightMost {
			if p.Right.(*Panel).eachLeaf(rightMost, f) {
				return true
			}
			return p.Left.(*Panel).eachLeaf(rightMost, f)
		} else {
			if p.Left.(*Panel).eachLeaf(rightMost, f) {
				return true
			}
			return p.Right.(*Panel).eachLeaf(rightMost, f)
		}

	default:
		return false
	}
}

// EachLeaf visits the entire tree, and calls function `f` at each leaf Panel.
// If the function `f` returns true, then visiting stops. if `rtl` is true,
// the tree is traversed in right-most order. The default is to traverse
// in left-most order.
//
// The caller of this function can safely assert that Panel's Kind is always
// either `PanelKindSingle` or `PanelKindEmpty`.
func (p *Panel) EachLeaf(rightMost bool, f func(*Panel) bool) {
	p.eachLeaf(rightMost, f)
}

// IsLeaf returns whether the Panel is a leaf or not. A leaf is a panel with
// Kind `PanelKindEmpty` or `PanelKindSingle`.
func (p *Panel) IsLeaf() bool {
	switch p.Kind {
	case PanelKindEmpty:
		fallthrough
	case PanelKindSingle:
		return true
	default:
		return false
	}
}

func (p *Panel) Draw(s tcell.Screen) {
	switch p.Kind {
	case PanelKindSplitVert:
		fallthrough
	case PanelKindSplitHor:
		p.Right.Draw(s)
		fallthrough
	case PanelKindSingle:
		p.Left.Draw(s)
	}
}

// SetFocused sets this Panel's Focused field to `v`. Then, if the Panel's Kind
// is PanelKindSingle, it sets its child (not a Panel) focused to `v`, also.
func (p *Panel) SetFocused(v bool) {
	p.Focused = v
	switch p.Kind {
	case PanelKindSplitVert:
		fallthrough
	case PanelKindSplitHor:
		p.Right.SetFocused(v)
		fallthrough
	case PanelKindSingle:
		p.Left.SetFocused(v)
	}
}

func (p *Panel) SetTheme(theme *Theme) {
	switch p.Kind {
	case PanelKindSplitVert:
		fallthrough
	case PanelKindSplitHor:
		p.Right.SetTheme(theme)
		fallthrough
	case PanelKindSingle:
		p.Left.SetTheme(theme)
	}
}

// GetPos returns the position of the panel.
func (p *Panel) GetPos() (int, int) {
	return p.width, p.height
}

// SetPos sets the position of the panel.
func (p *Panel) SetPos(x, y int) {
	p.x, p.y = x, y
}

// GetMinSize returns the combined minimum sizes of the Panel's children.
func (p *Panel) GetMinSize() (int, int) {
	switch p.Kind {
	case PanelKindSingle:
		return p.Left.GetMinSize()
	case PanelKindSplitVert:
		// use max width, add heights
		lWidth, lHeight := p.Left.GetMinSize()
		rWidth, rHeight := p.Right.GetMinSize()
		return Max(lWidth, rWidth), lHeight + rHeight
	case PanelKindSplitHor:
		// use max height, add widths
		lWidth, lHeight := p.Left.GetMinSize()
		rWidth, rHeight := p.Right.GetMinSize()
		return lWidth + rWidth, Max(lHeight, rHeight)
	default:
		return 0, 0
	}
}

func (p *Panel) GetSize() (int, int) {
	return p.width, p.height
}

// SetSize sets the Panel size to the given width, and height. It will not check
// against GetMinSize() because it may be costly to do so. SetSize clamps the
// Panel's SplitAt to be within the new size of the Panel.
func (p *Panel) SetSize(width, height int) {
	p.width, p.height = width, height
	switch p.Kind {
	case PanelKindSplitVert:
		p.SplitAt = Min(p.SplitAt, height)
	case PanelKindSplitHor:
		p.SplitAt = Min(p.SplitAt, width)
	}
}

// HandleEvent propogates the event to all children, calling HandleEvent()
// on left-most items. As usual: returns true if handled, false if unhandled.
// This function relies on the behavior of the child Components to only handle
// events if they are focused.
func (p *Panel) HandleEvent(event tcell.Event) bool {
	switch p.Kind {
	case PanelKindSingle:
		return p.Left.HandleEvent(event)
	case PanelKindSplitVert:
		fallthrough
	case PanelKindSplitHor:
		if p.Left.HandleEvent(event) {
			return true
		}
		return p.Right.HandleEvent(event)
	default:
		return false
	}
}
