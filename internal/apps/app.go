package apps

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	"github.com/ksdme/mail/internal/core/models"
	"github.com/ksdme/mail/internal/core/tui/colors"
)

type App interface {
	// Returns the name and description of the application.
	Info() (name string, title string, description string)

	// This method will be run before serving any request on this
	// application. You can initialize your workers here.
	Init()

	// Handle should handle the incoming request to this application.
	// The application can decide to present a tui interface or a non tui interface.
	// TODO: Structure the args and resources better.
	Handle(
		next ssh.Handler,
		session ssh.Session,

		args []string,
		account models.Account,

		interactive bool,
		renderer *lipgloss.Renderer,
		palette colors.ColorPalette,
	) (int, error)

	// Called to close and cleanup the application during shutdown.
	CleanUp()
}
