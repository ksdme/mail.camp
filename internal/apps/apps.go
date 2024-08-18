package apps

import (
	"github.com/ksdme/mail/internal/apps/clipboard"
	"github.com/ksdme/mail/internal/apps/mail"
	"github.com/ksdme/mail/internal/config"
	"github.com/uptrace/bun"
)

func EnabledApps(db *bun.DB) []App {
	apps := []App{}

	if config.Core.MailAppEnabled {
		apps = append(apps, &mail.App{
			DB: db,
		})
	}

	if config.Core.ClipboardAppEnabled {
		apps = append(apps, &clipboard.App{
			DB: db,
		})
	}

	return apps
}
