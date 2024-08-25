package tui

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	accounts "github.com/ksdme/mail/internal/apps/accounts/models"
	"github.com/ksdme/mail/internal/apps/clipboard/events"
	"github.com/ksdme/mail/internal/apps/clipboard/models"
	"github.com/ksdme/mail/internal/core/tui/colors"
	"github.com/ksdme/mail/internal/core/tui/components/help"
	"github.com/ksdme/mail/internal/utils"
	"github.com/muesli/reflow/wordwrap"
	"github.com/uptrace/bun"
)

type clipboardRealtimeUpdate struct{}

// The clipboard tui.
// At the moment, it only displays the current contents on the
// clipboard and/or the instructions on how to use it.
type Model struct {
	db      *bun.DB
	account accounts.Account
	key     ssh.PublicKey

	item *models.DecodedClipboardItem

	width  int
	height int

	renderer *lipgloss.Renderer
	palette  colors.ColorPalette
	keymap   KeyMap

	quit     tea.Cmd
	quitting bool
}

func NewModel(
	db *bun.DB,
	account accounts.Account,
	key ssh.PublicKey,
	renderer *lipgloss.Renderer,
	palette colors.ColorPalette,
	quit tea.Cmd,
) Model {
	return Model{
		db:      db,
		account: account,
		key:     key,

		item: nil,

		renderer: renderer,
		palette:  palette,
		keymap:   DefaultKeyMap(),

		quit: quit,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.loadClipboard,
		m.listenToClipboardUpdate,
	)
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
			return m, m.quit

		case key.Matches(msg, m.keymap.Clear):
			return m, m.clearClipboard
		}

	case tea.QuitMsg:
		m.quitting = true
		return m, nil

	case clipboardRealtimeUpdate:
		return m, tea.Batch(
			m.loadClipboard,
			m.listenToClipboardUpdate,
		)

	case *models.DecodedClipboardItem:
		m.item = msg
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

func (m Model) listenToClipboardUpdate() tea.Msg {
	slog.Debug("listening to clipboard updates", "account", m.account.ID)
	if _, aborted := events.ClipboardContentsUpdatedSignal.Wait(m.account.ID); !aborted {
		return clipboardRealtimeUpdate{}
	}

	return nil
}

func (m Model) loadClipboard() tea.Msg {
	// TODO: Handler error.
	item, err := models.GetClipboardValue(context.TODO(), m.db, m.key, m.account)
	if err != sql.ErrNoRows {
		slog.Error("could not get clipboard value", "err", err)
	}
	return item
}

func (m Model) clearClipboard() tea.Msg {
	// TODO: Handle error.
	err := models.DeleteClipboard(context.TODO(), m.db, m.account)
	slog.Error("could not clear the clipboard", "err", err)

	var item *models.DecodedClipboardItem = nil
	return item
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
