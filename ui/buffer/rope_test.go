package buffer

import "testing"

func TestRopePosToLineCol(t *testing.T) {
	var buf Buffer = NewRopeBuffer([]byte("line0\nline1\n\nline3\n"))
	//line0
	//line1
	//
	//line3
	//

	startLine, startCol := buf.PosToLineCol(0)
	if startLine != 0 {
		t.Errorf("Expected startLine == 0, got %v", startLine)
	}

	if startCol != 0 {
		t.Errorf("Expected startCol == 0, got %v", startCol)
	}

	endPos := buf.Len() - 1
	endLine, endCol := buf.PosToLineCol(endPos)
	t.Logf("endPos = %v", endPos)
	if endLine != 3 {
		t.Errorf("Expected endLine == 3, got %v", endLine)
	}

	if endCol != 5 {
		t.Errorf("Expected endCol == 5, got %v", endCol)
	}

	line1Pos := 11 // Byte index of the delim separating line1 and line 2
	line1Line, line1Col := buf.PosToLineCol(line1Pos)
	if line1Line != 1 {
		t.Errorf("Expected line1Line == 1, got %v", line1Line)
	}

	if line1Col != 5 {
		t.Errorf("Expected line1Col == 5, got %v", line1Col)
	}
}

func TestRopeInserting(t *testing.T) {
	var buf Buffer = NewRopeBuffer([]byte("some"))
	buf.Insert(0, 4, []byte(" text\n")) // Insert " text" after "some"
	buf.Insert(0, 0, []byte("with\n\t"))
	//with
	//	some text
	//

	buf.Remove(0, 4, 1, 5) // Delete from line 0, col 4, to line 1, col 6 "\n\tsome "

	if str := string(buf.Bytes()); str != "withtext\n" {
		t.Errorf("string does not match \"withtext\", got %#v", str)
	}
}

func TestRopeBounds(t *testing.T) {
	var buf Buffer = NewRopeBuffer([]byte("this\nis (は)\n\tsome\ntext\n"))
	//this
	//is (は)
	//	some
	//text
	//

	if buf.Lines() != 5 {
		t.Errorf("Expected buf.Lines() == 5")
	}

	if len := buf.RunesInLine(1); len != 6 { // "is" in English and in japanese
		t.Errorf("Expected 6 runes in line 2, found %v", len)
	}

	if len := buf.RunesInLineWithDelim(4); len != 0 {
		t.Errorf("Expected 0 runes in line 5, found %v", len)
	}

	line, col := buf.ClampLineCol(15, 5) // Should become last line, first column
	if line != 4 && col != 0 {
		t.Errorf("Expected to clamp line col to 4,0 got %v,%v", line, col)
	}

	line, col = buf.ClampLineCol(4, -1)
	if line != 4 && col != 0 {
		t.Errorf("Expected to clamp line col to 4,0 got %v,%v", line, col)
	}

	line, col = buf.ClampLineCol(2, 5) // Should be third line, pointing at the newline char
	if line != 2 && col != 5 {
		t.Errorf("Expected to clamp line, col to 2,5 got %v,%v", line, col)
	}

	if line := string(buf.Line(2)); line != "\tsome\n" {
		t.Errorf("Expected line 3 to equal \"\\tsome\", got %#v", line)
	}

	if line := string(buf.Line(4)); line != "" {
		t.Errorf("Got %#v", line)
	}
}

func TestRopeCount(t *testing.T) {
	var buf Buffer = NewRopeBuffer([]byte("\t\tlot of\n\ttabs"))

	tabsAtOf := buf.Count(0, 0, 0, 7, []byte{'\t'})
	if tabsAtOf != 2 {
		t.Errorf("Expected 2 tabs before 'of', got %#v", tabsAtOf)
	}

	tabs := buf.Count(0, 0, 0, 0, []byte{'\t'})
	if tabs != 0 {
		t.Errorf("Expected no tabs at column zero, got %v", tabs)
	}
}

func TestRopeSlice(t *testing.T) {
	var buf Buffer = NewRopeBuffer([]byte("abc\ndef\n"))

	wholeSlice := buf.Slice(0, 0, 2, 0) // Position points to after the newline char
	if string(wholeSlice) != "abc\ndef\n" {
		t.Errorf("Whole slice was not equal, got \"%s\"", wholeSlice)
	}

	secondLine := buf.Slice(1, 0, 1, 3)
	if string(secondLine) != "def\n" {
		t.Errorf("Second line and slice were not equal, got \"%s\"", secondLine)
	}
}
