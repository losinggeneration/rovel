package sdl

import (
	"sync"

	"github.com/losinggeneration/tui/backend"
	"github.com/losinggeneration/tui/backend/cellsurface"
	"github.com/losinggeneration/tui/event"
	"github.com/losinggeneration/tui/geom"
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

var (
	_ backend.Backend             = (*Core)(nil)
	_ backend.CapabilityReporter  = (*Core)(nil)
	_ backend.InputFeatureEnabler = (*Core)(nil)
	_ backend.CellFrameSink       = (*Core)(nil)
)

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

// Write is a no-op because SDL presentation consumes logical frames, not ANSI.
func (c *Core) Write(p []byte) (int, error) {
	return len(p), nil
}

func (c *Core) Flush() error {
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

func max(a, b int) int {
	if a > b {
		return a
	}

	return b
}
