package backend

import (
	"github.com/losinggeneration/tui/event"
	"github.com/losinggeneration/tui/geom"
)

// Backend is the interface for terminal backends.
type Backend interface {
	// Enable enables the terminal and returns the initial size.
	Enable() (geom.Size, error)

	// Restore restores the terminal to its original state.
	Restore() error

	// Write writes raw ANSI output to the terminal.
	Write(p []byte) (int, error)

	// Flush flushes any buffered output.
	Flush() error

	// ReadEvent reads and returns the next event, blocking until one is available.
	ReadEvent() event.Event

	// Size returns the current terminal size.
	Size() geom.Size
}
