package tui

import (
	"context"
	"log/slog"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ksdme/mail/internal/models"
	"github.com/ksdme/mail/internal/tui/home"
	"github.com/uptrace/bun"
)

type mode int

const (
	Home mode = iota
)

// Represents the top most model.
type Model struct {
	db      *bun.DB
	account models.Account

	mode mode
	home home.Model

	width  int
	height int

	quitting bool
}

func NewModel(db *bun.DB, account models.Account) Model {
	return Model{
		db:      db,
		account: account,

		mode: Home,
		home: home.NewModel(),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.home.Init(),
		m.loadMailboxes,
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		m.home.Width = m.width - 12
		m.home.Height = m.height - 6

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
		PaddingBottom(2).
		Foreground(lipgloss.Color("244")).
		Render("mail.ssh.camp")

	return lipgloss.
		NewStyle().
		Padding(2, 6).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Top,
				decoration,
				view,
			),
		)
}

func (m Model) loadMailboxes() tea.Msg {
	var mailboxes []models.Mailbox

	// TODO: The context should be bound to the ssh connection.
	err := m.db.NewSelect().
		Model(&mailboxes).
		Where("account_id = ?", m.account.ID).
		Scan(context.Background())
	slog.Debug("mailboxes", "count", len(mailboxes))

	return home.MailboxesUpdateMsg{Mailboxes: mailboxes, Err: err}
}
