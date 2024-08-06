package colors

import "github.com/charmbracelet/lipgloss"

type ColorPalette struct {
	Accent lipgloss.Color
	Muted  lipgloss.Color
	Text   lipgloss.Color
}

func DefaultColorDarkPalette() ColorPalette {
	return ColorPalette{
		Accent: lipgloss.Color("9"),
		Muted:  lipgloss.Color("8"),
		Text:   lipgloss.Color("15"),
	}
}

func DefaultLightColorPalette() ColorPalette {
	return ColorPalette{
		Accent: lipgloss.Color("9"),
		Muted:  lipgloss.Color("8"),
		Text:   lipgloss.Color("0"),
	}
}
