package apps

import (
	"io"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ksdme/mail/internal/core/models"
	"github.com/ksdme/mail/internal/core/tui/colors"
)

type App interface {
	// Returns the name and description of the application.
	Info() (name string, title string, description string)

	// This method will be run before serving any request on this
	// application. You can initialize your workers here.
	Init()

	// If the application supports a tui, return the corresponding
	// tea model. You can optionally also provide a clean up method
	// that will be called after the session ends.
	// TODO: Maybe have a struct to pack these args.
	HandleTUI(
		args []string,
		account models.Account,
		renderer *lipgloss.Renderer,
		palette colors.ColorPalette,
	) (tea.Model, func(), error)

	// Serve a non-tui session. You can optionally also return a clean up
	// method that will be called after the session ends.
	Handle(
		args []string,
		pipe io.ReadWriter,
		account models.Account,
	) (func() error, func(), error)

	// Called to close and cleanup the application during shutdown.
	CleanUp()
}
