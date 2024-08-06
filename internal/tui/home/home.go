package home

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ksdme/mail/internal/models"
	"github.com/ksdme/mail/internal/tui/colors"
	"github.com/ksdme/mail/internal/tui/components/picker"
	"github.com/ksdme/mail/internal/tui/components/table"
	"github.com/ksdme/mail/internal/tui/email"
	"github.com/ksdme/mail/internal/utils"
)

type MailboxesRefreshedMsg struct {
	Mailboxes []models.Mailbox
	Err       error
}

type MailboxSelectedMsg struct {
	Mailbox models.Mailbox
}

type MailsRefreshedMsg struct {
	Mailbox models.Mailbox
	Mails   []models.Mail
	Err     error
}

type CreateRandomMailboxMsg struct{}

type DeleteMailboxMsg struct {
	Mailbox models.Mailbox
}

type Model struct {
	mailboxes picker.Model
	mails     table.Model

	Width  int
	Height int

	KeyMap   KeyMap
	Renderer *lipgloss.Renderer
	Colors   colors.ColorPalette

	SelectedMailbox models.Mailbox
}

func NewModel(renderer *lipgloss.Renderer, colors colors.ColorPalette) Model {
	width := 80
	height := 80

	// Setup the mails picker.
	pStyles := picker.DefaultStyles(renderer)
	pStyles.Title = pStyles.Title.Foreground(colors.Muted)
	pStyles.Badge = pStyles.Badge.Foreground(colors.Muted)
	pStyles.Regular = pStyles.Regular.Foreground(colors.Text)
	pStyles.Highlighted = pStyles.Highlighted.Foreground(colors.Accent).Bold(true)
	pStyles.SelectedLegend = pStyles.SelectedLegend.Foreground(colors.Accent)
	mailboxes := picker.NewModel("Mailboxes", []picker.Item{}, width/3, height)
	mailboxes.Styles = pStyles
	mailboxes.Focus()

	// Setup the mails table.
	tStyles := table.DefaultStyles(renderer)
	tStyles.Header = renderer.NewStyle().Height(2).Foreground(colors.Muted)
	tStyles.Selected = tStyles.Selected.Foreground(colors.Accent).Bold(true)
	table := table.New(
		table.WithColumns(makeMailTableColumns(width*2/3)),
		table.WithHeight(height),
		table.WithRows([]table.Row{}),
		table.WithStyles(tStyles),
	)

	return Model{
		mailboxes: mailboxes,
		mails:     table,

		Width:  width,
		Height: height,

		KeyMap:   DefaultKeyMap(),
		Renderer: renderer,
		Colors:   colors,
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

		case key.Matches(msg, m.KeyMap.CreateRandomMailbox):
			return m, m.createRandomMailbox

		case key.Matches(msg, m.KeyMap.DeleteMailbox):
			if item, err := m.mailboxes.HighlightedItem(); err == nil {
				mailbox := item.Value.(models.Mailbox)
				return m, m.deleteMailbox(mailbox)
			}
		}

	case MailboxesRefreshedMsg:
		// TODO: Handle error.
		var items []picker.Item
		for _, mailbox := range msg.Mailboxes {
			items = append(items, picker.Item{
				ID:    int(mailbox.ID),
				Label: mailbox.Email(),
				Value: mailbox,
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

	case MailsRefreshedMsg:
		// TODO: Handle error.
		if msg.Mailbox.ID == m.SelectedMailbox.ID {
			var items []table.Row
			for _, mail := range msg.Mails {
				age := fmt.Sprintf(
					"%s ago",
					utils.RoundedAge(time.Since(mail.CreatedAt)),
				)

				items = append(items, table.Row{
					ID:    int(mail.ID),
					Value: mail,
					Cols:  []string{mail.Subject, mail.FromAddress, age},
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
		return m.Renderer.
			NewStyle().
			Width(m.Width).
			Height(m.Height).
			AlignHorizontal(lipgloss.Center).
			AlignVertical(lipgloss.Center).
			Foreground(m.Colors.Muted).
			Render("no mailboxes :(")
	}

	mailboxes := m.Renderer.
		NewStyle().
		PaddingRight(6).
		Foreground(m.Colors.Text).
		Render(m.mailboxes.View())

	var mails string
	if !m.mails.HasRows() {
		mails = lipgloss.JoinVertical(
			lipgloss.Top,
			m.mailboxes.
				Styles.
				Title.
				PaddingLeft(0).
				Render("Mails"),
			m.Renderer.
				NewStyle().
				Width(m.mails.Width()).
				Height(m.mails.Height()).
				Foreground(m.Colors.Text).
				Render(fmt.Sprintf(
					"no mails in %s, incoming mails are only stored for 48h",
					m.SelectedMailbox.Email(),
				)),
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

func (m Model) deleteMailbox(mailbox models.Mailbox) tea.Cmd {
	return func() tea.Msg {
		return DeleteMailboxMsg{mailbox}
	}
}

func (m Model) mailSelected(mailbox models.Mailbox, mail models.Mail) tea.Cmd {
	return func() tea.Msg {
		return email.MailSelectedMsg{Mailbox: mailbox, Mail: mail}
	}
}

func (m Model) createRandomMailbox() tea.Msg {
	return CreateRandomMailboxMsg{}
}

func (m Model) Help() []key.Binding {
	var help []key.Binding

	if m.mailboxes.IsFocused() {
		help = append(
			help,
			m.KeyMap.CreateRandomMailbox,
			m.KeyMap.DeleteMailbox,
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
	CreateRandomMailbox key.Binding
	DeleteMailbox       key.Binding

	Select key.Binding

	FocusMailboxes key.Binding
	FocusMails     key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		CreateRandomMailbox: key.NewBinding(
			key.WithKeys("ctrl+n"),
			key.WithHelp("ctrl+n", "generate mailbox"),
		),
		DeleteMailbox: key.NewBinding(
			key.WithKeys("ctrl+k"),
			key.WithHelp("ctrl+k", "delete mailbox"),
		),

		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),

		FocusMailboxes: key.NewBinding(
			key.WithKeys("left", "h", "esc"),
			key.WithHelp("←/h", "mailboxes"),
		),
		FocusMails: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "mails"),
		),
	}
}

func makeMailTableColumns(width int) []table.Column {
	at := width * 1 / 10
	from := (width * 3) / 10
	return []table.Column{
		{Title: "Subject", Width: width - at - from},
		{Title: "From", Width: from},
		{Title: "At", Width: at},
	}
}
