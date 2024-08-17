package utils

import (
	"io"

	"github.com/charmbracelet/ssh"
)

func WriteStringToSSH(s ssh.Session, text string) {
	io.WriteString(s, text+"\n")
}
