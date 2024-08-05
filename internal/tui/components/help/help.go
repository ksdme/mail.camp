package help

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
	"github.com/ksdme/mail/internal/tui/colors"
)

func View(bindings []key.Binding, colors colors.ColorPalette) string {
	keyStyle := lipgloss.
		NewStyle().
		Foreground(colors.Text).
		PaddingRight(1)

	descStyle := lipgloss.
		NewStyle().
		Foreground(colors.Muted).
		PaddingRight(3)

	items := []string{}
	for _, binding := range bindings {
		help := binding.Help()
		if len(help.Desc) == 0 || len(help.Key) == 0 {
			continue
		}

		items = append(
			items,
			lipgloss.JoinHorizontal(
				lipgloss.Left,
				keyStyle.Render(help.Key),
				descStyle.Render(help.Desc),
			),
		)
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, items...)
}
