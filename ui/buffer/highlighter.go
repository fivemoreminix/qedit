package buffer

import (
	"regexp"
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

type RegexpRegion struct {
	Start    *regexp.Regexp
	End      *regexp.Regexp // Should be "$" by default
	Skip     *regexp.Regexp // Optional
	Error    *regexp.Regexp // Optional
	Specials []*regexp.Regexp // Optional (nil or zero len)
}

type Match struct {
	Col     int
	EndLine int // Inclusive
	EndCol  int // Inclusive
	Syntax  Syntax
}

// ByCol implements sort.Interface for []Match based on the Col field.
type ByCol []Match

func (c ByCol) Len() int           { return len(c) }
func (c ByCol) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }
func (c ByCol) Less(i, j int) bool { return c[i].Col < c[j].Col }

// A Highlighter can answer how to color any part of a provided Buffer. It does so
// by applying regular expressions over a region of the buffer.
type Highlighter struct {
	Buffer      Buffer
	Language    *Language
	Colorscheme *Colorscheme

	lineMatches [][]Match
}

func NewHighlighter(buffer Buffer, lang *Language, colorscheme *Colorscheme) *Highlighter {
	return &Highlighter{
		buffer,
		lang,
		colorscheme,
		make([][]Match, buffer.Lines()),
	}
}

// UpdateLines forces the highlighting matches for lines between startLine to
// endLine, inclusively, to be updated. It is more efficient to mark lines as
// invalidated when changes occur and call UpdateInvalidatedLines(...).
func (h *Highlighter) UpdateLines(startLine, endLine int) {
	if lines := h.Buffer.Lines(); len(h.lineMatches) < lines {
		h.lineMatches = append(h.lineMatches, make([][]Match, lines)...) // Extend
	}
	for i := startLine; i <= endLine && i < len(h.lineMatches); i++ {
		if h.lineMatches[i] != nil {
			h.lineMatches[i] = h.lineMatches[i][:0] // Shrink slice to zero (hopefully save allocs)
		}
	}

	// If the rule k does not have an End, then it can be optimized that we search from the start
	// of view until the end of view. For any k that has an End, we search for ends from start
	// of view, backtracking when one is found, to fulfill a multiline highlight.

	endLine, endCol := h.Buffer.ClampLineCol(endLine, (h.Buffer).RunesInLineWithDelim(endLine)-1)
	startPos := h.Buffer.LineColToPos(startLine, 0)
	bytes := h.Buffer.Slice(startLine, 0, endLine, endCol)

	for k, v := range h.Language.Rules {
		var indexes [][]int // [][2]int
		if k.End != nil && k.End.String() != "$" { // If this range might be a multiline range...
			endIndexes := k.End.FindAllIndex(bytes, -1) // Attempt to find every ending match
			startIndexes := k.Start.FindAllIndex(bytes, -1) // Attempt to find every starting match
			// ...
			_ = endIndexes
			_ = startIndexes
		} else { // A standard single-line match
			indexes = k.Start.FindAllIndex(bytes, -1) // Attempt to find the start match
		}

		if indexes != nil {
			for i := range indexes {
				startLine, startCol := h.Buffer.PosToLineCol(indexes[i][0] + startPos)
				endLine, endCol := h.Buffer.PosToLineCol(indexes[i][1]-1 + startPos)

				match := Match { startCol, endLine, endCol, v }

				h.lineMatches[startLine] = append(h.lineMatches[startLine], match) // Unsorted
			}
		}
	}

	h.validateLines(startLine, endLine) // Marks any "unvalidated" or nil lines as valued
}

// UpdateInvalidatedLines only updates the highlighting for lines that are invalidated
// between lines startLine and endLine, inclusively.
func (h *Highlighter) UpdateInvalidatedLines(startLine, endLine int) {
	// Move startLine to first line with invalidated changes
	for startLine <= endLine && startLine < len(h.lineMatches)-1 {
		if h.lineMatches[startLine] == nil {
			break
		}
		startLine++
	}

	// Move endLine back to first line at or before endLine with invalidated changes
	for endLine >= startLine && endLine > 0 {
		if h.lineMatches[endLine] == nil {
			break
		}
		endLine--
	}

	if startLine > endLine {
		return // Do nothing; no invalidated lines
	}

	h.UpdateLines(startLine, endLine)
}

func (h *Highlighter) HasInvalidatedLines(startLine, endLine int) bool {
	for i := startLine; i <= endLine && i < len(h.lineMatches); i++ {
		if h.lineMatches[i] == nil {
			return true
		}
	}
	return false
}

func (h *Highlighter) validateLines(startLine, endLine int) {
	for i := startLine; i <= endLine && i < len(h.lineMatches); i++ {
		if h.lineMatches[i] == nil {
			h.lineMatches[i] = make([]Match, 0)
		}
	}
}

func (h *Highlighter) InvalidateLines(startLine, endLine int) {
	for i := startLine; i <= endLine && i < len(h.lineMatches); i++ {
		h.lineMatches[i] = nil
	}
}

func (h *Highlighter) GetLineMatches(line int) []Match {
	if line < 0 || line >= len(h.lineMatches) {
		return nil
	}
	data := h.lineMatches[line]
	sort.Sort(ByCol(data))
	return data
}

func (h *Highlighter) GetStyle(match Match) tcell.Style {
	return h.Colorscheme.GetStyle(match.Syntax)
}
