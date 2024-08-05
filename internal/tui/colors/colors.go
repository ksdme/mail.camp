package colors

import "github.com/charmbracelet/lipgloss"

type ColorPalette struct {
	Accent lipgloss.Color
	Muted  lipgloss.Color
	Text   lipgloss.Color
}

func DefaultColorPalette() ColorPalette {
	return ColorPalette{
		Accent: lipgloss.Color("212"),
		Muted:  lipgloss.Color("244"),
		Text:   lipgloss.Color("255"),
	}
}
