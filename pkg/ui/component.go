package ui

import (
	"github.com/gdamore/tcell/v2"
)

// A Component refers generally to the behavior of a UI "component". Components
// include buttons, input fields, and labels. It is expected that after constructing
// a component, to call the SetPos() function, and possibly SetSize() as well.
//
// Many components implement their own `New...()` function. In those constructor
// functions, it is good practice for that component to set its size to be its
// minimum size.
type Component interface {
	// A component knows its position and size, which is used to draw itself in
	// its bounding rectangle.
	Draw(tcell.Screen)
	// Components can be focused, which may affect how it handles events or draws.
	// For example, when a button is focused, the Return key may be pressed to
	// activate the button.
	SetFocused(bool)
	// Applies the theme to the component and all of its children.
	SetTheme(*Theme)

	// Get position of the Component.
	GetPos() (x, y int)
	// Set position of the Component.
	SetPos(x, y int)

	// Returns the smallest size the Component can be.
	GetMinSize() (w, h int)
	// Get size of the Component.
	GetSize() (w, h int)
	// Set size of the component. If size is smaller than minimum, minimum is
	// used, instead.
	SetSize(w, h int)

	// HandleEvent tells the Component to handle the provided event. The Component
	// should only handle events if it is focused. An event can optionally be
	// handled. If an event is handled, the function should return true. If the
	// event went unhandled, the function should return false.
	HandleEvent(tcell.Event) bool
}

// baseComponent can be embedded in a Component's struct to hide a few of the
// boilerplate fields and functions. The baseComponent defines defaults for
// ...Pos(), ...Size(), SetFocused(), and SetTheme() functions that can be
// overriden.
type baseComponent struct {
	focused       bool
	x, y          int
	width, height int
	theme         *Theme
}

func (c *baseComponent) SetFocused(v bool) {
	c.focused = v
}

func (c *baseComponent) SetTheme(theme *Theme) {
	c.theme = theme
}

func (c *baseComponent) GetPos() (int, int) {
	return c.x, c.y
}

func (c *baseComponent) SetPos(x, y int) {
	c.x, c.y = x, y
}

func (c *baseComponent) GetMinSize() (int, int) {
	return 0, 0
}

func (c *baseComponent) GetSize() (int, int) {
	return c.width, c.height
}

func (c *baseComponent) SetSize(width, height int) {
	c.width, c.height = width, height
}
