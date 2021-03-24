package buffer

import "testing"

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
		t.Fail()
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
		t.Fail()
	}

	if len := buf.RunesInLine(1); len != 6 { // "is" in English and in japanese
		t.Errorf("Expected 6 runes in line 2, found %v", len)
		t.Fail()
	}

	if len := buf.RunesInLineWithDelim(4); len != 0 {
		t.Errorf("Expected 0 runes in line 5, found %v", len)
		t.Fail()
	}

	line, col := buf.ClampLineCol(15, 5) // Should become last line, first column
	if line != 4 && col != 0 {
		t.Errorf("Expected to clamp line col to 4,0 got %v,%v", line, col)
		t.Fail()
	}

	line, col = buf.ClampLineCol(4, -1)
	if line != 4 && col != 0 {
		t.Errorf("Expected to clamp line col to 4,0 got %v,%v", line, col)
		t.Fail()
	}

	line, col = buf.ClampLineCol(2, 5) // Should be third line, pointing at the newline char
	if line != 2 && col != 5 {
		t.Errorf("Expected to clamp line, col to 2,5 got %v,%v", line, col)
		t.Fail()
	}

	if line := string(buf.Line(2)); line != "\tsome\n" {
		t.Errorf("Expected line 3 to equal \"\\tsome\", got %#v", line)
		t.Fail()
	}

	if line := string(buf.Line(4)); line != "" {
		t.Errorf("Got %#v", line)
		t.Fail()
	}
}

func TestRopeCount(t *testing.T) {
	var buf Buffer = NewRopeBuffer([]byte("\t\tlot of\n\ttabs"))

	tabsAtOf := buf.Count(0, 0, 0, 7, []byte{'\t'})
	if tabsAtOf != 2 {
		t.Errorf("Expected 2 tabs before 'of', got %#v", tabsAtOf)
		t.Fail()
	}

	tabs := buf.Count(0, 0, 0, 0, []byte{'\t'})
	if tabs != 0 {
		t.Errorf("Expected no tabs at column zero, got %v", tabs)
		t.Fail()
	}
}
