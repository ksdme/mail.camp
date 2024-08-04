package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ksdme/mail/internal/tui/home"
)

type mode int

const (
	Home mode = iota
)

// Represents the top most model.
type Model struct {
	mode mode
	home home.Model

	width  int
	height int

	quitting bool
}

func NewModel() Model {
	return Model{
		mode: Home,
		home: home.NewModel(),
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		m.home.Width = m.width
		m.home.Height = m.height - 5

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		}
	}

	if m.mode == Home {
		var cmd tea.Cmd
		m.home, cmd = m.home.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) View() string {
	// This lets us not leave behind lines at the end.
	if m.quitting {
		return ""
	}

	view := "loading"
	if m.mode == Home {
		view = m.home.View()
	}

	// TODO: This should actually be centered.
	decoration := lipgloss.
		NewStyle().
		Width(m.width).
		Align(lipgloss.Center).
		PaddingBottom(1).
		Foreground(lipgloss.Color("244")).
		Render("mail.ssh.camp")

	return lipgloss.
		NewStyle().
		Padding(2, 4).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Top,
				decoration,
				view,
			),
		)
}
