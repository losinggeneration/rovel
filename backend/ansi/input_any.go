package ansi

import "github.com/losinggeneration/rovel/event"

// DecodeInput decodes a chunked terminal input byte stream into normalized
// rovel events. It is a small public composition hook for non-terminal
// backends and tests that already own their input stream but should not
// duplicate ANSI keyboard/paste parsing.
type DecodeInput struct {
	decoder InputDecoder
}

// Reset clears any pending partial input sequence.
func (d *DecodeInput) Reset() {
	d.decoder.Reset()
}

// Push processes a byte slice and appends any complete events to dst.
func (d *DecodeInput) Push(dst []event.Event, data []byte) []event.Event {
	for _, b := range data {
		dst = d.decoder.PushByte(dst, b)
	}

	return dst
}

// FlushPending emits events that are safe to resolve at an input boundary.
func (d *DecodeInput) FlushPending(dst []event.Event) []event.Event {
	return d.decoder.FlushPending(dst)
}

// Finalize emits any remaining incomplete state at end-of-stream.
func (d *DecodeInput) Finalize(dst []event.Event) []event.Event {
	return d.decoder.Finalize(dst)
}
