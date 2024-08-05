package utils

import "strings"

func Decode(text string) string {
	// SMTP uses \r\n for line delimiting. Whereas bubbletea and SSH
	// seems to misbehave with this sometimes.
	// TODO: Dig into this more.
	return strings.ReplaceAll(text, "\r\n", "\n")
}
