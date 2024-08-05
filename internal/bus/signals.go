package bus

var (
	MailboxContentsUpdatedSignal = NewSignalBus[int64]()
)
