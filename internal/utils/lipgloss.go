package utils

import "github.com/charmbracelet/lipgloss"

func Box(width, height int, xCentered bool, yCentered bool) lipgloss.Style {
	style := lipgloss.NewStyle().Width(width).Height(height)

	if xCentered {
		style = style.AlignHorizontal(lipgloss.Center)
	}
	if yCentered {
		style = style.AlignVertical(lipgloss.Center)
	}

	return style
}
