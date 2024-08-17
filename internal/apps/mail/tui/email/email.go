package email

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ksdme/mail/internal/apps/mail/models"
	"github.com/ksdme/mail/internal/core/tui/colors"
	"github.com/ksdme/mail/internal/utils"
)

type MailSelectedMsg struct {
	To   string
	Mail models.Mail
}

type MailDismissMsg struct{}

type Model struct {
	viewport viewport.Model

	Width  int
	Height int

	KeyMap   KeyMap
	Renderer *lipgloss.Renderer
	Colors   colors.ColorPalette
}

func NewModel(renderer *lipgloss.Renderer, colors colors.ColorPalette) Model {
	width := 64
	height := 64

	return Model{
		viewport: viewport.New(width, height),

		Width:  width,
		Height: height,

		KeyMap:   DefaultKeyMap(),
		Renderer: renderer,
		Colors:   colors,
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
		switch {
		case key.Matches(msg, m.KeyMap.Dismiss):
			return m, m.dismiss
		}

	case MailSelectedMsg:
		m.viewport.SetContent(m.makeContent(msg.To, msg.Mail))
		m.viewport.SetYOffset(0)
		return m, nil
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	return m.viewport.View()
}

func (m Model) makeContent(toAddress string, mail models.Mail) string {
	labelStyle := m.Renderer.
		NewStyle().
		Foreground(m.Colors.Muted).
		PaddingRight(1)

	valueStyle := m.Renderer.
		NewStyle().
		Foreground(m.Colors.Text)

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
		valueStyle.Render(toAddress),
	)

	subject := lipgloss.JoinHorizontal(
		lipgloss.Left,
		labelStyle.Render("Subject"),
		valueStyle.Render(utils.Decode(mail.Subject)),
	)

	created := lipgloss.JoinHorizontal(
		lipgloss.Left,
		labelStyle.Render("Received"),
		valueStyle.Render(mail.CreatedAt.Format(time.RFC822)),
	)

	text := valueStyle.
		MarginTop(1).
		Render(utils.Decode(mail.Text))

	return lipgloss.JoinVertical(
		lipgloss.Top,
		to,
		from,
		subject,
		created,
		text,
	)
}

func (m Model) dismiss() tea.Msg {
	return MailDismissMsg{}
}

type KeyMap struct {
	Dismiss key.Binding
}

func (m Model) Help() []key.Binding {
	return []key.Binding{
		m.KeyMap.Dismiss,
	}
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Dismiss: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "go back"),
		),
	}
}
