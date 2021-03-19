package main

import "github.com/zyedidia/clipboard"

type ClipMethod uint8

const (
	ClipExternal ClipMethod = iota
	_
	ClipInternal
)

var ClipCurrentMethod ClipMethod

var internalClipboard string

// ClipInitialize will initialize the clipboard for the given method first,
// and if that fails, an internal method will be chosen, instead. The Method
// chosen is returned along with any error that may have occurred while
// selecting the method. The error is not fatal because an internal method
// is used.
func ClipInitialize(m ClipMethod) (ClipMethod, error) {
	err := clipboard.Initialize()
	if err != nil {
		ClipCurrentMethod = ClipInternal
		return ClipInternal, err
	}
	ClipCurrentMethod = ClipExternal
	return ClipExternal, nil
}

// ClipRead receives the clipboard contents using the ClipCurrentMethod.
func ClipRead() (string, error) {
	switch ClipCurrentMethod {
	case ClipExternal:
		return clipboard.ReadAll("clipboard")
	case ClipInternal:
		return internalClipboard, nil
	}
	panic("How did execution get here?")
}

// ClipWrite sets the clipboard contents using the ClipCurrentMethod.
func ClipWrite(content string) error {
	switch ClipCurrentMethod {
	case ClipExternal:
		return clipboard.WriteAll(content, "clipboard")
	case ClipInternal:
		internalClipboard = content
		return nil
	}
	panic("How did execution get here?")
}
