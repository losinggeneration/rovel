// Package headless provides a programmatic backend for testing TUI applications
// without a real terminal. It implements backend.Backend with an injected event
// queue and captured output for rendering assertions.
package headless

import (
	"sync"

	"github.com/losinggeneration/tui/backend"
	"github.com/losinggeneration/tui/event"
	"github.com/losinggeneration/tui/geom"
)

// Backend is a headless terminal backend for testing. It implements
// backend.Backend, backend.CapabilityReporter, and backend.InputFeatureEnabler.
type Backend struct {
	mu       sync.Mutex
	size     geom.Size
	eventCh  chan event.Event
	output   []byte
	enabled  bool
	restored bool
	caps     backend.InputCapabilities
	features backend.InputFeatures
}

var (
	_ backend.Backend             = (*Backend)(nil)
	_ backend.CapabilityReporter  = (*Backend)(nil)
	_ backend.InputFeatureEnabler = (*Backend)(nil)
)

// New creates a headless backend with the given initial terminal size.
func New(size geom.Size) *Backend {
	return &Backend{
		size:    size,
		eventCh: make(chan event.Event, 256),
		caps: backend.InputCapabilities{
			Mouse:          true,
			MouseMotion:    true,
			BracketedPaste: true,
		},
	}
}

// Enable enables the headless backend. Returns the configured size.
func (b *Backend) Enable() (geom.Size, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.enabled = true
	b.restored = false

	return b.size, nil
}

// Restore is a no-op that records the restore was called.
func (b *Backend) Restore() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.restored = true

	return nil
}

// Write captures raw output bytes.
func (b *Backend) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.output = append(b.output, p...)

	return len(p), nil
}

// Flush is a no-op for the headless backend.
func (b *Backend) Flush() error {
	return nil
}

// ReadEvent reads the next event from the injected event queue.
// Returns nil when the event channel is closed (signaling shutdown).
func (b *Backend) ReadEvent() event.Event {
	ev, ok := <-b.eventCh
	if !ok {
		return nil
	}

	return ev
}

// Size returns the current terminal size.
func (b *Backend) Size() geom.Size {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.size
}

// InputCapabilities reports that the headless backend supports common features.
func (b *Backend) InputCapabilities() backend.InputCapabilities {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.caps
}

// SetInputFeatures records the requested input features.
func (b *Backend) SetInputFeatures(f backend.InputFeatures) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.features = f

	return nil
}

// --- Test helpers ---

// SendEvent injects an event into the backend's event queue.
func (b *Backend) SendEvent(ev event.Event) {
	b.eventCh <- ev
}

// SendResize injects a resize event and updates the internal size.
func (b *Backend) SendResize(w, h int) {
	b.mu.Lock()
	b.size = geom.Size{W: w, H: h}
	b.mu.Unlock()

	b.eventCh <- event.ResizeEvent{W: w, H: h}
}

// Close closes the event channel, causing ReadEvent to return nil.
// This triggers graceful shutdown of the app's read loop.
func (b *Backend) Close() {
	close(b.eventCh)
}

// Output returns a copy of all captured output bytes.
func (b *Backend) Output() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()

	out := make([]byte, len(b.output))
	copy(out, b.output)

	return out
}

// ClearOutput discards all captured output bytes.
func (b *Backend) ClearOutput() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.output = b.output[:0]
}

// Restored reports whether Restore has been called.
func (b *Backend) Restored() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.restored
}

// Enabled reports whether Enable has been called.
func (b *Backend) Enabled() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.enabled
}

// Features returns the most recently set input features.
func (b *Backend) Features() backend.InputFeatures {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.features
}
