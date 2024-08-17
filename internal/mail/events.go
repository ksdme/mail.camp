package mail

import "github.com/ksdme/mail/internal/utils"

var (
	MailboxContentsUpdatedSignal = utils.NewBroadcastBus[int64]()
)
