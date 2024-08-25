package home

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	accounts "github.com/ksdme/mail/internal/apps/accounts/models"
	"github.com/ksdme/mail/internal/apps/mail/events"
	"github.com/ksdme/mail/internal/apps/mail/models"
	"github.com/ksdme/mail/internal/apps/mail/tui/email"
	"github.com/ksdme/mail/internal/core/tui/colors"
	"github.com/ksdme/mail/internal/core/tui/components/picker"
	"github.com/ksdme/mail/internal/core/tui/components/table"
	"github.com/ksdme/mail/internal/utils"
	"github.com/uptrace/bun"
)

type MailboxRealTimeUpdate struct {
	mailbox int64
}

type mailboxWithUnread struct {
	models.Mailbox
	Unread int
}

type mailboxesRefreshedMsg struct {
	passive   bool
	mailboxes []mailboxWithUnread
	err       error
}

type mailsRefreshedMsg struct {
	mailbox *mailboxWithUnread
	mails   []models.Mail
	err     error
}

type Model struct {
	db      *bun.DB
	account accounts.Account

	mailboxes picker.Model
	mailbox   *mailboxWithUnread
	mails     table.Model

	Width  int
	Height int

	KeyMap   KeyMap
	Renderer *lipgloss.Renderer
	Colors   colors.ColorPalette
}

