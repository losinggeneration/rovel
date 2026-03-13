// Package tui provides a retained-mode terminal UI toolkit.
//
// # Runtime Model
//
// The app loop goroutine owns UI state. View tree mutation is not generally
// goroutine-safe. Layout, focus changes, paint, and event handling all occur
// on the app loop.
//
// Background goroutines must not directly mutate UI-visible state. Instead,
// use App.Post to schedule updates on the app loop:
//
//	go func() {
//		result := fetchData()
//
//		_ = app.Post(func(ctx *tui.UpdateCtx) {
//			model.items = append(model.items, result)
//			ctx.Invalidate(list.Rect())
//		})
//	}()
//
// Posted callbacks are serialized with input handling and run on the app loop.
// Multiple posted updates coalesce into bounded rendering work.
package tui

import (
	"sync"
	"sync/atomic"

	"github.com/losinggeneration/tui/backend"
	errbuf "github.com/losinggeneration/tui/errors"
	"github.com/losinggeneration/tui/geom"
	"github.com/losinggeneration/tui/render"
	"github.com/losinggeneration/tui/style"
)

// App represents a TUI application.
type App struct {
	opts    AppOpts
	backend backend.Backend
	size    geom.Size

	backBuf  *render.Buffer
	frontBuf *render.Buffer
	damage   *render.Damage
	flusher  *render.ANSIFlusher

	errs *errbuf.ErrorBuffer

	root  View
	views map[ID]View

	rectByID map[ID]geom.Rect

	layoutDirty bool

	invalidRects []geom.Rect

	focusedID ID

	eventCh chan Event

	postMu    sync.Mutex
	postQueue []func(*UpdateCtx)

	wakeCh chan struct{}

	running atomic.Bool
	closed  bool
	closeMu sync.RWMutex

	backendWriter *backendWriter

	resolvedTheme Theme
	capability    style.Capability
}

// backendWriter adapts a backend.Backend to io.Writer.
type backendWriter struct {
	b backend.Backend
}

func (w *backendWriter) Write(p []byte) (int, error) {
	return w.b.Write(p)
}

// New creates a new App with the given options.
func New(opts AppOpts) (*App, error) {
	size := geom.Size{W: 80, H: 24} // Default, will be updated on Enable

	app := &App{
		opts:      opts,
		size:      size,
		backBuf:   render.NewBuffer(size.W, size.H),
		frontBuf:  render.NewBuffer(size.W, size.H),
		damage:    render.NewDamage(size.W, size.H),
		views:     make(map[ID]View),
		rectByID:  make(map[ID]geom.Rect),
		eventCh:   make(chan Event, 16),
		postQueue: make([]func(*UpdateCtx), 0, 64),
		wakeCh:    make(chan struct{}, 1),
		errs:      errbuf.New(50),
	}

	// Create flusher - will be set to backend writer on Enable
	app.flusher = render.NewANSIFlusher(nil)

	return app, nil
}

// SetRoot sets the root view of the application.
func (a *App) SetRoot(v View) {
	a.root = v
	a.addView(v)
	a.layoutDirty = true
}

// addView adds a view to the view registry.
func (a *App) addView(v View) {
	a.views[v.ID()] = v

	// Track rect for focus invalidation (will be updated during layout).
	if _, ok := a.rectByID[v.ID()]; !ok {
		a.rectByID[v.ID()] = geom.Rect{} // Empty until first layout
	}

	// Recursively add child views if this is a container
	if container, ok := v.(interface{ Children() []View }); ok {
		for _, child := range container.Children() {
			a.addView(child)
		}
	}
}

