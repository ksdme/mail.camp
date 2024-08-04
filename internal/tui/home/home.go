package home

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ksdme/mail/internal/tui/components/picker"
	"github.com/ksdme/mail/internal/tui/components/table"
)

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
		[]picker.Item{
			{Label: "kilariteja@mail.ssh.camp", Value: 1, Badge: "2"},
			{Label: "thebigapplething@mail.ssh.camp", Value: 1},
		},
		initialWidth/4,
		initialHeight,
	)
	mailboxes.Focus()

	// Setup the mails table.
	rows := []table.Row{
		{"Verify your LinkedIn", "no-reply@linkedin.com", "2 mins ago"},
		{"Verify your LinkedIn", "no-reply@linkedin.com", "2 mins ago"},
	}

	styles := table.DefaultStyles()
	styles.Header = mailboxes.Styles.Title

	table := table.New(
		table.WithColumns(makeMailTableColumns(initialWidth*3/4)),
		table.WithHeight(initialHeight),
		table.WithRows(rows),
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
		gap := 2

		m.mailboxes.Width = m.Width / 4
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
	mailboxes := lipgloss.NewStyle().
		PaddingRight(2).
		Render(m.mailboxes.View())

	return lipgloss.JoinHorizontal(
		lipgloss.Left,
		mailboxes,
		m.mails.View(),
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