func NewModel(db *bun.DB, renderer *lipgloss.Renderer, colors colors.ColorPalette) Model {
	width := 80
	height := 80

	// Setup the mails picker.
	pStyles := picker.DefaultStyles(renderer)
	pStyles.Title = pStyles.Title.Foreground(colors.Muted)
	pStyles.Badge = pStyles.Badge.Foreground(colors.Muted)
	pStyles.Regular = pStyles.Regular.Foreground(colors.Text)
	pStyles.Highlighted = pStyles.Highlighted.Foreground(colors.Accent).Bold(true)
	pStyles.SelectedLegend = pStyles.SelectedLegend.Foreground(colors.Accent)
	mailboxes := picker.NewModel("Mailboxes", []picker.Item{}, width/3, height)
	mailboxes.Styles = pStyles
	mailboxes.Focus()

	// Setup the mails table.
	tStyles := table.DefaultStyles(renderer)
	tStyles.Header = renderer.NewStyle().Height(2).Foreground(colors.Muted).PaddingLeft(1)
	tStyles.Selected = tStyles.Selected.Foreground(colors.Accent).Bold(true)
	tStyles.Cell = tStyles.Cell.PaddingLeft(1)
	table := table.New(
		table.WithColumns(makeMailTableColumns(width*2/3)),
		table.WithHeight(height),
		table.WithRows([]table.Row{}),
		table.WithStyles(tStyles),
	)

	return Model{
		db: db,

		mailboxes: mailboxes,
		mails:     table,

		Width:  width,
		Height: height,

		KeyMap:   DefaultKeyMap(),
		Renderer: renderer,
		Colors:   colors,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.refreshMailboxes(false),
		m.listenToMailboxUpdate,
	)
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		gap := 6

		m.mailboxes.Width = m.Width / 3
		m.mailboxes.Height = m.Height

		m.mails.SetWidth(m.Width - m.mailboxes.Width - gap)
		m.mails.SetHeight(m.Height)
		m.mails.SetColumns(makeMailTableColumns(m.mails.Width()))

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.KeyMap.FocusMailboxes):
			m.mailboxes.Focus()
			m.mails.Blur()

		case key.Matches(msg, m.KeyMap.FocusMails):
			if m.mails.HasRows() {
				m.mailboxes.Blur()
				m.mails.Focus()
			}

		case key.Matches(msg, m.KeyMap.Select):
			if m.mailboxes.IsFocused() {
				if item := m.mailboxes.Select(); item != nil {
					m.mailbox = item.(*mailboxItem).mailbox

					m.mails.SetRows([]table.Row{})
					m.mailboxes.Blur()
					m.mails.Focus()

					return m, m.refreshMails(m.mailbox)
				}
			} else if m.mails.Focused() {
				if row, err := m.mails.SelectedRow(); err == nil {
					mail := row.Value.(models.Mail)
					return m, tea.Batch(
						m.mailSelected(m.mailbox, mail),
						m.markMailSeen(mail),
					)
				}
			}

		case key.Matches(msg, m.KeyMap.CreateRandomMailbox):
			return m, m.createRandomMailbox

		case key.Matches(msg, m.KeyMap.DeleteMailbox):
			if item := m.mailboxes.HighlightedItem(); item != nil {
				mailbox := item.(*mailboxItem).mailbox
				return m, m.deleteMailbox(mailbox)
			}
		}

	case MailboxRealTimeUpdate:
		// TODO: Maybe don't refresh if the home view is not active.
		slog.Debug("received mailbox update", "mailbox", msg.mailbox)
		if msg.mailbox == m.mailbox.ID {
			return m, tea.Batch(
				// TODO: Debounce these loads.
				m.refreshMails(m.mailbox),
				m.refreshMailboxes(true),
				m.listenToMailboxUpdate,
			)
		}
		return m, tea.Batch(
			// TODO: Debounce this load.
			m.refreshMailboxes(true),
			m.listenToMailboxUpdate,
		)

	case mailboxesRefreshedMsg:
		// TODO: Handle error.
		var items []picker.Item
		for _, mailbox := range msg.mailboxes {
			items = append(items, &mailboxItem{
				mailbox: &mailbox,
			})
		}
		m.mailboxes.SetItems(items)

		// Trigger mails load.
		if !msg.passive {
			m.mails.SetRows([]table.Row{})
			if m.mailboxes.HasItems() {
				if item := m.mailboxes.SelectedItem(); item != nil {
					m.mailbox = item.(*mailboxItem).mailbox
					return m, m.refreshMails(m.mailbox)
				}
			}
		}
		return m, nil

	case mailsRefreshedMsg:
		// TODO: Handle error.
		if msg.mailbox.ID == m.mailbox.ID {
			var items []table.Row
			for _, mail := range msg.mails {
				age := fmt.Sprintf(
					"%s ago",
					utils.RoundedAge(time.Since(mail.CreatedAt)),
				)

				items = append(items, table.Row{
					ID:    int(mail.ID),
					Value: mail,
					Cols:  []string{mail.Subject, mail.FromAddress, age},
				})
			}
			m.mails.SetRows(items)

			// If the update caused there to be no mails.
			if !m.mails.HasRows() {
				m.mails.Blur()
				m.mailboxes.Focus()
			}
			return m, nil
		} else {
			return m, nil
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
	if !m.mailboxes.HasItems() {
		return m.Renderer.
			NewStyle().
			Width(m.Width).
			Height(m.Height).
			AlignHorizontal(lipgloss.Center).
			AlignVertical(lipgloss.Center).
			Foreground(m.Colors.Muted).
			Render("no mailboxes :(")
	}

	mailboxes := m.Renderer.
		NewStyle().
		PaddingRight(5).
		Foreground(m.Colors.Text).
		Render(m.mailboxes.View())

	var mails string
	if !m.mails.HasRows() {
		mails = lipgloss.JoinVertical(
			lipgloss.Top,
			m.mailboxes.
				Styles.
				Title.
				PaddingLeft(0).
				Render("Mails"),
			m.Renderer.
				NewStyle().
				Width(m.mails.Width()).
				Height(m.mails.Height()).
				Foreground(m.Colors.Text).
				Render(fmt.Sprintf(
					"no mails in %s, incoming mails are only stored for 48h",
					m.mailbox.Email(),
				)),
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

func (m Model) refreshMailboxes(passive bool) tea.Cmd {
	return func() tea.Msg {
		var mailboxes []mailboxWithUnread

		var mailbox *models.Mailbox
		err := m.db.NewSelect().
			Model(mailbox).
			Column("mailbox.*").
			ColumnExpr("COUNT(mail.id) AS unread").
			Where("mailbox.account_id = ?", m.account.ID).
			Join("LEFT JOIN mails AS mail").
			JoinOn("mail.mailbox_id = mailbox.id").
			JoinOn("mail.seen = false").
			Order("mailbox.id DESC").
			Group("mailbox.id").
			Scan(context.TODO(), &mailboxes)

		return mailboxesRefreshedMsg{
			passive:   passive,
			mailboxes: mailboxes,
			err:       err,
		}
	}
}

func (m Model) refreshMails(mailbox *mailboxWithUnread) tea.Cmd {
	return func() tea.Msg {
		var mails []models.Mail

		err := m.db.NewSelect().
			Model(&mails).
			Where("mailbox_id = ?", mailbox.ID).
			Order("id DESC").
			Scan(context.TODO())

		return mailsRefreshedMsg{
			mailbox: mailbox,
			mails:   mails,
			err:     err,
		}
	}
}

func (m Model) createRandomMailbox() tea.Msg {
	_, err := models.CreateRandomMailbox(context.TODO(), m.db, m.account)
	if err != nil {
		// TODO: Handle this error.
		slog.Error("could not create mailbox", "err", err)
		return nil
	}

	return m.refreshMailboxes(false)()
}

func (m Model) deleteMailbox(mailbox *mailboxWithUnread) tea.Cmd {
	return func() tea.Msg {
		_, err := m.db.
			NewDelete().
			Model(&models.Mailbox{}).
			Where("id = ?", mailbox.ID).
			Exec(context.TODO())
		if err != nil {
			slog.Error("could not delete mailbox", "mailbox", mailbox.ID, "err", err)
			return nil
		}

		_, err = m.db.
			NewDelete().
			Model(&models.Mail{}).
			Where("mailbox_id = ?", mailbox.ID).
			Exec(context.TODO())
		if err != nil {
			slog.Error("could not delete mails", "mailbox", mailbox.ID, "err", err)
			return nil
		}

		return m.refreshMailboxes(false)()
	}
}

func (m Model) listenToMailboxUpdate() tea.Msg {
	slog.Debug("listening to mailbox updates", "account", m.account.ID)
	if value, aborted := events.MailboxContentsUpdatedSignal.Wait(m.account.ID); !aborted {
		return MailboxRealTimeUpdate{value}
	}

	return nil
}

func (m Model) markMailSeen(mail models.Mail) tea.Cmd {
	return func() tea.Msg {
		if !mail.Seen {
			mail.Seen = true

			_, err := m.db.NewUpdate().Model(&mail).WherePK().Exec(context.TODO())
			if err != nil {
				slog.Error("could not mark email read", "mail", mail.ID, "err", err)
			}

			if m.mailbox != nil && mail.MailboxID == m.mailbox.ID {
				m.mailbox.Unread -= 1
			}
		}

		return nil
	}
}

func (m Model) mailSelected(mailbox *mailboxWithUnread, mail models.Mail) tea.Cmd {
	return func() tea.Msg {
		return email.MailSelectedMsg{To: mailbox.Email(), Mail: mail}
	}
}

func (m Model) Help() []key.Binding {
	var help []key.Binding

	if m.mailboxes.IsFocused() {
		help = append(
			help,
			m.KeyMap.CreateRandomMailbox,
			m.KeyMap.DeleteMailbox,
			m.KeyMap.Select,
			m.KeyMap.FocusMails,
		)
	} else if m.mails.Focused() {
		help = append(
			help,
			m.KeyMap.Select,
			m.KeyMap.FocusMailboxes,
		)
	}

	return help
}

type KeyMap struct {
	CreateRandomMailbox key.Binding
	DeleteMailbox       key.Binding

	Select key.Binding

	FocusMailboxes key.Binding
	FocusMails     key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		CreateRandomMailbox: key.NewBinding(
			key.WithKeys("ctrl+n"),
			key.WithHelp("ctrl+n", "generate mailbox"),
		),
		DeleteMailbox: key.NewBinding(
			key.WithKeys("ctrl+k"),
			key.WithHelp("ctrl+k", "delete mailbox"),
		),

		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),

		FocusMailboxes: key.NewBinding(
			key.WithKeys("left", "h", "esc"),
			key.WithHelp("←/h", "mailboxes"),
		),
		FocusMails: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "mails"),
		),
	}
}

type mailboxItem struct {
	mailbox *mailboxWithUnread
}

func (m *mailboxItem) ID() int {
	return int(m.mailbox.ID)
}

func (m *mailboxItem) Label() string {
	return m.mailbox.Email()
}

func (m *mailboxItem) Badge() string {
	return strconv.Itoa(m.mailbox.Unread)
}

func makeMailTableColumns(width int) []table.Column {
	at := width * 1 / 10
	from := (width * 3) / 10
	return []table.Column{
		{Title: "Subject", Width: width - at - from},
		{Title: "From", Width: from},
		{Title: "At", Width: at},
	}
}
