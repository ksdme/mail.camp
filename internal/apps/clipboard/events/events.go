package events

import "github.com/ksdme/mail/internal/utils"

var (
	ClipboardContentsUpdatedSignal = utils.NewBroadcastBus[int64, int64]()
)
