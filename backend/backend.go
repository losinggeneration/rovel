package backend

import (
	"github.com/losinggeneration/tui/event"
	"github.com/losinggeneration/tui/geom"
	"github.com/losinggeneration/tui/style"
)

// Backend is the host-facing interface for backends. It handles lifecycle,
// input, and size — but not rendering transport.
type Backend interface {
	// Enable enables the backend and returns the initial size.
	Enable() (geom.Size, error)

	// Restore restores the backend to its original state.
	Restore() error

	// ReadEvent reads and returns the next event, blocking until one is available.
	// It returns nil when the backend is shutting down or the input stream ends.
	ReadEvent() event.Event

	// Size returns the current size.
	Size() geom.Size
}

// ANSITransport is implemented by backends that support raw ANSI byte-stream
// output. The ANSI presenter uses this to write terminal escape sequences.
type ANSITransport interface {
	Write(p []byte) (int, error)
	Flush() error
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

// FrameCell represents a single logical cell in a presented frame.
type FrameCell struct {
	R        rune
	Style    style.Style
	Wide     bool
	WideCont bool
}

// CellFrame is a presented logical cell frame.
type CellFrame struct {
	W, H  int
	Cells []FrameCell
}

// CellFrameSink is implemented by backends that accept logical cell frames
// directly instead of terminal byte streams.
type CellFrameSink interface {
	PresentCellFrame(frame CellFrame) error
}
