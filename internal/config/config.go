package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

// General server level configuration.
type coreSettings struct {
	Debug bool `envDefault:"true"`

	DBURI     string `env:"DB_URI" envDefault:"file:db.sqlite3"`
	DBMigrate bool   `env:"DB_MIGRATE"`

	SSHHostKeyPath        string `env:"SSH_HOST_KEY_PATH,expand" envDefault:"${HOME}/.ssh/id_rsa"`
	SSHAuthorizedKeysPath string `env:"SSH_AUTHORIZED_KEYS_PATH"`
	SSHBindAddr           string `env:"SSH_BIND_ADDR" envDefault:"127.0.0.1:2222"`
}

// Settings related to the mail app.
type mailSettings struct {
	MXHost       string `env:"MX_HOST" envDefault:"localhost"`
	SMTPBindAddr string `env:"SMTP_BIND_ADDR" envDefault:"127.0.0.1:1025"`
	Signature    string `env:",expand" envDefault:"${MX_HOST}"`
}

func init() {
	if err := env.Parse(&Core); err != nil {
		panic(fmt.Sprintf("could not parse core configuration: %v", err))
	}

	if err := env.Parse(&Mail); err != nil {
		panic(fmt.Sprintf("could not parse mail configuration: %v", err))
	}
}

var Core coreSettings
var Mail mailSettings