// Enable enables the terminal and starts the application.
func (a *App) Enable() error {
	// Create backend if not provided
	if a.opts.Backend == nil {
		b, err := defaultBackend()
		if err != nil {
			return err
		}
		a.opts.Backend = b
	}
	a.backend = a.opts.Backend

	// Enable the backend
	size, err := a.backend.Enable()
	if err != nil {
		return err
	}
	a.size = size

	// Detect or use provided capability
	cap := a.opts.Capability
	if cap == nil {
		detected := style.DetectCapabilityFromEnv()
		cap = &detected
	}
	a.capability = *cap

	// Resolve theme for capability
	a.resolvedTheme = a.opts.Theme.Resolved(a.capability)

	// Resize buffers to terminal size
	a.resizeBuffers(size.W, size.H)

	// Create backend writer adapter and flusher
	a.backendWriter = &backendWriter{b: a.backend}
	a.flusher = render.NewANSIFlusher(a.backendWriter)

	// Clear screen and hide cursor
	if err := a.flusher.ClearScreen(); err != nil {
		return err
	}
	if err := a.flusher.HideCursor(); err != nil {
		return err
	}
	if err := a.flusher.Flush(); err != nil {
		return err
	}

	// Initial layout
	a.layout()

	// Initial paint and flush
	a.doInitialPaint()

	return nil
}

// Restore restores the terminal to its original state.
func (a *App) Restore() error {
	// Show cursor before restoring
	if err := a.flusher.ShowCursor(); err != nil {
		return err
	}
	if err := a.flusher.Flush(); err != nil {
		return err
	}

	if a.backend != nil {
		return a.backend.Restore()
	}
	return nil
}

// doInitialPaint performs the initial paint of the screen.
func (a *App) doInitialPaint() {
	if a.root == nil {
		return
	}

	// For initial paint, ensure front buffer has zero cells (different from painted content)
	// Buffers are already zero-initialized, so we just need to make sure they're the right size.

	// Mark entire screen as dirty
	a.damage.Clear()
	a.damage.AddRect(geom.Rect{X: 0, Y: 0, W: a.size.W, H: a.size.H})

	// Clear damaged spans to theme base (Paint Contract A)
	a.clearDamagedSpans()

	// Paint the root view
	a.paintViews()

	// Flush everything
	a.flush()
}

// resizeBuffers resizes the render buffers when the terminal size changes.
func (a *App) resizeBuffers(w, h int) {
	a.backBuf.Resize(w, h)
	a.frontBuf.Resize(w, h)
	a.damage.Reset(w, h)
	a.size = geom.Size{W: w, H: h}
}

// updateRectTracking walks the view tree and records rects for bounded focus
// invalidation. Must be called after layout.
func (a *App) updateRectTracking() {
	if a.root == nil {
		return
	}
	visited := make(map[ID]struct{}, 64)
	a.updateRectTrackingView(a.root, visited)
}

// updateRectTrackingView recursively updates rect tracking for a view and its children.
func (a *App) updateRectTrackingView(v View, visited map[ID]struct{}) {
	id := v.ID()
	if _, ok := visited[id]; ok {
		return
	}
	visited[id] = struct{}{}

	a.rectByID[id] = v.Rect()

	// Recursively update children
	if c, ok := v.(interface{ Children() []View }); ok {
		for _, child := range c.Children() {
			a.updateRectTrackingView(child, visited)
		}
	}
}

// layout performs a full layout pass from the root.
func (a *App) layout() {
	if a.root == nil {
		return
	}

	// Layout the root view to fill the entire screen
	fullRect := geom.Rect{X: 0, Y: 0, W: a.size.W, H: a.size.H}
	a.root.Layout(fullRect)

	// Update rect tracking for bounded focus invalidation.
	a.updateRectTracking()

	a.layoutDirty = false
}

// Run starts the main event loop and blocks until the application quits.
func (a *App) Run() (err error) {
	defer func() {
		if r := recover(); r != nil {
			_ = a.Restore()
			panic(r)
		}
	}()

	a.closeMu.Lock()
	a.closed = false
	a.closeMu.Unlock()

	a.running.Store(true)

	go a.readEvents()

	for a.running.Load() {
		const maxPostsPerIteration = 64
		ctx := a.mkUpdateCtx()

		for range maxPostsPerIteration {
			a.postMu.Lock()
			if len(a.postQueue) == 0 {
				a.postMu.Unlock()
				break
			}
			fn := a.postQueue[0]
			a.postQueue = a.postQueue[1:]
			a.postMu.Unlock()

			fn(ctx)

			if !a.running.Load() {
				a.setClosed()
				return nil
			}
		}

		select {
		case e, ok := <-a.eventCh:
			if !ok {
				a.setClosed()
				return nil
			}
			a.handleEvent(e)

			if !a.running.Load() {
				a.setClosed()
				return nil
			}

			a.render()

		case <-a.wakeCh:
			a.render()
		}
	}

	a.setClosed()
	return nil
}

