package tui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ksdme/mail/internal/models"
	"github.com/ksdme/mail/internal/tui/components/help"
	"github.com/ksdme/mail/internal/tui/email"
	"github.com/ksdme/mail/internal/tui/home"
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
	account models.Account

	mode  mode
	home  home.Model
	email email.Model

	width  int
	height int

	KeyMap KeyMap

	quitting bool
}

func NewModel(db *bun.DB, account models.Account) Model {
	return Model{
		db:      db,
		account: account,

		mode:  Home,
		home:  home.NewModel(),
		email: email.NewModel(),

		KeyMap: DefaultKeyMap(),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.home.Init(),
		m.email.Init(),
		m.refreshMailboxes,
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
			return m, tea.Quit
		}

	case home.MailboxesRefreshedMsg:
		m.home, cmd = m.home.Update(msg)
		return m, cmd

	case home.MailboxSelectedMsg:
		return m, m.refreshMails(msg.Mailbox)

	case home.CreateRandomMailboxMsg:
		return m, m.createRandomMailbox

	case email.MailSelectedMsg:
		m.mode = Email
		m.email, cmd = m.email.Update(msg)
		return m, tea.Batch(tea.ClearScreen, cmd)

	case email.MailDismissMsg:
		m.mode = Home
		return m, tea.ClearScreen
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

	view := "loading"
	if m.mode == Home {
		view = m.home.View()
	} else if m.mode == Email {
		view = m.email.View()
	}

	return lipgloss.
		NewStyle().
		Padding(2, 6).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Top,
				view,
				help.View(m.Help()),
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

func (m Model) refreshMailboxes() tea.Msg {
	var mailboxes []models.Mailbox

	// TODO: The context should be bound to the ssh connection.
	err := m.db.NewSelect().
		Model(&mailboxes).
		Where("account_id = ?", m.account.ID).
		Order("id DESC").
		Scan(context.Background())

	return home.MailboxesRefreshedMsg{Mailboxes: mailboxes, Err: err}
}

func (m Model) refreshMails(mailbox models.Mailbox) tea.Cmd {
	return func() tea.Msg {
		var mails []models.Mail

		// TODO: The context should be bound to the ssh connection.
		err := m.db.NewSelect().
			Model(&mails).
			Where("mailbox_id = ?", mailbox.ID).
			Scan(context.Background())

		return home.MailsRefreshedMsg{
			Mailbox: mailbox,
			Mails:   mails,
			Err:     err,
		}
	}
}

func (m Model) createRandomMailbox() tea.Msg {
	// TODO: The context should be bound to the ssh connection.
	_, err := models.CreateRandomMailbox(context.Background(), m.db, m.account)
	if err != nil {
		fmt.Println(err)
		// TODO: Handle this error.
		return nil
	}

	return m.refreshMailboxes()
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
