package main

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ksdme/mail/internal/tui"
)

func main() {
	program := tea.NewProgram(tui.NewModel())
	if _, err := program.Run(); err != nil {
		os.Exit(1)
	}
}
