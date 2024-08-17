package mail

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/emersion/go-smtp"
	"github.com/ksdme/mail/internal/apps/mail/backend"
	"github.com/ksdme/mail/internal/apps/mail/events"
	"github.com/ksdme/mail/internal/apps/mail/models"
	"github.com/ksdme/mail/internal/apps/mail/tui"
	"github.com/ksdme/mail/internal/config"
	core "github.com/ksdme/mail/internal/core/models"
	"github.com/ksdme/mail/internal/core/tui/colors"
	"github.com/uptrace/bun"
)

// Represents the temporary mail application.
type App struct {
	DB     *bun.DB
	server *smtp.Server
}

func (m *App) Info() (string, string, string) {
	return "mail", "Mail", "Temporary mail"
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
	args []string,
	pipe io.ReadWriter,
	account core.Account,
) (func() error, func(), error) {
	return nil, nil, fmt.Errorf("mail app does not support non-tui mode")
}

func (m *App) HandleTUI(
	args []string,
	account core.Account,
	renderer *lipgloss.Renderer,
	palette colors.ColorPalette,
) (tea.Model, func(), error) {
	return tui.NewModel(m.DB, account, renderer, palette), m.cleanUpSession(account), nil
}

func (m *App) CleanUp() {
	m.server.Shutdown(context.TODO())
}

func (m *App) cleanUpSession(account core.Account) func() {
	return func() {
		events.MailboxContentsUpdatedSignal.CleanUp(account.ID)
	}
}
