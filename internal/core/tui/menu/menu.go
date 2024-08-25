package menu

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	"github.com/ksdme/mail/internal/config"
	"github.com/ksdme/mail/internal/core"
	"github.com/ksdme/mail/internal/core/models"
	"github.com/ksdme/mail/internal/core/tui/colors"
	"github.com/ksdme/mail/internal/core/tui/components/help"
)

type BackToMenuMsg struct{}

// Represents a menu to select between an application.
type Model struct {
	session ssh.Session
	account models.Account

	list list.Model

	model   tea.Model // The selected app model.
	cleanup func()    // The clean up method to run when offloading app.

	width  int
	height int

	keymap   KeyMap
	palette  colors.ColorPalette
	renderer *lipgloss.Renderer

	quitting bool
}

func NewModel(
	apps []core.App,

	session ssh.Session,
	account models.Account,

	renderer *lipgloss.Renderer,
	palette colors.ColorPalette,
) Model {
	// Make the list take colors from our palette.
	delegate := list.NewDefaultDelegate()
	delegate.Styles.DimmedTitle = delegate.Styles.DimmedTitle.
		Foreground(palette.Muted)
	delegate.Styles.DimmedDesc = delegate.Styles.DimmedDesc.
		Foreground(palette.Muted)
	delegate.Styles.NormalTitle = delegate.Styles.NormalTitle.
		Foreground(palette.Text)
	delegate.Styles.NormalDesc = delegate.Styles.NormalDesc.
		Foreground(palette.Muted)
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Border(lipgloss.ThickBorder(), false, false, false, true).
		BorderForeground(palette.Accent).
		Foreground(palette.Accent)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Border(lipgloss.ThickBorder(), false, false, false, true).
		BorderForeground(palette.Accent).
		Foreground(palette.Muted)

	items := []list.Item{}
	for _, app := range apps {
		items = append(items, item{app})
	}
	list := list.New(items, delegate, 0, 0)

	// Make the list look minimal.
	list.SetShowTitle(false)
	list.SetShowStatusBar(false)
	list.SetShowFilter(false)
	list.SetShowPagination(false)
	list.SetShowHelp(false)

	return Model{
		session: session,
		account: account,

		list:   list,
		keymap: DefaultKeyMap(),

		renderer: renderer,
		palette:  palette,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.Quit):
			if m.model == nil {
				if m.cleanup != nil {
					m.cleanup()
				}

				m.quitting = true
				return m, tea.Quit
			}

		case key.Matches(msg, m.keymap.Select):
			if m.model == nil {
				item, _ := m.list.SelectedItem().(item)

				if m.cleanup != nil {
					m.cleanup()
				}

				m.model, m.cleanup = item.app.HandleApp(
					m.session,
					m.account,
					m.renderer,
					m.palette,
					func() tea.Msg {
						return BackToMenuMsg{}
					},
				)

				return m, tea.Batch(
					m.model.Init(),
					func() tea.Msg {
						return tea.WindowSizeMsg{
							Width:  m.width,
							Height: m.height,
						}
					},
				)
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		m.list.SetSize(msg.Width-8, msg.Height-7)
		if m.model != nil {
			m.model, _ = m.model.Update(msg)
		}

	case BackToMenuMsg:
		m.model = nil
		if m.cleanup != nil {
			m.cleanup()
		}

		return m, nil
	}

	if m.model == nil {
		m.list, cmd = m.list.Update(msg)
	} else {
		m.model, cmd = m.model.Update(msg)
	}
	return m, cmd
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	if m.model != nil {
		return m.model.View()
	}

	// Render the menu.
	width := m.list.Width()

	title := m.renderer.
		NewStyle().
		Width(width).
		AlignHorizontal(lipgloss.Center).
		Foreground(m.palette.Muted).
		PaddingBottom(2).
		Render("ssh.camp")

	help := help.View(
		[]key.Binding{m.keymap.Select, m.keymap.Quit},
		m.renderer,
		m.palette,
	)

	footer := help
	gap := width - lipgloss.Width(help) - len(config.Core.Signature)
	if len(config.Core.Signature) > 0 && gap > 8 {
		footer = lipgloss.JoinHorizontal(
			lipgloss.Left,
			help,
			m.renderer.
				NewStyle().
				AlignHorizontal(lipgloss.Right).
				PaddingLeft(gap).
				Foreground(m.palette.Muted).
				Render(config.Core.Signature),
		)
	}

	contents := lipgloss.JoinVertical(
		lipgloss.Top,
		title,
		m.list.View(),
		lipgloss.
			NewStyle().
			PaddingTop(1).
			Render(footer),
	)

	return lipgloss.
		NewStyle().
		Padding(1, 4).
		Render(contents)
}

type KeyMap struct {
	Select key.Binding
	Quit   key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

type item struct {
	app core.App
}

func (i item) Title() string {
	_, title, _ := i.app.Info()
	return title
}

func (i item) Description() string {
	_, _, description := i.app.Info()
	return description
}

func (i item) FilterValue() string {
	_, title, _ := i.app.Info()
	return title
}
