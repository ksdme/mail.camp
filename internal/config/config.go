package config

import (
	"github.com/caarlos0/env/v11"
)

type settings struct {
	Debug bool `envDefault:"true"`

	DBURI     string `env:"DB_URI" envDefault:"file:db.sqlite3"`
	DBMigrate bool   `env:"DB_MIGRATE"`

	MXHost       string `env:"MX_HOST" envDefault:"localhost"`
	SMTPBindAddr string `env:"SMTP_BIND_ADDR" envDefault:"127.0.0.1:1025"`

	SSHBindAddr           string `env:"SSH_BIND_ADDR" envDefault:"127.0.0.1:2222"`
	SSHHostKeyPath        string `env:"SSH_HOST_KEY_PATH,expand" envDefault:"${HOME}/.ssh/id_rsa"`
	SSHAuthorizedKeysPath string `env:"SSH_AUTHORIZED_KEYS_PATH"`

	Signature string `env:",expand" envDefault:"${MX_HOST} by @ksdme"`
}

func init() {
	if err := env.Parse(&Settings); err != nil {
		panic(err)
	}
}

var Settings settings
