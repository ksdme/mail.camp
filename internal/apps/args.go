package apps

// For the sake of presenting a global help command, we aggregate all the
// supported subcommands here.
type AppArgs struct {
	// Mail application.
	Mail *struct{} `arg:"subcommand:mail" help:"a disposable email app"`

	// Clipboard application.
	Clipboard *struct {
		Get *struct{} `arg:"subcommand:get" help:"retrieve contents currently on the clipboard"`

		// TODO: Support explicitly passing contents as a positional
		// argument to this subcommand. This doesn't work at the moment
		// because something somewhere is breaking down a value with spaces
		// into separate arguments.
		Put *struct{} `arg:"subcommand:put" help:"put text on the clipboard"`

		Clear *struct{} `arg:"subcommand:clear" help:"clear the contents on the clipboard"`
	} `arg:"subcommand:clipboard" help:"a clipboard app"`
}
