package buffer

type Syntax uint8

const (
	Default Syntax = iota
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
