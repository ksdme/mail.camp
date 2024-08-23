package clipboard

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	"github.com/ksdme/mail/internal/apps/clipboard/models"
	"github.com/ksdme/mail/internal/apps/clipboard/tui"
	core "github.com/ksdme/mail/internal/core/models"
	"github.com/ksdme/mail/internal/core/tui/colors"
	"github.com/ksdme/mail/internal/utils"
	"github.com/pkg/errors"
	"github.com/uptrace/bun"
)

type cli struct {
	Get *struct{} `arg:"subcommand:get"`

	// TODO: Support explicitly passing contents as a positional
	// argument to this subcommand. This doesn't work at the moment
	// because something somewhere is breaking down a value with spaces
	// into separate arguments.
	Put *struct{} `arg:"subcommand:put"`

	Clear *struct{} `arg:"subcommand:clear"`
}

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
func (a *App) Handle(
	next ssh.Handler,
	session ssh.Session,

	args []string,
	account core.Account,

	interactive bool,
	renderer *lipgloss.Renderer,
	palette colors.ColorPalette,
) error {
	// Handle cli level behavior.
	var cli cli
	if utils.ParseArgs(session, "ssh.camp clipboard", args, &cli) {
		return nil
	}

	// Show a tui only if we are in interactive mode and there are no
	// explicit arguments.
	if interactive && len(args) == 0 {
		utils.RunTeaInSession(next, session, tui.NewModel(
			a.DB,
			account,
			session.PublicKey(),
			renderer,
			palette,
		))
		return nil
	}

	if interactive {
		fmt.Fprintln(session, "todo: interactive mode is not supported for this command")
		return nil
	}

	// Otherwise, process the command.
	switch {
	case cli.Put != nil:
		// Read the value from the connection.
		value, err := io.ReadAll(session)
		if err != nil {
			return errors.Wrap(err, "could not read contents to put")
		}

		// Save the value.
		err = models.CreateClipboardItem(
			session.Context(),
			a.DB,
			value,
			session.PublicKey(),
			account,
		)
		if err != nil {
			return errors.Wrap(err, "could not put to the clipboard")
		}

		return nil

	case cli.Clear != nil:
		err := models.DeleteClipboard(session.Context(), a.DB, account)
		if err != nil {
			return errors.Wrap(err, "could not clear the clipboard")
		}

		return nil

	default:
		item, err := models.GetClipboardValue(session.Context(), a.DB, session.PublicKey(), account)
		if err != nil {
			return errors.Wrap(err, "could not fetch the clipboard")
		}

		_, err = session.Write(item.Value)
		if err != nil {
			slog.Debug("could not write the clipboard to session")
			return errors.Wrap(err, "could not write to the session")
		}

		return nil
	}
}

func (a *App) CleanUp() {
	slog.Debug("cleaning up clipboard")
}
