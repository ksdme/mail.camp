package email

import (
	"fmt"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ksdme/mail/internal/models"
)

type MailSelectedMsg struct {
	Mailbox models.Mailbox
	Mail    models.Mail
}

type MailDismissMsg struct{}

type Model struct {
	viewport viewport.Model

	Width  int
	Height int
}

func NewModel() Model {
	initialWidth := 64
	initialHeight := 64

	return Model{
		viewport: viewport.New(initialWidth, initialHeight),

		Width:  initialWidth,
		Height: initialHeight,
	}
}

func (m Model) Init() tea.Cmd {
	return m.viewport.Init()
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewport.Width = m.Width
		m.viewport.Height = m.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			return m, m.dismiss
		}

	case MailSelectedMsg:
		m.viewport.SetContent(m.makeContent(msg.Mailbox, msg.Mail))
		m.viewport.SetYOffset(0)
		return m, tea.ClearScreen
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	return m.viewport.View()
}

func (m Model) makeContent(mailbox models.Mailbox, mail models.Mail) string {
	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		PaddingRight(1)
	valueStyle := lipgloss.NewStyle()

	from := mail.FromAddress
	if len(mail.FromName) > 0 {
		from = fmt.Sprintf("%s <%s>", mail.FromName, mail.FromAddress)
	}
	from = lipgloss.JoinHorizontal(
		lipgloss.Left,
		labelStyle.Render("From"),
		valueStyle.Render(from),
	)

	to := lipgloss.JoinHorizontal(
		lipgloss.Left,
		labelStyle.Render("To"),
		valueStyle.Render(mailbox.Email()),
	)

	subject := lipgloss.JoinHorizontal(
		lipgloss.Left,
		labelStyle.Render("Subject"),
		valueStyle.Render(mail.Subject),
	)

	text := valueStyle.
		MarginTop(1).
		Render(mail.Text)

	return lipgloss.JoinVertical(
		lipgloss.Top,
		to,
		from,
		subject,
		text,
	)
}

func (m Model) dismiss() tea.Msg {
	return MailDismissMsg{}
}
