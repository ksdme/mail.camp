package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

// TODO: Support configuring from cli flags and configuration files too.

// General server level configuration.
type coreSettings struct {
	Debug bool `envDefault:"true"`

	Entropy string `env:"ENTROPY,required"`

	DBURI     string `env:"DB_URI" envDefault:"file:db.sqlite3"`
	DBMigrate bool   `env:"DB_MIGRATE"`

	SSHHostKeyPath        string `env:"SSH_HOST_KEY_PATH,expand" envDefault:"${HOME}/.ssh/id_rsa"`
	SSHAuthorizedKeysPath string `env:"SSH_AUTHORIZED_KEYS_PATH"`
	SSHBindAddr           string `env:"SSH_BIND_ADDR" envDefault:"127.0.0.1:2222"`

	MailAppEnabled      bool `env:"MAIL_APP_ENABLED" envDefault:"true"`
	ClipboardAppEnabled bool `env:"CLIPBOARD_APP_ENABLED" envDefault:"true"`
}

// Settings related to the mail app.
type mailSettings struct {
	MXHost       string `env:"MX_HOST" envDefault:"localhost"`
	SMTPBindAddr string `env:"SMTP_BIND_ADDR" envDefault:"127.0.0.1:1025"`
	Signature    string `env:",expand" envDefault:"${MX_HOST}"`
}

// Settings related to the clipboard app.
type clipboardSettings struct {
	MaxContentSize string `env:"CLIPBOARD_MAX_CONTENTS_SIZE" envDefault:"8208"`
}

func init() {
	if err := env.Parse(&Core); err != nil {
		panic(fmt.Sprintf("could not parse core configuration: %v", err))
	}

	if Core.MailAppEnabled {
		if err := env.Parse(&Mail); err != nil {
			panic(fmt.Sprintf("could not parse mail configuration: %v", err))
		}
	}

	if Core.ClipboardAppEnabled {
		if err := env.Parse(&Clipboard); err != nil {
			panic(fmt.Sprintf("could not parse clipboard configuration: %v", err))
		}
	}
}

var Core coreSettings

// While it would be nicer to not have a global reference to these settings so
// we don't accidentally read it when the app is disabled, having it be global
// makes our life slightly easier for now.
var Mail mailSettings
var Clipboard clipboardSettings
