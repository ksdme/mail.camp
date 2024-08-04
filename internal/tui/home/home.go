package home

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ksdme/mail/internal/models"
	"github.com/ksdme/mail/internal/tui/components/picker"
	"github.com/ksdme/mail/internal/tui/components/table"
	"github.com/ksdme/mail/internal/tui/email"
	"github.com/ksdme/mail/internal/utils"
)

type MailboxesUpdateMsg struct {
	Mailboxes []models.Mailbox
	Err       error
}

type MailsUpdateMsg struct {
	Mailbox models.Mailbox
	Mails   []models.Mail
	Err     error
}

type MailboxSelectedMsg struct {
	Mailbox models.Mailbox
}

type Model struct {
	mailboxes picker.Model
	mails     table.Model

	Width  int
	Height int
	KeyMap KeyMap

	SelectedMailbox models.Mailbox
}

func NewModel() Model {
	initialWidth := 80
	initialHeight := 80

	// Setup the mails picker.
	mailboxes := picker.NewModel(
		"Mailboxes",
		[]picker.Item{},
		initialWidth/3,
		initialHeight,
	)
	mailboxes.Focus()

	// Setup the mails table.
	styles := table.DefaultStyles()
	styles.Header = mailboxes.Styles.Title.PaddingLeft(1)
	table := table.New(
		table.WithColumns(makeMailTableColumns(initialWidth*2/3)),
		table.WithHeight(initialHeight),
		table.WithRows([]table.Row{}),
		table.WithStyles(styles),
		table.WithStyleFunc(func(row int) lipgloss.Style {
			if row > 6 {
				return mailboxes.Styles.SelectedLegend
			}
			return styles.Cell
		}),
	)

	return Model{
		mailboxes: mailboxes,
		mails:     table,

		Width:  initialWidth,
		Height: initialHeight,
		KeyMap: DefaultKeyMap(),
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		gap := 6

		m.mailboxes.Width = m.Width / 3
		m.mailboxes.Height = m.Height

		m.mails.SetWidth(m.Width - m.mailboxes.Width - gap)
		m.mails.SetHeight(m.Height)
		m.mails.SetColumns(makeMailTableColumns(m.mails.Width()))

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.KeyMap.FocusMailboxes):
			m.mailboxes.Focus()
			m.mails.Blur()

		case key.Matches(msg, m.KeyMap.FocusMails):
			if m.mails.HasRows() {
				m.mailboxes.Blur()
				m.mails.Focus()
			}

		case key.Matches(msg, m.KeyMap.Select):
			if m.mailboxes.IsFocused() {
				if item, err := m.mailboxes.Select(); err == nil {
					m.SelectedMailbox = item.Value.(models.Mailbox)

					m.mails.SetRows([]table.Row{})
					m.mailboxes.Blur()
					m.mails.Focus()

					return m, m.mailboxSelected
				}
			} else if m.mails.Focused() {
				if row, err := m.mails.SelectedRow(); err == nil {
					mail := row.Value.(models.Mail)
					return m, m.mailSelected(m.SelectedMailbox, mail)
				}
			}
		}

	case MailboxesUpdateMsg:
		// TODO: Handle error.
		var items []picker.Item
		for _, mailbox := range msg.Mailboxes {
			items = append(items, picker.Item{
				Label: mailbox.Email(),
				Value: mailbox,
				Badge: "2",
			})
		}
		m.mailboxes.SetItems(items)

		// Trigger mails load.
		m.mails.SetRows([]table.Row{})
		if m.mailboxes.HasItems() {
			if item, err := m.mailboxes.SelectedItem(); err == nil {
				m.SelectedMailbox = item.Value.(models.Mailbox)
				return m, m.mailboxSelected
			}
		}
		return m, nil

	case MailsUpdateMsg:
		// TODO: Retain selection if the same mailbox is updated.
		// TODO: Handle error.
		if msg.Mailbox.ID == m.SelectedMailbox.ID {
			var items []table.Row
			for _, mail := range msg.Mails {
				items = append(items, table.Row{
					Cols: []string{
						mail.Subject,
						mail.FromAddress,
						"8 mins ago",
					},
					Value: mail,
				})
			}
			m.mails.SetRows(items)

			// If the update caused there to be no mails.
			if !m.mails.HasRows() {
				m.mails.Blur()
				m.mailboxes.Focus()
			}
			return m, nil
		} else {
			return m, nil
		}
	}

	var cmd tea.Cmd
	if m.mailboxes.IsFocused() {
		m.mailboxes, cmd = m.mailboxes.Update(msg)
		return m, cmd
	} else {
		m.mails, cmd = m.mails.Update(msg)
		return m, cmd
	}
}

func (m Model) View() string {
	if !m.mailboxes.HasItems() {
		return utils.
			Box(m.Width, m.Height, true, true).
			Foreground(lipgloss.Color("244")).
			Render("no mailboxes :(")
	}

	mailboxes := lipgloss.NewStyle().
		PaddingRight(6).
		Render(m.mailboxes.View())

	var mails string
	if !m.mails.HasRows() {
		mails = lipgloss.JoinVertical(
			lipgloss.Top,
			m.mailboxes.Styles.Title.PaddingLeft(0).Render("Mails"),
			utils.
				Box(m.mails.Width(), m.mails.Height(), false, false).
				Foreground(lipgloss.Color("244")).
				Render(fmt.Sprintf("no mails in %s, incoming mails are only stored for 48h", "mailbox@localhost")),
		)
	} else {
		mails = m.mails.View()
	}

	return lipgloss.JoinHorizontal(
		lipgloss.Left,
		mailboxes,
		mails,
	)
}

func (m Model) mailboxSelected() tea.Msg {
	return MailboxSelectedMsg{Mailbox: m.SelectedMailbox}
}

func (m Model) mailSelected(mailbox models.Mailbox, mail models.Mail) tea.Cmd {
	return func() tea.Msg {
		return email.MailSelectedMsg{Mailbox: mailbox, Mail: mail}
	}
}

func (m Model) Help() []key.Binding {
	var help []key.Binding

	if m.mailboxes.IsFocused() {
		help = append(
			help,
			m.KeyMap.Select,
			m.KeyMap.FocusMails,
		)
	} else if m.mails.Focused() {
		help = append(
			help,
			m.KeyMap.Select,
			m.KeyMap.FocusMailboxes,
		)
	}

	return help
}

type KeyMap struct {
	FocusMailboxes key.Binding
	FocusMails     key.Binding
	Select         key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		FocusMailboxes: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "focus mailboxes"),
		),
		FocusMails: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "focus mails"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
	}
}

func makeMailTableColumns(width int) []table.Column {
	at := width * 2 / 10
	from := (width * 3) / 10
	return []table.Column{
		{Title: "Subject", Width: width - at - from},
		{Title: "From", Width: from},
		{Title: "At", Width: at},
	}
}
