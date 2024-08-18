package utils

import (
	"fmt"

	"github.com/alexflint/go-arg"
	"github.com/charmbracelet/ssh"
)

// Parses args using go-arg and returns a boolean value indicating if the
// parse consumed the session. This usually happens when the client is requesting
// usage information.
func ParseArgs(session ssh.Session, name string, args []string, destination any) (consumed bool) {
	parser, _ := arg.NewParser(arg.Config{Program: name}, destination)

	// Borrowed from MustParse.
	err := parser.Parse(args)
	switch err {
	case nil:
		return false

	case arg.ErrHelp:
		parser.WriteHelpForSubcommand(session, parser.SubcommandNames()...)
		return true

	case arg.ErrVersion:
		fmt.Fprintln(session, "unknown")
		return true

	default:
		parser.WriteUsageForSubcommand(session, parser.SubcommandNames()...)
		fmt.Fprintln(session, "error:", err.Error())
		return true
	}
}
