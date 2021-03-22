package buffer

import "testing"

func TestRopeInserting(t *testing.T) {
	var buf Buffer = NewRopeBuffer([]byte("some"))
	buf.Insert(0, 4, []byte(" text")) // Insert " text" after "some"
	buf.Insert(0, 0, []byte("with\n\t"))
	// "with\n\tsome text"

	buf.Remove(0, 4, 1, 5) // Delete from line 0, col 4, to line 1, col 6 "\n\tsome "

	if str := string(buf.Bytes()); str != "withtext" {
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

	line, col := buf.ClampLineCol(15, 5) // Should become last line, first column
	if line != 4 && col != 0 {
		t.Errorf("Expected to clamp line col to 4,0 got %v,%v", line, col)
		t.Fail()
	}

	if line := string(buf.Line(2)); line != "\tsome\n" {
		t.Errorf("Expected line 3 to equal \"\\tsome\", got %#v", line)
		t.Fail()
	}
}
