// Package sdl will provide a windowed cell-surface backend built on SDL.
//
// The intended implementation model is:
//
//   - SDL owns window lifecycle and event polling
//   - tui still owns the retained widget/runtime model
//   - logical cell frames are consumed through backend.CellFrameSink
//   - backend/cellsurface provides the frame traversal and cell-to-pixel helpers
//
// If this backend is implemented, prefer github.com/veandco/go-sdl2 for now.
// The SDL3 Go bindings are still experimental and should not be the default
// target yet.
//
// This package is under active development and its API is not yet stable.
//
// # Unstable API
//
// Before v1.0.0, the API may change without notice. Use with caution.
package sdl

import (
	"sync"

	"github.com/losinggeneration/rovel/backend"
	"github.com/losinggeneration/rovel/backend/cellsurface"
	"github.com/losinggeneration/rovel/event"
	"github.com/losinggeneration/rovel/geom"
	"github.com/losinggeneration/rovel/style"
)

// Core is the pure-Go state and mapping layer for the future SDL backend.
//
// It intentionally contains no SDL dependency. The concrete go-sdl2 binding can
// later use this type for:
//   - lifecycle/backend state
//   - logical frame storage
//   - cell metrics
//   - pixel-to-cell event mapping
type Core struct {
	mu       sync.Mutex
	opts     Options
	metrics  cellsurface.Metrics
	size     geom.Size
	eventCh  chan event.Event
	frame    backend.CellFrame
	enabled  bool
	restored bool
	features backend.InputFeatures
}

// Options configures the future SDL backend.
type Options struct {
	Title string

	// Window size in pixels.
	WindowWidth  int
	WindowHeight int

	// Logical cell metrics in pixels.
	CellWidth  int
	CellHeight int

	// Font configuration for text rendering.
	FontPath string
	FontSize int

	// Surface default colors used when frame styles leave fg/bg as default.
	DefaultFG style.RGBA
	DefaultBG style.RGBA
}

// DefaultOptions returns a conservative default SDL configuration suitable for
// an initial cell-surface spike.
func DefaultOptions() Options {
	return Options{
		Title:        "tui",
		WindowWidth:  960,
		WindowHeight: 640,
		CellWidth:    8,
		CellHeight:   16,
		FontSize:     16,
		DefaultFG:    style.RGBA{R: 229, G: 229, B: 229, A: 0xFF},
		DefaultBG:    style.RGBA{R: 0, G: 0, B: 0, A: 0xFF},
	}
}

// NewCore creates the dependency-free SDL backend core.
func NewCore(opts Options) *Core {
	if opts.CellWidth <= 0 || opts.CellHeight <= 0 {
		def := DefaultOptions()
		if opts.CellWidth <= 0 {
			opts.CellWidth = def.CellWidth
		}

		if opts.CellHeight <= 0 {
			opts.CellHeight = def.CellHeight
		}
	}

	if opts.WindowWidth <= 0 || opts.WindowHeight <= 0 {
		def := DefaultOptions()
		if opts.WindowWidth <= 0 {
			opts.WindowWidth = def.WindowWidth
		}

		if opts.WindowHeight <= 0 {
			opts.WindowHeight = def.WindowHeight
		}
	}

	return &Core{
		opts:    opts,
		metrics: cellsurface.Metrics{CellWidth: opts.CellWidth, CellHeight: opts.CellHeight},
		size: geom.Size{
			W: max(1, opts.WindowWidth/opts.CellWidth),
			H: max(1, opts.WindowHeight/opts.CellHeight),
		},
		eventCh: make(chan event.Event, 256),
	}
}

func (c *Core) Enable() (geom.Size, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.enabled = true
	c.restored = false

	return c.size, nil
}

func (c *Core) Restore() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.restored = true

	return nil
}

func (c *Core) ReadEvent() event.Event {
	ev, ok := <-c.eventCh
	if !ok {
		return nil
	}

	return ev
}

func (c *Core) Size() geom.Size {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.size
}

func (c *Core) InputCapabilities() backend.InputCapabilities {
	return backend.InputCapabilities{
		Mouse:          true,
		MouseMotion:    true,
		BracketedPaste: true,
		ClipboardWrite: true,
		ClipboardRead:  true,
	}
}

func (c *Core) SetInputFeatures(f backend.InputFeatures) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.features = f

	return nil
}

func (c *Core) PresentCellFrame(frame backend.CellFrame) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	cloned := backend.CellFrame{
		W:     frame.W,
		H:     frame.H,
		Cells: make([]backend.FrameCell, len(frame.Cells)),
	}
	copy(cloned.Cells, frame.Cells)
	c.frame = cloned

	return nil
}

// Metrics returns the configured cell-surface metrics.
func (c *Core) Metrics() cellsurface.Metrics {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.metrics
}

// LastFrame returns a copy of the most recently presented logical frame.
func (c *Core) LastFrame() (backend.CellFrame, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.frame.W == 0 || c.frame.H == 0 {
		return backend.CellFrame{}, false
	}

	out := backend.CellFrame{
		W:     c.frame.W,
		H:     c.frame.H,
		Cells: make([]backend.FrameCell, len(c.frame.Cells)),
	}
	copy(out.Cells, c.frame.Cells)

	return out, true
}

// SendEvent injects an event into the core event queue.
func (c *Core) SendEvent(ev event.Event) {
	c.eventCh <- ev
}

// Close closes the event queue.
func (c *Core) Close() {
	close(c.eventCh)
}

// ResizeWindow updates the logical size from a pixel-sized window and injects a
// resize event in cell coordinates.
func (c *Core) ResizeWindow(pixelW, pixelH int) event.ResizeEvent {
	c.mu.Lock()
	c.size = geom.Size{
		W: max(1, pixelW/c.metrics.CellWidth),
		H: max(1, pixelH/c.metrics.CellHeight),
	}
	ev := event.ResizeEvent{W: c.size.W, H: c.size.H}
	c.mu.Unlock()

	c.eventCh <- ev

	return ev
}

// Refresh requests a full redraw by re-emitting the current logical size as a
// resize event. This lets the runtime reuse its existing full-redraw path for
// surface re-expose/focus-regain situations.
func (c *Core) Refresh() event.ResizeEvent {
	c.mu.Lock()
	ev := event.ResizeEvent{W: c.size.W, H: c.size.H}
	c.mu.Unlock()

	c.eventCh <- ev

	return ev
}

// MapMouse maps a pixel-space mouse position into a logical cell mouse event.
func (c *Core) MapMouse(pixelX, pixelY int, button event.MouseButton, action event.MouseAction, mod event.ModMask) event.MouseEvent {
	c.mu.Lock()
	metrics := c.metrics
	c.mu.Unlock()

	x, y := metrics.PixelToCell(pixelX, pixelY)

	return event.MouseEvent{
		X:      x,
		Y:      y,
		Button: button,
		Action: action,
		Mod:    mod,
	}
}

// Options returns the current SDL options.
func (c *Core) Options() Options {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.opts
}
