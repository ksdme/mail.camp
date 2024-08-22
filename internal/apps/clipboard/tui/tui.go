package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ksdme/mail/internal/apps/clipboard/models"
	"github.com/ksdme/mail/internal/core/tui/colors"
	"github.com/ksdme/mail/internal/core/tui/components/help"
	"github.com/ksdme/mail/internal/utils"
	"github.com/muesli/reflow/wordwrap"
	"github.com/uptrace/bun"
)

// The clipboard tui.
// At the moment, it only displays the current contents on the
// clipboard and/or the instructions on how to use it.
type Model struct {
	db *bun.DB

	item *models.ClipboardItem

	width  int
	height int

	renderer *lipgloss.Renderer
	palette  colors.ColorPalette
	keymap   KeyMap

	quitting bool
}

func NewModel(db *bun.DB, renderer *lipgloss.Renderer, palette colors.ColorPalette) Model {
	return Model{
		db: db,

		item: &models.ClipboardItem{
			Value:     []byte("Lorem ipsum dolor sit amet, consectetur adipiscing elit"),
			CreatedAt: time.Now(),
		},

		renderer: renderer,
		palette:  palette,
		keymap:   DefaultKeyMap(),
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keymap.Clear):
			return m, nil
		}

	case tea.QuitMsg:
		m.quitting = true
		return m, nil
	}

	return m, nil
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	xp, yp := 6, 1
	width := m.width - 2*xp
	height := m.height - 2*yp

	// Help
	help := help.View(
		[]key.Binding{m.keymap.Clear, m.keymap.Quit},
		m.renderer,
		m.palette,
	)
	height -= 2

	// Contents
	var contents string
	if m.item == nil {
		contents = m.renderer.
			NewStyle().
			Height(height).
			AlignVertical(lipgloss.Center).
			Render(m.empty(width))
	} else {
		contents = m.renderer.
			NewStyle().
			Height(height).
			Render(m.value(width))
	}

	return m.renderer.
		NewStyle().
		Padding(yp, xp).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Top,
				contents,
				lipgloss.
					NewStyle().
					PaddingTop(1).
					Render(help),
			),
		)
}

// Render the clipboard item.
func (m Model) value(width int) string {
	title := "Your Clipboard"
	title = m.renderer.
		NewStyle().
		PaddingBottom(1).
		Foreground(m.palette.Muted).
		Width(width).
		AlignHorizontal(lipgloss.Center).
		Render(title)

	age := utils.RoundedAge(time.Since(m.item.CreatedAt))
	label := m.renderer.
		NewStyle().
		PaddingTop(1).
		Foreground(m.palette.Muted).
		Render(fmt.Sprintf("Updated %s ago", age))

	cells := m.item.Value
	length := m.width * m.height * 7 / 10
	if len(m.item.Value) > length && length >= 12 {
		cells = append(cells[:length-12], []byte(" (truncated)")...)
	}

	value := wordwrap.String(string(cells), width)
	value = m.renderer.
		NewStyle().
		PaddingTop(1).
		Foreground(m.palette.Text).
		Render(value)

	return lipgloss.JoinVertical(lipgloss.Top, title, label, value)
}

// Render the empty clipboard message.
func (m Model) empty(width int) string {
	msg := "Your clipboard is empty"
	msg = m.renderer.
		NewStyle().
		PaddingBottom(1).
		PaddingLeft((width - len(msg)) / 2).
		Foreground(m.palette.Muted).
		Render(msg)

	descriptionStyle := m.renderer.
		NewStyle().
		PaddingTop(1).
		Foreground(m.palette.Muted)

	commandStyle := m.renderer.
		NewStyle().
		Foreground(m.palette.Text)

	tip := lipgloss.JoinVertical(
		lipgloss.Top,

		descriptionStyle.Render("To put text on the clipboard"),
		commandStyle.Render("echo \"Hello World\" | ssh ssh.camp clipboard put"),

		descriptionStyle.Render("And, to read the clipboard non-interactively"),
		commandStyle.Render("ssh ssh.camp clipboard get"),

		descriptionStyle.Render("You can also pipe it to your system clipboard"),
		commandStyle.Render("ssh ssh.camp clipboard get | xsel -ib"),
	)
	tip = lipgloss.
		NewStyle().
		PaddingLeft((width - lipgloss.Width(tip)) / 2).
		Render(tip)

	return lipgloss.JoinVertical(lipgloss.Top, msg, tip)
}

type KeyMap struct {
	Quit  key.Binding
	Clear key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c", "q"),
			key.WithHelp("q", "quit"),
		),
		Clear: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "clear clipboard"),
		),
	}
}
