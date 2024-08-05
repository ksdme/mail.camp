package config

const (
	DevBuild = true

	MxHost = "localhost"
	DbURI  = "file:db.sqlite3"

	SMTPBindAddr          = "127.0.0.1:1025"
	SSHBindAddr           = "127.0.0.1:2222"
	SSHHostKeyPath        = "/home/kilari/.ssh/id_rsa"
	SSHAuthorizedKeysPath = ""

	Signature = "mail.ssh.camp by @ksdme"
)
