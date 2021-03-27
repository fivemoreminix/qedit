package buffer

import "regexp"

type Syntax uint8

const (
	Default Syntax = iota
	Keyword
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
	Rules     map[*regexp.Regexp]Syntax
	// TODO: add other language details
}
