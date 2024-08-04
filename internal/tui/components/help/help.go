package help

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
)

func View(bindings []key.Binding) string {
	keyStyle := lipgloss.
		NewStyle().
		PaddingRight(1)

	descStyle := lipgloss.
		NewStyle().
		PaddingRight(3).
		Foreground(lipgloss.AdaptiveColor{
			Light: "#909090",
			Dark:  "#626262",
		})

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
