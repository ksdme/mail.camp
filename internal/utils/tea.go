package utils

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/muesli/termenv"
)

// Run a bubble tea program on the session.
func RunTeaInSession(next ssh.Handler, session ssh.Session, model tea.Model) {
	middleware := bubbletea.MiddlewareWithColorProfile(func(s ssh.Session) (tea.Model, []tea.ProgramOption) {
		options := []tea.ProgramOption{tea.WithAltScreen()}
		return model, options
	}, termenv.ANSI)

	middleware(next)(session)
}
