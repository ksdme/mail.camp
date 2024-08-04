package home

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ksdme/mail/internal/models"
	"github.com/ksdme/mail/internal/tui/components/picker"
	"github.com/ksdme/mail/internal/tui/components/table"
	"github.com/ksdme/mail/internal/utils"
)

type MailboxesUpdateMsg struct {
	Mailboxes []models.Mailbox
	Err       error
}

type Model struct {
	mailboxes picker.Model
	mails     table.Model

	Width  int
	Height int
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
	styles.Header = mailboxes.Styles.Title
	table := table.New(
		table.WithColumns(makeMailTableColumns(initialWidth*2/3)),
		table.WithHeight(initialHeight),
		table.WithRows([]table.Row{}),
		table.WithStyles(styles),
		table.WithStyleFunc(func(row int) lipgloss.Style {
			if row > 6 {
				return mailboxes.Styles.SelectedLabel
			}
			return styles.Cell
		}),
	)

	return Model{
		mailboxes: mailboxes,
		mails:     table,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		gap := 3

		m.mailboxes.Width = m.Width / 3
		m.mailboxes.Height = m.Height

		m.mails.SetWidth(m.Width - m.mailboxes.Width - gap)
		m.mails.SetHeight(m.Height)
		m.mails.SetColumns(makeMailTableColumns(m.mails.Width()))

	case tea.KeyMsg:
		switch msg.String() {
		case "right", "l":
			m.mailboxes.Blur()
			m.mails.Focus()

		case "left", "h":
			m.mailboxes.Focus()
			m.mails.Blur()
		}

	case MailboxesUpdateMsg:
		// TODO: Handle error.
		var items []picker.Item
		for _, mailbox := range msg.Mailboxes {
			items = append(items, picker.Item{
				Label: mailbox.Email(),
				Value: int(mailbox.ID),
			})
		}
		m.mailboxes.SetItems(items)
		// TODO: Raise a load mails command
		return m, nil
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
		PaddingRight(3).
		Render(m.mailboxes.View())

	var mails string
	if len(mails) == 0 {
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

type KeyMap struct {
	FocusMailboxes key.Binding
	FocusMails     key.Binding
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
