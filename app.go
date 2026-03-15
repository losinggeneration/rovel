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
	inputCaps     backend.InputCapabilities
}

type viewChildren interface {
	Children() []View
}

type viewFocusable interface {
	Focusable() bool
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
	if container, ok := v.(viewChildren); ok {
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

	// Query backend capabilities
	if cr, ok := a.backend.(backend.CapabilityReporter); ok {
		a.inputCaps = cr.InputCapabilities()
	}

	// Enable requested input features
	if fe, ok := a.backend.(backend.InputFeatureEnabler); ok {
		features := backend.InputFeatures{
			Mouse:          a.opts.Input.Mouse && a.inputCaps.Mouse,
			BracketedPaste: a.opts.Input.BracketedPaste && a.inputCaps.BracketedPaste,
		}
		if err := fe.SetInputFeatures(features); err != nil {
			return err
		}
	}

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

	// Initial paint is a full-frame flush; discard any layout-driven invalidation
	// accumulated during startup so it doesn't trigger redundant repaints.
	a.invalidRects = a.invalidRects[:0]
}

// resizeBuffers resizes the render buffers when the terminal size changes.
func (a *App) resizeBuffers(w, h int) {
	a.backBuf.Resize(w, h)
	a.frontBuf.Resize(w, h)
	a.damage.Reset(w, h)
	a.size = geom.Size{W: w, H: h}
}

type layoutInfo struct {
	rects          map[ID]geom.Rect
	focusedView    View
	firstFocusable View
}

// collectLayoutInfo walks the view tree and records rects (for bounded focus
// invalidation), and also finds (a) the currently focused view pointer (if it
// exists in the mounted tree) and (b) the first focusable view (for focus repair).
//
// Must be called after layout.
func (a *App) collectLayoutInfo() layoutInfo {
	if a.root == nil {
		return layoutInfo{rects: make(map[ID]geom.Rect)}
	}

	info := layoutInfo{
		rects: make(map[ID]geom.Rect, 64),
	}

	visited := make(map[ID]struct{}, 64)
	var walk func(v View)
	walk = func(v View) {
		id := v.ID()
		if _, ok := visited[id]; ok {
			return
		}
		visited[id] = struct{}{}

		info.rects[id] = v.Rect()

		if a.focusedID != 0 && id == a.focusedID {
			info.focusedView = v
		}

		if info.firstFocusable == nil {
			if f, ok := v.(viewFocusable); ok && f.Focusable() {
				info.firstFocusable = v
			}
		}

		if c, ok := v.(viewChildren); ok {
			for _, child := range c.Children() {
				walk(child)
			}
		}
	}

	walk(a.root)
	return info
}

func (a *App) invalidateLayoutDiff(oldRects, newRects map[ID]geom.Rect) {
	// Removed views.
	for id, oldR := range oldRects {
		if _, ok := newRects[id]; !ok {
			if !oldR.Empty() {
				a.Invalidate(oldR)
			}
		}
	}

	// Added / changed views.
	for id, newR := range newRects {
		oldR, okOld := oldRects[id]
		if !okOld {
			if !newR.Empty() {
				a.Invalidate(newR)
			}
			continue
		}
		if oldR != newR {
			if !oldR.Empty() {
				a.Invalidate(oldR)
			}
			if !newR.Empty() {
				a.Invalidate(newR)
			}
		}
	}
}

func (a *App) ensureValidFocus(info layoutInfo) {
	if a.focusedID == 0 {
		return
	}

	// Focused ID points to a mounted, focusable view => keep it.
	if info.focusedView != nil {
		if f, ok := info.focusedView.(viewFocusable); ok && f.Focusable() {
			return
		}
	}

	// Otherwise, repair focus to the first focusable view, or clear focus.
	if info.firstFocusable != nil {
		a.setRequestFocus(info.firstFocusable.ID())
	} else {
		a.setRequestFocus(0)
	}
}

// layout performs a full layout pass from the root.
func (a *App) layout() {
	if a.root == nil {
		return
	}

	oldRects := a.rectByID

	// Layout the root view to fill the entire screen
	fullRect := geom.Rect{X: 0, Y: 0, W: a.size.W, H: a.size.H}
	a.root.Layout(fullRect)

	// Update rect tracking and do conservative invalidation for structural
	// movement/removal/addition, per focus_input.md internal-metadata guidance.
	info := a.collectLayoutInfo()
	a.rectByID = info.rects
	a.invalidateLayoutDiff(oldRects, info.rects)

	// If the focused view disappeared or is no longer focusable, repair focus.
	a.ensureValidFocus(info)

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
	case MouseEvent:
		a.handleMouseEvent(evt)
	case PasteEvent:
		a.handlePasteEvent(evt)
	}
}

