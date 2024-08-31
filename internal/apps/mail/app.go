package mail

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	"github.com/emersion/go-smtp"
	"github.com/ksdme/mail/internal/apps"
	accounts "github.com/ksdme/mail/internal/apps/accounts/models"
	"github.com/ksdme/mail/internal/apps/mail/backend"
	"github.com/ksdme/mail/internal/apps/mail/events"
	"github.com/ksdme/mail/internal/apps/mail/models"
	"github.com/ksdme/mail/internal/apps/mail/tui"
	"github.com/ksdme/mail/internal/config"
	"github.com/ksdme/mail/internal/core/tui/colors"
	"github.com/ksdme/mail/internal/utils"
	"github.com/uptrace/bun"
)

// Represents the temporary mail application.
type App struct {
	DB     *bun.DB
	server *smtp.Server
}

func (m *App) Info() (string, string, string) {
	description := "You can use it for things like registering on websites that " +
		"don't respect\nyour mailbox, or, to receive emails from your development server."
	return "mail", "Disposable Mailboxes", description
}

func (m *App) Init() {
	m.server = smtp.NewServer(backend.NewBackend(m.DB))
	m.server.Addr = config.Mail.SMTPBindAddr
	m.server.Domain = config.Mail.MXHost

	// SMTP Server.
	go func() {
		slog.Info("starting smtp server", "at", config.Mail.SMTPBindAddr)
		if err := m.server.ListenAndServe(); err != nil {
			panic(fmt.Sprintf("failed serving smtp server: %v", err))
		}
	}()

	// Mail clean up worker.
	go func() {
		for {
			models.CleanupMails(context.Background(), m.DB)
			time.Sleep(time.Hour)
		}
	}()
}

func (m *App) HandleRequest(
	next ssh.Handler,
	session ssh.Session,

	args apps.AppArgs,
	account accounts.Account,

	interactive bool,
	renderer *lipgloss.Renderer,
	palette colors.ColorPalette,
) (int, error) {
	// TODO: Because the help message will be empty, we should explicitly mention
	// that the ssh.camp mail is tui only at the moment.
	// Email at the moment only supports a tui mode.
	// But, we could show a help message if args has it.

	// Otherwise, complain if we are running in an non-interactive mode.
	if !interactive {
		fmt.Fprintln(session, "mail app can only be run interactively")
		return 1, nil
	}

	// And, then, run the tea application.
	defer m.cleanUpSession(account)
	utils.RunTeaInSession(
		next,
		session,
		tui.NewModel(m.DB, account, renderer, palette, tea.Quit),
	)
	return 0, nil
}

func (a *App) HandleApp(
	session ssh.Session,
	account accounts.Account,

	renderer *lipgloss.Renderer,
	palette colors.ColorPalette,

	quit tea.Cmd,
) (tea.Model, func()) {
	model := tui.NewModel(
		a.DB,
		account,
		renderer,
		palette,
		quit,
	)
	cleanup := func() {
		a.cleanUpSession(account)
	}
	return model, cleanup
}

func (a *App) cleanUpSession(account accounts.Account) {
	events.MailboxContentsUpdatedSignal.CleanUp(account.ID)
}

func (m *App) CleanUp() {
	m.server.Shutdown(context.TODO())
}
