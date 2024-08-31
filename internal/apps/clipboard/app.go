package clipboard

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	"github.com/ksdme/mail/internal/apps"
	accounts "github.com/ksdme/mail/internal/apps/accounts/models"
	"github.com/ksdme/mail/internal/apps/clipboard/events"
	"github.com/ksdme/mail/internal/apps/clipboard/models"
	"github.com/ksdme/mail/internal/apps/clipboard/tui"
	"github.com/ksdme/mail/internal/config"
	"github.com/ksdme/mail/internal/core/tui/colors"
	"github.com/ksdme/mail/internal/utils"
	"github.com/pkg/errors"
	"github.com/uptrace/bun"
)

type App struct {
	DB *bun.DB
}

func (a *App) Info() (string, string, string) {
	return "clipboard", "Clipboard", "Clipboard"
}

func (a *App) Init() {
	slog.Debug("initializing clipboard")

	// Set up clean up.
	models.CleanAll(context.Background(), a.DB)
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			models.CleanUp(context.Background(), a.DB)
		}
	}()
}

// Handle the incoming connection.
func (a *App) HandleRequest(
	next ssh.Handler,
	session ssh.Session,

	args apps.AppArgs,
	account accounts.Account,

	interactive bool,
	renderer *lipgloss.Renderer,
	palette colors.ColorPalette,
) (int, error) {
	// Show a tui only if we are in interactive mode and there are no
	// explicit arguments.
	if interactive {
		if args.Clipboard.Put == nil && args.Clipboard.Clear == nil {
			defer a.cleanUpSession(account)
			utils.RunTeaInSession(next, session, tui.NewModel(
				a.DB,
				account,
				renderer,
				palette,
				tea.Quit,
			))
			return 0, nil
		}

		// TODO: Recommend a solution.
		return 1, fmt.Errorf("command not supported in interactive mode")
	}

	// Otherwise, process the command.
	switch {
	case args.Clipboard.Put != nil:
		r := io.LimitReader(session, int64(config.Clipboard.MaxContentSize)+1)
		value, err := io.ReadAll(r)
		if err != nil {
			return 1, errors.Wrap(err, "could not read contents")
		}
		if len(value) > config.Clipboard.MaxContentSize {
			return 1, fmt.Errorf(
				"could not put on the clipboard: contents exceed the max size limit of %d bytes",
				config.Clipboard.MaxContentSize,
			)
		}
		if !utf8.Valid(value) {
			return 1, fmt.Errorf(
				"could not put on the clipboard: contents are not a text string",
			)
		}

		// Save the value.
		err = models.CreateClipboardItem(session.Context(), a.DB, value, account)
		if err != nil {
			return 1, errors.Wrap(err, "could not put on the clipboard")
		}

		return 0, nil

	case args.Clipboard.Clear != nil:
		err := models.DeleteClipboard(session.Context(), a.DB, account)
		if err != nil {
			return 1, errors.Wrap(err, "could not clear the clipboard")
		}

		return 0, nil

	default:
		item, err := models.GetClipboardValue(session.Context(), a.DB, account)
		if err != nil {
			return 1, errors.Wrap(err, "could not fetch the clipboard")
		}
		if item == nil {
			return 1, nil
		}

		_, err = session.Write(item.Value)
		if err != nil {
			slog.Debug("could not write the clipboard to session")
			return 1, errors.Wrap(err, "could not write to the session")
		}

		return 0, nil
	}
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
	events.ClipboardContentsUpdatedSignal.CleanUp(account.ID)
}

func (a *App) CleanUp() {
	slog.Debug("cleaning up clipboard")
}
