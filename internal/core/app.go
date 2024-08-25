package core

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	"github.com/ksdme/mail/internal/apps"
	accounts "github.com/ksdme/mail/internal/apps/accounts/models"
	"github.com/ksdme/mail/internal/core/tui/colors"
)

type App interface {
	// Returns the name and description of the application.
	Info() (name string, title string, description string)

	// This method will be run before serving any request on this
	// application. You can initialize your workers here.
	Init()

	// An App can be booted up in two ways. The client can directly request
	// the services of a specific app using a sub command. When that happens,
	// this Handle method is invoked. It can then decide on presenting a tui
	// or a non-tui interface.
	// TODO: Structure the args and resources better.
	HandleRequest(
		next ssh.Handler,
		session ssh.Session,

		args apps.AppArgs,
		account accounts.Account,

		interactive bool,
		// The configuration that should be used if a tui is being served.
		// TODO: Figure out a better way to pass this.
		renderer *lipgloss.Renderer,
		palette colors.ColorPalette,
	) (int, error)

	// The other mode is when the application is selected from an interactive
	// application menu. When that happens, the application is not in charge of
	// handling the request, but, instead it should only return a tui interface.
	HandleApp(
		session ssh.Session,
		account accounts.Account,

		renderer *lipgloss.Renderer,
		palette colors.ColorPalette,

		quit tea.Cmd,
	) (tea.Model, func())

	// Called to close and cleanup the application during shutdown.
	CleanUp()
}
