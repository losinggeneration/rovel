package event

// ClipboardResponseEvent is emitted when the terminal responds to an OSC 52 read query.
type ClipboardResponseEvent struct {
	Text string
}

func (ClipboardResponseEvent) isEvent() {}
