package event

// PasteEvent represents a bracketed paste event.
type PasteEvent struct {
	Text string
}

func (PasteEvent) isEvent() {}
