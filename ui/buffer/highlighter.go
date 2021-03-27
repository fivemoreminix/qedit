package buffer

import (
	"sort"

	"github.com/gdamore/tcell/v2"
)

type Colorscheme map[Syntax]tcell.Style

// Gets the tcell.Style from the Colorscheme map for the given Syntax.
// If the Syntax cannot be found in the map, either the `Default` Syntax
// is used, or `tcell.DefaultStyle` is returned if the Default is not assigned.
func (c *Colorscheme) GetStyle(s Syntax) tcell.Style {
	if c != nil {
		if val, ok := (*c)[s]; ok {
			return val // Try to return the requested value
		} else if s != Default {
			if val, ok := (*c)[Default]; ok {
				return val // Use default colorscheme value, instead
			}
		}
	}

	return tcell.StyleDefault; // No value for Default; use default style.
}

type SyntaxData struct {
	Col     int
	EndLine int
	EndCol  int
	Syntax  Syntax
}

// ByCol implements sort.Interface for []SyntaxData based on the Col field.
type ByCol []SyntaxData

func (c ByCol) Len() int           { return len(c) }
func (c ByCol) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }
func (c ByCol) Less(i, j int) bool { return c[i].Col < c[j].Col }

// A Highlighter can answer how to color any part of a provided Buffer. It does so
// by applying regular expressions over a region of the buffer.
type Highlighter struct {
	Buffer      Buffer
	Language    *Language
	Colorscheme *Colorscheme

	lineData [][]SyntaxData
}

func NewHighlighter(buffer Buffer, lang *Language, colorscheme *Colorscheme) *Highlighter {
	return &Highlighter{
		buffer,
		lang,
		colorscheme,
		make([][]SyntaxData, buffer.Lines()),
	}
}

func (h *Highlighter) Update() {
	if lines := h.Buffer.Lines(); len(h.lineData) < lines {
		h.lineData = append(h.lineData, make([][]SyntaxData, lines)...) // Extend
	}
	for i := range h.lineData { // Invalidate all line data
		h.lineData[i] = nil
	}

	// For each compiled syntax regex:
	//   Use FindAllIndex to get all instances of a single match, then for each match found:
	//     use Find to get the bytes of the match and get the length. Calculate to what line
	//     and column the bytes span and its syntax. Append a SyntaxData to the output.

	bytes := (h.Buffer).Bytes() // Allocates size of the buffer	

	for k, v := range h.Language.Rules {
		indexes := k.FindAllIndex(bytes, -1)
		if indexes != nil {
			for i := range indexes {
				endPos := indexes[i][1] - 1
				startLine, startCol := h.Buffer.PosToLineCol(indexes[i][0])
				endLine, endCol := h.Buffer.PosToLineCol(endPos)

				syntaxData := SyntaxData { startCol, endLine, endCol, v }

				h.lineData[startLine] = append(h.lineData[startLine], syntaxData) // Not sorted
			}
		}
	}
}

func (h *Highlighter) GetLine(line int) []SyntaxData {
	if line < 0 || line >= len(h.lineData) {
		return nil
	}
	data := h.lineData[line]
	sort.Sort(ByCol(data))
	return data
}

func (h *Highlighter) GetStyle(syn SyntaxData) tcell.Style {
	return h.Colorscheme.GetStyle(syn.Syntax)
}
