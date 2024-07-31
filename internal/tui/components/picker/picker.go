package picker

import (
	"fmt"
	"slices"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// A picker is a component that can be used to show a list of
// values and have the user pick from them.
type Model struct {
	title string
	items []PickerItem

	// Tracks the cursor and selected values.
	selected    int
	highlighted int
	focused     bool

	Width  int
	Height int
	Styles Styles
	KeyMap KeyMap
}

func NewModel(title string, items []PickerItem, selected int, width int, height int) Model {
	return Model{
		title: title,
		items: items,

		focused:  false,
		selected: selected,

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

func (m Model) HasItems() bool {
	return len(m.items) > 0
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.KeyMap.GoToTop):
			if m.HasItems() {
				m.highlighted = m.items[0].Value
			}
		case key.Matches(msg, m.KeyMap.GoToLast):
			if m.HasItems() {
				m.highlighted = m.items[len(m.items)-1].Value
			}
		case key.Matches(msg, m.KeyMap.Up):
			if m.HasItems() {
				index := slices.IndexFunc(m.items, func(item PickerItem) bool {
					return item.Value == m.highlighted
				})
				m.highlighted = m.items[max(index-1, 0)].Value
			}
		case key.Matches(msg, m.KeyMap.Down):
			if m.HasItems() {
				index := slices.IndexFunc(m.items, func(item PickerItem) bool {
					return item.Value == m.highlighted
				})
				m.highlighted = m.items[min(index+1, len(m.items)-1)].Value
			}
		case key.Matches(msg, m.KeyMap.Select):
			// TODO: We should push an command.
			m.selected = m.highlighted
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
	var items []PickerItem
	location := slices.IndexFunc(m.items, func(item PickerItem) bool {
		return item.Value == m.highlighted
	})
	if location > height-1 {
		items = m.items[location-height+1 : location+1]
	} else {
		items = m.items[:height]
	}

	for _, item := range items {
		legend := " "
		if item.Value == m.selected {
			legend = "┃"
		}
		label := fmt.Sprintf("%s %s", legend, item.Label)

		// If the text overflows, trim it, add ellipsis.
		total := len(label) + len(item.Badge)
		if total > m.Width {
			label = label[:m.Width-total-6] + "..."
		}

		line := ""
		if item.Value == m.selected {
			line = m.Styles.Selected.Render(label)
		} else if item.Value == m.highlighted {
			if m.IsFocused() {
				line = m.Styles.Highlighted.Render(label)
			} else {
				line = m.Styles.HighlightedBlur.Render(label)
			}
		} else {
			line = m.Styles.Regular.Render(label)
		}

		if len(item.Badge) != 0 {
			badge := m.Styles.Badge.Render(item.Badge)
			space := m.Width - lipgloss.Width(line) - lipgloss.Width(badge)
			line = lipgloss.JoinHorizontal(
				lipgloss.Bottom,
				line,
				lipgloss.NewStyle().PaddingRight(space).Render(" "),
				badge,
			)
		}

		lines = append(lines, line)
	}

	return lipgloss.JoinVertical(lipgloss.Top, lines...)
}

type Styles struct {
	Title           lipgloss.Style
	Badge           lipgloss.Style
	Regular         lipgloss.Style
	Selected        lipgloss.Style
	Highlighted     lipgloss.Style
	HighlightedBlur lipgloss.Style
}

func DefaultStyles() Styles {
	return Styles{
		Title:           lipgloss.NewStyle().PaddingLeft(2).Height(2).Foreground(lipgloss.Color("244")),
		Badge:           lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
		Regular:         lipgloss.NewStyle(),
		Selected:        lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true),
		Highlighted:     lipgloss.NewStyle().Foreground(lipgloss.Color("99")),
		HighlightedBlur: lipgloss.NewStyle(),
	}
}

type KeyMap struct {
	GoToTop  key.Binding
	GoToLast key.Binding

	Up   key.Binding
	Down key.Binding

	Select key.Binding
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Select}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Select},
		{k.GoToTop, k.GoToLast},
	}
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		GoToTop:  key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "first")),
		GoToLast: key.NewBinding(key.WithKeys("G"), key.WithHelp("G", "last")),

		Up:   key.NewBinding(key.WithKeys("k", "up", "ctrl+p"), key.WithHelp("k", "up")),
		Down: key.NewBinding(key.WithKeys("j", "down", "ctrl+n"), key.WithHelp("j", "down")),

		Select: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
	}
}

// Represents an item in the picker list.
type PickerItem struct {
	Label string
	Value int
	Badge string
}

func (item *PickerItem) FilterValue() string {
	return item.Label
}
