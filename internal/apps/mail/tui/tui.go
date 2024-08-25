package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ksdme/mail/internal/apps/mail/tui/email"
	"github.com/ksdme/mail/internal/apps/mail/tui/home"
	core "github.com/ksdme/mail/internal/core/models"
	"github.com/ksdme/mail/internal/core/tui/colors"
	"github.com/ksdme/mail/internal/core/tui/components/help"
	"github.com/uptrace/bun"
)

type mode int

const (
	Home mode = iota
	Email
)

// Represents the top most model.
type Model struct {
	db      *bun.DB
	account core.Account

	mode  mode
	home  home.Model
	email email.Model

	width  int
	height int

	KeyMap   KeyMap
	Colors   colors.ColorPalette
	Renderer *lipgloss.Renderer

	quit     tea.Cmd
	quitting bool
}

func NewModel(
	db *bun.DB,
	account core.Account,
	renderer *lipgloss.Renderer,
	colors colors.ColorPalette,
	quit tea.Cmd,
) Model {
	return Model{
		db:      db,
		account: account,

		mode:  Home,
		home:  home.NewModel(db, renderer, colors),
		email: email.NewModel(renderer, colors),

		KeyMap:   DefaultKeyMap(),
		Renderer: renderer,
		Colors:   colors,

		quit: quit,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.home.Init(),
		m.email.Init(),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		m.home.Width = m.width - 12
		m.home.Height = m.height - 5

		m.email.Width = m.home.Width
		m.email.Height = m.home.Height

		m.home, _ = m.home.Update(msg)
		m.email, _ = m.email.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.KeyMap.Quit):
			m.quitting = true
			return m, m.quit
		}

	case home.MailboxRealTimeUpdate:
		m.home, cmd = m.home.Update(msg)
		return m, cmd

	case email.MailSelectedMsg:
		m.mode = Email
		m.email, cmd = m.email.Update(msg)
		return m, cmd

	case email.MailDismissMsg:
		m.mode = Home
		return m, nil
	}

	if m.mode == Home {
		m.home, cmd = m.home.Update(msg)
		return m, cmd
	} else if m.mode == Email {
		m.email, cmd = m.email.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) View() string {
	// This lets us not leave behind lines at the end.
	if m.quitting {
		return ""
	}

	content := "loading"
	if m.mode == Home {
		content = m.home.View()
	} else if m.mode == Email {
		content = m.email.View()
	}

	return m.Renderer.
		NewStyle().
		Padding(2, 6).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Top,
				content,
				help.View(m.Help(), m.Renderer, m.Colors),
			),
		)
}

func (m Model) Help() []key.Binding {
	var bindings []key.Binding

	if m.mode == Home {
		bindings = append(bindings, m.home.Help()...)
	} else if m.mode == Email {
		bindings = append(bindings, m.email.Help()...)
	}

	return append(bindings, m.KeyMap.Quit)
}

type KeyMap struct {
	Quit key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c", "q"),
			key.WithHelp("q", "quit"),
		),
	}
}