func (a *App) setClosed() {
	a.closeMu.Lock()
	a.closed = true
	a.closeMu.Unlock()
}

// readEvents reads events from the backend and sends them to the event channel.
func (a *App) readEvents() {
	defer close(a.eventCh)

	for a.running.Load() {
		e := a.backend.ReadEvent()
		if e == nil {
			// Backend shutdown/EOF.
			// Contract: Backend.ReadEvent() returns nil only on shutdown/EOF.
			return
		}
		a.eventCh <- e
	}
}

// handleEvent processes a single event.
func (a *App) handleEvent(e Event) {
	switch evt := e.(type) {
	case KeyEvent:
		a.handleKeyEvent(evt)
	case ResizeEvent:
		a.handleResizeEvent(evt)
	}
}

// handleKeyEvent processes a key event by dispatching through the root.
// Containers route to their focused descendant, then handle Tab cycling.
func (a *App) handleKeyEvent(e KeyEvent) {
	if a.root != nil {
		ctx := a.mkCtx(a.root)
		a.root.Handle(e, ctx)
	}
}

// handleResizeEvent processes a resize event.
func (a *App) handleResizeEvent(e ResizeEvent) {
	// Resize buffers
	a.resizeBuffers(e.W, e.H)

	// Full layout pass
	a.layout()

	// Mark entire screen as damaged
	a.Invalidate(geom.Rect{X: 0, Y: 0, W: e.W, H: e.H})
}

// Invalidate marks a rect as needing repaint.
func (a *App) Invalidate(r geom.Rect) {
	a.invalidRects = append(a.invalidRects, r)
}

// InvalidateAll marks the entire screen as needing repaint.
func (a *App) InvalidateAll() {
	a.Invalidate(geom.Rect{X: 0, Y: 0, W: a.size.W, H: a.size.H})
}

// InvalidateLayout marks that a layout pass is needed.
func (a *App) InvalidateLayout(id ID) {
	a.layoutDirty = true
}

// mkCtx creates a context for a view.
func (a *App) mkCtx(v View) *Ctx {
	return &Ctx{
		Theme:            a.resolvedTheme,
		Cap:              a.capability,
		Invalidate:       func(r geom.Rect) { a.Invalidate(r) },
		InvalidateAll:    func() { a.InvalidateAll() },
		InvalidateLayout: func(id ID) { a.InvalidateLayout(id) },
		RequestFocus:     func(id ID) { a.setRequestFocus(id) },
		Quit:             func() { a.Quit() },
		FocusedID:        a.focusedID,
	}
}

// mkUpdateCtx creates an update context for posted callbacks.
func (a *App) mkUpdateCtx() *UpdateCtx {
	return &UpdateCtx{
		Invalidate:       func(r geom.Rect) { a.Invalidate(r) },
		InvalidateAll:    func() { a.InvalidateAll() },
		InvalidateLayout: func(id ID) { a.InvalidateLayout(id) },
		RequestFocus:     func(id ID) { a.setRequestFocus(id) },
		Quit:             func() { a.Quit() },
	}
}

// setRequestFocus requests focus for a view, using bounded invalidation when
// rects are known.
func (a *App) setRequestFocus(id ID) {
	old := a.focusedID
	if old == id {
		return
	}

	a.focusedID = id

	oldRect, okOld := a.rectByID[old]
	newRect, okNew := a.rectByID[id]

	// Treat empty rect as unknown (handles pre-layout focus requests)
	unknownOld := !okOld || oldRect.Empty()
	unknownNew := !okNew || newRect.Empty()

	if !unknownOld {
		a.Invalidate(oldRect)
	}
	if !unknownNew {
		a.Invalidate(newRect)
	}

	// Fallback for early startup / unknown IDs (should be rare).
	if unknownOld || unknownNew {
		a.InvalidateAll()
	}
}

