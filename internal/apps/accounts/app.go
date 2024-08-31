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
	"github.com/ksdme/mail/internal/utils"
	"github.com/pkg/errors"
	"github.com/uptrace/bun"
)

type App struct {
	DB *bun.DB
}

func (a *App) Info() (string, string, string) {
	return "accounts", "Accounts", "Manage your account."
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
			lines = append(lines, fmt.Sprintf(
				"%s %s",
				key.CreatedAt.Format(time.DateTime),
				key.Fingerprint,
			))
		}

		if len(lines) > 0 {
			fmt.Fprintln(session, strings.Join(lines, "\n"))
		}
		return 0, nil

	case args.Accounts.DeleteAccount != nil:
		delete := utils.AskConsent(
			session,
			"This operation will delete your account on ssh.camp.\n"+
				"Are you sure? (yes/no) ",
		)
		if !delete {
			return 1, fmt.Errorf("aborting account deletion operation")
		}

		err := account.Delete(session.Context(), a.DB)
		if err != nil {
			return 1, errors.Wrap(err, "could not delete account")
		}

	case args.Accounts.ListTokens != nil:
		tokens, err := account.ListTokens(session.Context(), a.DB)
		if err != nil {
			return 1, errors.Wrap(err, "could not list tokens")
		}

		lines := []string{}
		for _, token := range tokens {
			lines = append(lines, fmt.Sprintf(
				"%s %s",
				token.CreatedAt.Format(time.Stamp),
				token.Name,
			))
		}

		if len(lines) > 0 {
			fmt.Fprintln(session, strings.Join(lines, "\n"))
		}
		return 0, nil

	case args.Accounts.IssueToken != nil:
		var zero time.Duration

		expiry := time.Now().Add(3 * 24 * time.Hour)
		if args.Accounts.IssueToken.Validity != zero {
			expiry = time.Now().Add(args.Accounts.IssueToken.Validity)
		}

		token, err := account.IssueToken(session.Context(), a.DB, expiry)
		if err != nil {
			return 1, errors.Wrap(err, "could not issue token")
		}

		fmt.Fprintf(
			session,
			"%s\n\n"+
				"You can use it to login with,\n"+
				"ssh %s@ssh.camp\n",
			token.Token,
			token.Token,
		)
		return 0, nil

	case args.Accounts.RemoveToken != nil:
		affected, err := account.RemoveToken(session.Context(), a.DB, args.Accounts.RemoveToken.Name)
		if err != nil {
			return 1, errors.Wrap(err, "could not delete tokens")
		}

		if affected == -1 {
			fmt.Fprintln(session, "unknown number of tokens deleted")
		} else {
			fmt.Fprintf(session, "%d token(s) deleted\n", affected)
		}
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
