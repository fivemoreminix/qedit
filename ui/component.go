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
	GetPos() (int, int)
	// Set position of the Component.
	SetPos(int, int)

	// Returns the smallest size the Component can be.
	GetMinSize() (int, int)
	// Get size of the Component.
	GetSize() (int, int)
	// Set size of the component. If size is smaller than minimum, minimum is
	// used, instead.
	SetSize(int, int)

	// It is good practice for a Component to check if it is focused before handling
	// events.
	tcell.EventHandler // A Component can handle events
}