// handleKeyEvent processes a key event by dispatching through the root.
// If ResolveAction is set and resolves a semantic action, dispatch via
// HandleAction on the focused view first. Falls back to raw key dispatch.
func (a *App) handleKeyEvent(e KeyEvent) {
	if a.root == nil {
		return
	}

	ctx := a.mkCtx(a.root)

	// Try semantic action resolution if configured
	if a.opts.ResolveAction != nil {
		focused := a.findFocusedView()
		if focused != nil {
			if action, ok := a.opts.ResolveAction(e, focused); ok {
				// Check if focused view handles actions (structural interface)
				type actionHandler interface {
					HandleAction(act int, ctx *Ctx) bool
				}
				if ah, ok := focused.(actionHandler); ok {
					if ah.HandleAction(action, ctx) {
						return
					}
				}
			}
		}
	}

	// Fall back to raw key dispatch
	a.root.Handle(e, ctx)
}

// findFocusedView returns the currently focused view, or nil.
func (a *App) findFocusedView() View {
	if a.focusedID == 0 || a.root == nil {
		return nil
	}
	if v, ok := a.views[a.focusedID]; ok {
		return v
	}
	return nil
}

// handleMouseEvent dispatches a mouse event via hit-testing the view tree.
func (a *App) handleMouseEvent(e MouseEvent) {
	if a.root == nil {
		return
	}
	ctx := a.mkCtx(a.root)
	target := a.hitTest(a.root, e.X, e.Y)
	if target != nil {
		target.Handle(e, ctx)
	}
}

// hitTest walks the view tree to find the deepest view containing (x, y).
func (a *App) hitTest(v View, x, y int) View {
	r := v.Rect()
	if x < r.X || x >= r.X+r.W || y < r.Y || y >= r.Y+r.H {
		return nil
	}

	// Check children (deepest first)
	if c, ok := v.(viewChildren); ok {
		children := c.Children()
		// Iterate in reverse so later (top-most) children take priority
		for i := len(children) - 1; i >= 0; i-- {
			if hit := a.hitTest(children[i], x, y); hit != nil {
				return hit
			}
		}
	}

	return v
}

// handlePasteEvent dispatches a paste event to the focused view.
func (a *App) handlePasteEvent(e PasteEvent) {
	if a.root == nil {
		return
	}
	ctx := a.mkCtx(a.root)
	a.root.Handle(e, ctx)
}

// handleResizeEvent processes a resize event.
func (a *App) handleResizeEvent(e ResizeEvent) {
	// Resize buffers
	a.resizeBuffers(e.W, e.H)

	// Clear front buffer so every cell diffs as changed, forcing full redraw.
	// The terminal garbles content during resize (reflow), so the front buffer
	// no longer reflects what's actually on screen.
	a.frontBuf.Clear(render.Cell{})
	a.flusher.ResetCursor()

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
		InputCaps:        a.inputCaps,
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

	// Clearing focus is a real state transition. Only the old focus needs to
	// be invalidated (if known). Do not fall back to full-screen invalidation.
	if id == 0 {
		if oldRect, ok := a.rectByID[old]; ok && !oldRect.Empty() {
			a.Invalidate(oldRect)
		} else {
			a.InvalidateAll()
		}
		return
	}

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

	// Fallback for early startup / unknown new focus target.
	// If the new target's rect is unknown, we can't do bounded repaint safely.
	// If only the old rect is unknown, the old view is likely unmounted/stale
	// (layout invalidation handles it), so avoid a full-screen invalidate.
	if unknownNew {
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
