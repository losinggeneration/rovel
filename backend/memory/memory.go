// Package memory provides a logical cell-frame backend for tests and
// non-terminal presentation experiments.
//
// This package is under active development and its API is not yet stable.
//
// # Unstable API
//
// Before v0.1.0, the API may change without notice. Use with caution.
package memory

import (
	"sync"

	"github.com/losinggeneration/tui/backend"
	"github.com/losinggeneration/tui/event"
	"github.com/losinggeneration/tui/geom"
)

// Backend is a non-terminal backend that captures logical cell frames.
type Backend struct {
	mu       sync.Mutex
	size     geom.Size
	eventCh  chan event.Event
	frames   []backend.CellFrame
	enabled  bool
	restored bool
	caps     backend.InputCapabilities
	features backend.InputFeatures
}

var (
	_ backend.Backend             = (*Backend)(nil)
	_ backend.CapabilityReporter  = (*Backend)(nil)
	_ backend.InputFeatureEnabler = (*Backend)(nil)
	_ backend.CellFrameSink       = (*Backend)(nil)
)

// New creates a memory backend with the given initial size.
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

func (b *Backend) Enable() (geom.Size, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.enabled = true
	b.restored = false

	return b.size, nil
}

func (b *Backend) Restore() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.restored = true

	return nil
}

func (b *Backend) ReadEvent() event.Event {
	ev, ok := <-b.eventCh
	if !ok {
		return nil
	}

	return ev
}

func (b *Backend) Size() geom.Size {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.size
}

func (b *Backend) InputCapabilities() backend.InputCapabilities {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.caps
}

func (b *Backend) SetInputFeatures(f backend.InputFeatures) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.features = f

	return nil
}

func (b *Backend) PresentCellFrame(frame backend.CellFrame) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	cloned := backend.CellFrame{
		W:     frame.W,
		H:     frame.H,
		Cells: make([]backend.FrameCell, len(frame.Cells)),
	}
	copy(cloned.Cells, frame.Cells)
	b.frames = append(b.frames, cloned)

	return nil
}

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
func (b *Backend) Close() {
	close(b.eventCh)
}

// FrameCount reports how many frames have been presented.
func (b *Backend) FrameCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()

	return len(b.frames)
}

// LastFrame returns a copy of the most recently presented frame.
func (b *Backend) LastFrame() (backend.CellFrame, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.frames) == 0 {
		return backend.CellFrame{}, false
	}

	frame := b.frames[len(b.frames)-1]
	cloned := backend.CellFrame{
		W:     frame.W,
		H:     frame.H,
		Cells: make([]backend.FrameCell, len(frame.Cells)),
	}
	copy(cloned.Cells, frame.Cells)

	return cloned, true
}

// Frames returns a copy of all presented frames.
func (b *Backend) Frames() []backend.CellFrame {
	b.mu.Lock()
	defer b.mu.Unlock()

	out := make([]backend.CellFrame, len(b.frames))
	for i, frame := range b.frames {
		out[i] = backend.CellFrame{
			W:     frame.W,
			H:     frame.H,
			Cells: make([]backend.FrameCell, len(frame.Cells)),
		}
		copy(out[i].Cells, frame.Cells)
	}

	return out
}

// Enabled reports whether Enable has been called.
func (b *Backend) Enabled() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.enabled
}

// Restored reports whether Restore has been called.
func (b *Backend) Restored() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.restored
}

// Features returns the most recently requested input features.
func (b *Backend) Features() backend.InputFeatures {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.features
}
