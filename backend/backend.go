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
	// It returns nil when the backend is shutting down or the input stream ends.
	ReadEvent() event.Event

	// Size returns the current terminal size.
	Size() geom.Size
}

// InputCapabilities describes what input features the backend supports.
type InputCapabilities struct {
	Mouse          bool
	MouseMotion    bool
	BracketedPaste bool
	ClipboardWrite bool
	ClipboardRead  bool
}

// CapabilityReporter is implemented by backends that can report their capabilities.
type CapabilityReporter interface {
	InputCapabilities() InputCapabilities
}

// InputFeatures describes which input features to enable.
type InputFeatures struct {
	Mouse          bool
	MouseMotion    bool
	BracketedPaste bool
}

// InputFeatureEnabler is implemented by backends that can enable/disable input features.
type InputFeatureEnabler interface {
	SetInputFeatures(f InputFeatures) error
}

// ClipboardBackend is implemented by backends that support clipboard operations.
type ClipboardBackend interface {
	ClipboardWrite(text string) error
	ClipboardRead() (string, error)
}

// ClipboardAsyncReader is implemented by backends that support async clipboard
// read via terminal query/response (e.g., OSC 52). The response arrives
// asynchronously as a ClipboardResponseEvent from ReadEvent.
type ClipboardAsyncReader interface {
	ClipboardReadRequest() error
}
