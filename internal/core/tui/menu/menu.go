package menu

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ksdme/mail/internal/config"
	"github.com/ksdme/mail/internal/core"
	"github.com/ksdme/mail/internal/core/tui/colors"
	"github.com/ksdme/mail/internal/core/tui/components/help"
)

// Represents a menu to select between an application.
type Model struct {
	list list.Model

	keymap KeyMap

	palette  colors.ColorPalette
	renderer *lipgloss.Renderer
}

func NewModel(
	apps []core.App,
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
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width-8, msg.Height-7)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) View() string {
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
