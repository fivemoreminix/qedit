package ui

// Align defines the text alignment of a label.
type Align uint8

const (
	// AlignLeft is the normal text alignment where text is aligned to the left
	// of its bounding box.
	AlignLeft Align = iota
	// AlignRight causes text to be aligned to the right of its bounding box.
	AlignRight
	// AlignJustify causes text to be left-aligned, but also spaced so that it
	// fits the entire box where it is being rendered.
	AlignJustify
)

// A Label is a component for rendering text. Text can be rendered easily
// without a Label, but this component forces the text to fit within its
// bounding box and allows for left-align, right-align, and justify.
type Label struct {
	Text          string
	Alignment     Align

	baseComponent
}
