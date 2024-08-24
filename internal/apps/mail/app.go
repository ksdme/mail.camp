package mail

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	"github.com/emersion/go-smtp"
	"github.com/ksdme/mail/internal/apps"
	"github.com/ksdme/mail/internal/apps/mail/backend"
	"github.com/ksdme/mail/internal/apps/mail/events"
	"github.com/ksdme/mail/internal/apps/mail/models"
	"github.com/ksdme/mail/internal/apps/mail/tui"
	"github.com/ksdme/mail/internal/config"
	core "github.com/ksdme/mail/internal/core/models"
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
	return "mail", "Mail", "Mail"
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

func (m *App) Handle(
	next ssh.Handler,
	session ssh.Session,

	args apps.AppArgs,
	account core.Account,

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
	defer events.MailboxContentsUpdatedSignal.CleanUp(account.ID)
	utils.RunTeaInSession(
		next,
		session,
		tui.NewModel(m.DB, account, renderer, palette),
	)
	return 0, nil
}

func (m *App) CleanUp() {
	m.server.Shutdown(context.TODO())
}
