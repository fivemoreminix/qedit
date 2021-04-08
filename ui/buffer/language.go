package buffer

type Syntax uint8

const (
	Default Syntax = iota
	Column // Not necessarily a Syntax; useful for Colorscheming editor column
	Keyword
	String
	Special
	Type
	Number
	Builtin
	Comment
	DocComment
)

type Language struct {
	Name      string
	Filetypes []string // .go, .c, etc.
	Rules     map[*RegexpRegion]Syntax
	// TODO: add other language details
}
