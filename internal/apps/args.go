package apps

import "time"

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

	// Accounts application.
	Accounts *struct {
		// Keys
		ListKeys *struct{} `arg:"subcommand:list-keys" help:"list all keys attached to the account"`

		AddKey *struct {
			Key string `arg:"positional,required" help:"SHA256 fingerprint of the key to add"`
		} `arg:"subcommand:add-key" help:"add a key to your account"`

		RemoveKey *struct {
			Key string `arg:"positional,required" help:"SHA256 fingerprint of the key to remove"`
		} `arg:"subcommand:remove-key" help:"add a key to your account"`

		// Keyless login.
		ListTokens *struct{} `arg:"subcommand:list-tokens" help:"list all previously issued keyless login tokens"`

		IssueToken *struct {
			Validity time.Duration `help:"duration of the token validity"`
		} `arg:"subcommand:issue-token" help:"issue a new token for keyless login"`

		RemoveToken *struct {
			Name string `arg:"positional,required"`
		} `arg:"subcommand:remove-token" help:"remove a previously issued keyless login token"`

		// Account.
		DeleteAccount *struct{} `arg:"subcommand:delete-account" help:"delete the current account"`
	} `arg:"subcommand:accounts" help:"manage your account"`
}
