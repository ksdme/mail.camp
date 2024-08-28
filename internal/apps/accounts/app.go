package accounts

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	"github.com/ksdme/mail/internal/apps"
	accounts "github.com/ksdme/mail/internal/apps/accounts/models"
	"github.com/ksdme/mail/internal/core/tui/colors"
	"github.com/pkg/errors"
	"github.com/uptrace/bun"
)

type App struct {
	DB *bun.DB
}

func (a *App) Info() (string, string, string) {
	return "accounts", "Accounts", "Accounts"
}

func (a *App) Init() {
}

func (a *App) HandleRequest(
	next ssh.Handler,
	session ssh.Session,

	args apps.AppArgs,
	account accounts.Account,

	interactive bool,
	renderer *lipgloss.Renderer,
	palette colors.ColorPalette,
) (int, error) {
	switch {
	case args.Accounts.AddKey != nil:
		err := account.AddKey(session.Context(), a.DB, args.Accounts.AddKey.Key)
		if err != nil {
			return 1, errors.Wrap(err, "could not add key")
		}

	case args.Accounts.RemoveKey != nil:
		err := account.RemoveKey(session.Context(), a.DB, args.Accounts.RemoveKey.Key)
		if err != nil {
			return 1, errors.Wrap(err, "could not remove key")
		}

	case args.Accounts.ListKeys != nil:
		keys, err := account.ListKeys(session.Context(), a.DB)
		if err != nil {
			return 1, errors.Wrap(err, "could not list keys")
		}

		lines := []string{}
		for _, key := range keys {
			lines = append(
				lines,
				fmt.Sprintf(
					"%s %s",
					key.CreatedAt.Format(time.DateTime),
					key.Fingerprint,
				),
			)
		}

		fmt.Fprintln(session, strings.Join(lines, "\n"))
		return 0, nil

	default:
		return 1, fmt.Errorf("unknown operation")
	}

	return 0, nil
}

func (a *App) HandleApp(
	session ssh.Session,
	account accounts.Account,

	renderer *lipgloss.Renderer,
	palette colors.ColorPalette,

	quit tea.Cmd,
) (tea.Model, func()) {
	return nil, nil
}

func (a *App) CleanUp() {
}
