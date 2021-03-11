package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
)

// A Theme is a map of string names to styles. Themes can be passed by reference to components
// to set their styles. Some components will depend upon the basic keys, but most components
// may use keys specific to their component. If a theme value cannot be found, then the
// `DefaultTheme` value will be used, instead. An updated list of theme keys can be found on
// the default theme.
type Theme map[string]tcell.Style

func (theme *Theme) GetOrDefault(key string) tcell.Style {
	if theme != nil {
		if val, ok := (*theme)[key]; ok {
			return val
		}
	}

	if val, ok := DefaultTheme[key]; ok {
		return val
	} else {
		panic(fmt.Sprintf("key \"%v\" not present in default theme", key))
	}
}

// DefaultTheme uses only the first 16 colors present in most colored terminals.
var DefaultTheme = Theme{
	"Normal":          tcell.Style{}.Foreground(tcell.ColorSilver).Background(tcell.ColorBlack),
	"Button":          tcell.Style{}.Foreground(tcell.ColorBlack).Background(tcell.ColorWhite),
	"InputField":      tcell.Style{}.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack),
	"MenuBar":         tcell.Style{}.Foreground(tcell.ColorBlack).Background(tcell.ColorSilver),
	"MenuBarSelected": tcell.Style{}.Foreground(tcell.ColorSilver).Background(tcell.ColorBlack),
	"Menu":            tcell.Style{}.Foreground(tcell.ColorBlack).Background(tcell.ColorSilver),
	"MenuSelected":    tcell.Style{}.Foreground(tcell.ColorSilver).Background(tcell.ColorBlack),
	"QuickChar":       tcell.Style{}.Foreground(tcell.ColorYellow).Background(tcell.ColorBlack),
	"Tab":             tcell.Style{}.Foreground(tcell.ColorSilver).Background(tcell.ColorBlack),
	"TabContainer":    tcell.Style{}.Foreground(tcell.ColorSilver).Background(tcell.ColorBlack),
	"TabSelected":     tcell.Style{}.Foreground(tcell.ColorBlack).Background(tcell.ColorSilver),
	"TextEdit":        tcell.Style{}.Foreground(tcell.ColorSilver).Background(tcell.ColorBlack),
	"TextEditColumn":  tcell.Style{}.Foreground(tcell.ColorBlack).Background(tcell.ColorGray),
	"Window":          tcell.Style{}.Foreground(tcell.ColorBlack).Background(tcell.ColorSilver),
	"WindowHeader":    tcell.Style{}.Foreground(tcell.ColorBlack).Background(tcell.ColorWhite),
}
