package backend

import (
	"github.com/losinggeneration/tui/event"
	"github.com/losinggeneration/tui/geom"
	"github.com/losinggeneration/tui/style"
)

// TerminalMode controls how the backend configures the terminal's input mode.
type TerminalMode uint8

const (
	// ModeRaw puts the terminal into full raw mode (default). Character-at-a-time
	// input, no echo, no signal keys. Required for full-featured TUI apps.
	ModeRaw TerminalMode = iota

	// ModeCBreak puts the terminal into cbreak mode: character-at-a-time
	// input with ECHO disabled, but ISIG stays on so Ctrl+C delivers SIGINT
	// and OPOST stays on so output is line-processed. Mouse tracking and
	// bracketed paste are unavailable; text remains selectable in the
	// terminal. The screen is not cleared on startup; output is drawn
	// inline at the current cursor position. Suitable for interactive CLI
	// tools — dialog boxes, prompts, script-driven TUI components — that
	// use the full rendering pipeline without taking over the entire
	// terminal.
	ModeCBreak
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

// LifecycleSignal identifies a terminal lifecycle signal surfaced to the app
// loop.
type LifecycleSignal uint8

const (
	// SignalSuspend indicates SIGTSTP: the app should suspend (restore the
	// terminal, stop the process, and repaint on resume).
	SignalSuspend LifecycleSignal = iota

	// SignalTerminate indicates SIGTERM/SIGHUP: the app should quit gracefully
	// so the terminal is restored through the normal path.
	SignalTerminate
)

// SignalController is implemented by backends that catch terminal lifecycle
// signals and can drive the process-suspend handshake. Backends that do not
// implement it get no built-in signal handling (the app loop leaves the
// terminal-lifecycle behavior to the caller).
type SignalController interface {
	// Signals delivers lifecycle notifications to the app loop. It returns nil
	// when signal handling is disabled, in which case the app loop performs no
	// automatic suspend/terminate handling.
	Signals() <-chan LifecycleSignal

	// Suspend restores cooked terminal state, stops the process (SIGTSTP), and
	// on resume re-establishes raw state and refreshes the cached terminal size
	// (the terminal is commonly resized while stopped; a pending SIGWINCH would
	// otherwise race the resume repaint). It blocks while the process is
	// stopped. The app loop calls it after saving screen state and re-inits the
	// screen when it returns.
	Suspend() error
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
