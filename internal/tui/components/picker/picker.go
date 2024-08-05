package picker

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// An item in the picker list.
type Item struct {
	ID    int
	Label string
	Badge string
	Value any
}

// A picker is a component that can be used to show a list of
// values and have the user pick from them.
type Model struct {
	title string
	items []Item

	selected    int
	highlighted int
	focused     bool

	Width  int
	Height int
	Styles Styles
	KeyMap KeyMap
}

func NewModel(title string, items []Item, width int, height int) Model {
	return Model{
		title: title,
		items: items,

		focused:     false,
		selected:    0,
		highlighted: 0,

		Width:  width,
		Height: height,
		Styles: DefaultStyles(),
		KeyMap: DefaultKeyMap(),
	}
}

// Handle the focused-ness of the component.
func (m *Model) Focus() {
	m.focused = true
}

func (m *Model) Blur() {
	m.focused = false
}

func (m Model) IsFocused() bool {
	return m.focused
}

func (m *Model) SetItems(items []Item) {
	newSelected := 0
	if item, err := m.SelectedItem(); err == nil {
		for index, element := range items {
			if element.ID == item.ID {
				newSelected = index
				break
			}
		}
	}

	m.items = items
	m.selected = newSelected
	m.highlighted = newSelected
}

func (m Model) HasItems() bool {
	return len(m.items) > 0
}

func (m Model) SelectedItem() (Item, error) {
	if m.selected < 0 || m.selected >= len(m.items) {
		return Item{}, fmt.Errorf("invalid selection")
	}
	return m.items[m.selected], nil
}

func (m Model) HighlightedItem() (Item, error) {
	if m.highlighted < 0 || m.highlighted >= len(m.items) {
		return Item{}, fmt.Errorf("invalid highlight")
	}
	return m.items[m.highlighted], nil
}

// Mark and return the current highlighted item as the selected item.
func (m *Model) Select() (Item, error) {
	m.selected = m.highlighted
	return m.SelectedItem()
}

func (m Model) clampedIndex(index int) int {
	if index < 0 {
		return 0
	}

	if index >= len(m.items) {
		return len(m.items) - 1
	}

	return index
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.IsFocused() && m.HasItems() {
			switch {
			case key.Matches(msg, m.KeyMap.GoToTop):
				m.highlighted = 0

			case key.Matches(msg, m.KeyMap.GoToLast):
				m.highlighted = len(m.items) - 1

			case key.Matches(msg, m.KeyMap.Up):
				m.highlighted = m.clampedIndex(m.highlighted - 1)

			case key.Matches(msg, m.KeyMap.Down):
				m.highlighted = m.clampedIndex(m.highlighted + 1)
			}
		}
	}

	return m, nil
}

func (m Model) View() string {
	lines := []string{}

	// Add a title to the widget.
	height := m.Height
	if m.title != "" {
		title := m.Styles.Title.Render(m.title)
		lines = append(lines, title)
		height -= lipgloss.Height(title)
	}

	// Keep the highlighted line visible.
	var items []Item
	if m.highlighted > height-1 {
		items = m.items[m.highlighted-height+1 : m.highlighted+1]
	} else {
		if len(m.items) > height {
			items = m.items[:height]
		} else {
			items = m.items
		}
	}

	// Render lines.
	for index, item := range items {
		legend := " "
		if index == m.selected {
			legend = m.Styles.SelectedLegend.Render("┃")
		}

		// If the text overflows, trim it, add ellipsis.
		label := item.Label
		total := len(label) + len(item.Badge) + 2
		if total > m.Width {
			// TODO: Trim from badges too.
			label = label[:len(label)-(total-m.Width+3)] + "…"
		}

		if index == m.highlighted && m.IsFocused() {
			label = m.Styles.Highlighted.Render(label)
		} else {
			label = m.Styles.Regular.Render(label)
		}

		line := lipgloss.JoinHorizontal(
			lipgloss.Left,
			legend,
			lipgloss.NewStyle().Width(1).Render(),
			label,
		)

		if len(item.Badge) != 0 {
			badge := m.Styles.Badge.Render(item.Badge)
			line = lipgloss.JoinHorizontal(
				lipgloss.Bottom,
				line,
				lipgloss.
					NewStyle().
					Width(m.Width-lipgloss.Width(line)-lipgloss.Width(badge)).
					Render(),
				badge,
			)
		}

		lines = append(lines, line)
	}

	return lipgloss.
		NewStyle().
		Width(m.Width).
		Render(lipgloss.JoinVertical(lipgloss.Top, lines...))
}

type Styles struct {
	Title          lipgloss.Style
	Badge          lipgloss.Style
	Regular        lipgloss.Style
	SelectedLegend lipgloss.Style
	Highlighted    lipgloss.Style
}

func DefaultStyles() Styles {
	return Styles{
		Title:          lipgloss.NewStyle().PaddingLeft(2).Height(2),
		Badge:          lipgloss.NewStyle(),
		Regular:        lipgloss.NewStyle(),
		SelectedLegend: lipgloss.NewStyle().Bold(true),
		Highlighted:    lipgloss.NewStyle(),
	}
}

type KeyMap struct {
	GoToTop  key.Binding
	GoToLast key.Binding

	Up   key.Binding
	Down key.Binding
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down},
		{k.GoToTop, k.GoToLast},
	}
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		GoToTop:  key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "first")),
		GoToLast: key.NewBinding(key.WithKeys("G"), key.WithHelp("G", "last")),

		Up:   key.NewBinding(key.WithKeys("k", "up", "ctrl+p"), key.WithHelp("k", "up")),
		Down: key.NewBinding(key.WithKeys("j", "down", "ctrl+n"), key.WithHelp("j", "down")),
	}
}