// render performs a single render frame.
func (a *App) render() {
	// If no damage and no layout needed, nothing to do
	if len(a.invalidRects) == 0 && !a.layoutDirty {
		return
	}

	// If layout is dirty, do a layout pass
	if a.layoutDirty {
		a.layout()
		// Layout may move things, so invalidate everything
		a.InvalidateAll()
	}

	// Coalesce invalidations into damage
	a.damage.Clear()
	for _, r := range a.invalidRects {
		a.damage.AddRect(r)
	}
	a.invalidRects = a.invalidRects[:0]

	// If still no damage, nothing to do
	if a.damage.IsEmpty() {
		return
	}

	// Paint Contract A: Clear damaged spans to theme base
	a.clearDamagedSpans()

	// Paint intersecting views
	a.paintViews()

	// Diff and flush
	a.flush()
}

// clearDamagedSpans implements Paint Contract A by clearing damaged regions
// to the theme's base style before painting.
func (a *App) clearDamagedSpans() {
	baseCell := render.Cell{
		R:     ' ',
		Style: a.resolvedTheme.Base,
		Wide:  false,
	}

	for y := 0; y < a.damage.H; y++ {
		spans := a.damage.Rows[y]
		for _, sp := range spans {
			for x := sp.X0; x < sp.X1; x++ {
				cell := a.backBuf.At(x, y)
				*cell = baseCell
			}
		}
	}
}

// paintViews paints all views that intersect with damaged regions.
func (a *App) paintViews() {
	if a.root == nil {
		return
	}

	ctx := a.mkCtx(a.root)

	// Create a painter for each damaged region
	// For now, we'll create one painter with full clip and let views paint
	// The clipping will happen in the render.Painter
	clip := geom.Rect{X: 0, Y: 0, W: a.size.W, H: a.size.H}
	rp := render.NewPainter(a.backBuf, clip, a.resolvedTheme.Base)
	p := NewPainter(rp, a.resolvedTheme.Base)

	// Paint the root (which will paint its children)
	a.root.Paint(p, ctx)
}

// flush diffs the buffers and flushes changes to the terminal.
func (a *App) flush() {
	runs := render.DiffRuns(a.backBuf, a.frontBuf, a.damage)
	if len(runs) > 0 {
		a.errs.Add(a.flusher.FlushRuns(a.backBuf, a.frontBuf, runs))
	}
	a.errs.Add(a.backend.Flush())
}

// Errors returns the accumulated errors from terminal operations.
// The returned slice is a copy and is safe to modify.
// Errors are capped at 50 by default; older errors are discarded.
func (a *App) Errors() []error {
	return a.errs.Get()
}

// Quit requests the application to stop.
// It is safe to call from any goroutine.
// Repeated calls are harmless and idempotent.
func (a *App) Quit() {
	a.closeMu.Lock()
	defer a.closeMu.Unlock()

	if a.closed {
		return
	}

	a.running.Store(false)
}

// Post schedules fn to run on the app loop.
// It is safe to call from any goroutine, including before Run().
// Returns ErrClosed if the app is shutting down or closed.
func (a *App) Post(fn func(ctx *UpdateCtx)) error {
	if fn == nil {
		return nil
	}

	a.closeMu.RLock()
	if a.closed {
		a.closeMu.RUnlock()
		return ErrClosed
	}
	a.closeMu.RUnlock()

	a.postMu.Lock()
	a.postQueue = append(a.postQueue, fn)
	a.postMu.Unlock()

	a.wake()
	return nil
}

// wake signals the event loop to wake up.
// It is internal machinery, not part of the public API.
func (a *App) wake() {
	select {
	case a.wakeCh <- struct{}{}:
	default:
	}
}

// Size returns the current terminal size.
func (a *App) Size() geom.Size {
	return a.size
}

// SetTheme changes the application theme at runtime.
// Must be called from the app loop goroutine (e.g., via App.Post).
// Triggers full repaint with new theme.
func (a *App) SetTheme(theme Theme) {
	a.opts.Theme = theme
	a.resolvedTheme = theme.Resolved(a.capability)
	a.flusher.ResetStyle() // Force re-emission of all SGR codes with new theme
	a.InvalidateAll()
}
