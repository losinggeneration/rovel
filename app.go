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
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/losinggeneration/tui/backend"
	"github.com/losinggeneration/tui/event"
	"github.com/losinggeneration/tui/geom"
	"github.com/losinggeneration/tui/internal/errbuf"
	"github.com/losinggeneration/tui/style"
)

// App represents a TUI application.
type App struct {
	opts AppOpts
	host runtimeHost
	size geom.Size

	renderer  runtimeRenderer
	presenter runtimePresenter

	errs *errbuf.ErrorBuffer

	root     View
	nodes    map[ID]*nodeEntry
	overlays OverlayManager

	layoutDirty bool

	invalidRects []geom.Rect

	focusedID   ID
	scopeMemory map[ID]*scopeState

	eventCh chan Event

	postMu    sync.Mutex
	postQueue []func(*UpdateCtx)

	wakeCh chan struct{}

	running atomic.Bool
	closed  bool
	closeMu sync.RWMutex

	mouse          mouseState
	lastRenderTime time.Time

	resolvedTheme Theme
	capability    style.Capability
	inputCaps     backend.InputCapabilities
}

// mouseState tracks press/drag/click state for mouse event enrichment.
type mouseState struct {
	pressButton event.MouseButton
	pressX      int
	pressY      int
	dragging    bool

	lastClickTime   time.Time
	lastClickX      int
	lastClickY      int
	lastClickButton event.MouseButton
	clickCount      int
}

// Action constants mirror ui/action.go values to avoid an import cycle.
// These must stay in sync with the values in ui/action.go.
const (
	actionFocusNext = 1
	actionFocusPrev = 2
	actionCancel    = 6
)

const maxFrameInterval = 16 * time.Millisecond

type viewChildren interface {
	Children() []View
}

// mouseOpaque is a marker interface. Views that implement it stop hit-test
// recursion — the view receives all mouse events for its rect and is
// responsible for delegating to children itself (e.g. ScrollView).
type mouseOpaque interface {
	MouseOpaque()
}

type viewFocusable interface {
	Focusable() bool
}

// backendWriter adapts a backend.Backend to io.Writer.
type backendWriter struct {
	b backend.Backend
}

type actionHandler interface {
	HandleAction(act int, ctx *Ctx) bool
}

func (w *backendWriter) Write(p []byte) (int, error) {
	return w.b.Write(p)
}

// New creates a new App with the given options.
func New(opts AppOpts) (*App, error) {
	size := geom.Size{W: 80, H: 24} // Default, will be updated on Enable

	app := &App{
		opts:        opts,
		size:        size,
		renderer:    newCellRenderer(size),
		presenter:   newANSIPresenter(),
		nodes:       make(map[ID]*nodeEntry),
		scopeMemory: make(map[ID]*scopeState),
		eventCh:     make(chan Event, 256),
		postQueue:   make([]func(*UpdateCtx), 0, 64),
		wakeCh:      make(chan struct{}, 1),
		errs:        errbuf.New(50),
	}

	return app, nil
}

// SetRoot sets the root view of the application.
func (a *App) SetRoot(v View) {
	a.root = v
	a.rebuildTree()
	a.layoutDirty = true
}

// Enable enables the terminal and starts the application.
func (a *App) Enable() error {
	// Create backend if not provided
	if a.opts.Backend == nil {
		b, err := defaultBackend(a.errs)
		if err != nil {
			return err
		}

		a.opts.Backend = b
	}

	a.host = newAppHost(a.opts.Backend)

	// Enable the backend
	size, err := a.host.Enable()
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
	a.inputCaps = a.host.InputCapabilities()

	// Enable requested input features
	features := backend.InputFeatures{
		Mouse:          a.opts.Input.Mouse && a.inputCaps.Mouse,
		BracketedPaste: a.opts.Input.BracketedPaste && a.inputCaps.BracketedPaste,
	}
	if err := a.host.SetInputFeatures(features); err != nil {
		return err
	}

	// Resolve theme for capability
	a.resolvedTheme = a.opts.Theme.Resolved(a.capability)

	// Resize buffers to terminal size
	a.resizeBuffers(size.W, size.H)

	// Choose and attach the presenter for the concrete backend.
	a.presenter = presenterForBackend(a.host.Backend())
	a.presenter.Attach(a.host.Backend())
	if err := a.presenter.InitScreen(); err != nil {
		return err
	}

	// Initial layout
	a.layout()

	// Auto-focus the first focusable widget if nothing is focused yet.
	if a.focusedID == 0 && a.root != nil {
		a.focusFirstIn(a.root)
	}

	// Initial paint and flush
	a.doInitialPaint()

	return nil
}

// Restore restores the terminal to its original state.
func (a *App) Restore() error {
	if err := a.presenter.RestoreScreen(); err != nil {
		return err
	}

	if a.host != nil {
		return a.host.Restore()
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
	if a.beginFrame([]geom.Rect{{X: 0, Y: 0, W: a.size.W, H: a.size.H}}) {
		a.paintFramePass()
		a.presentFrame()
	}

	// Initial paint is a full-frame flush; discard any layout-driven invalidation
	// accumulated during startup so it doesn't trigger redundant repaints.
	a.invalidRects = a.invalidRects[:0]
}

// resizeBuffers resizes the render buffers when the terminal size changes.
func (a *App) resizeBuffers(w, h int) {
	a.renderer.Resize(w, h)
	a.size = geom.Size{W: w, H: h}
}

// invalidateLayoutDiff compares old and new node rects and invalidates regions
// that changed, were added, or were removed.
func (a *App) invalidateLayoutDiff(oldNodes map[ID]*nodeEntry) {
	// Removed views — present in old but not in new.
	for id, oldEntry := range oldNodes {
		if _, ok := a.nodes[id]; !ok {
			if !oldEntry.rect.Empty() {
				a.Invalidate(oldEntry.rect)
			}
		}
	}

	// Added / changed views.
	for id, newEntry := range a.nodes {
		oldEntry, okOld := oldNodes[id]
		if !okOld {
			if !newEntry.rect.Empty() {
				a.Invalidate(newEntry.rect)
			}

			continue
		}

		if oldEntry.rect != newEntry.rect {
			if !oldEntry.rect.Empty() {
				a.Invalidate(oldEntry.rect)
			}

			if !newEntry.rect.Empty() {
				a.Invalidate(newEntry.rect)
			}
		}
	}
}

// ensureValidFocus checks that the focused view is still mounted and focusable.
// If not, it repairs focus using scope-aware fallback.
func (a *App) ensureValidFocus() {
	a.ensureValidFocusScoped()
}

// layout performs a full layout pass from the root.
func (a *App) layout() {
	if a.root == nil {
		return
	}

	// Snapshot old node rects for diff-based invalidation.
	oldNodes := make(map[ID]*nodeEntry, len(a.nodes))

	for id, e := range a.nodes {
		snapshot := *e // copy by value
		oldNodes[id] = &snapshot
	}

	// Layout the root view to fill the entire screen.
	fullRect := geom.Rect{X: 0, Y: 0, W: a.size.W, H: a.size.H}
	a.root.Layout(fullRect)

	// Rebuild tree structure (handles dynamic children like Tabs).
	a.rebuildTree()

	// Update geometry in the nodes map.
	a.updateBounds()

	// Invalidate regions that changed structurally.
	a.invalidateLayoutDiff(oldNodes)

	// If the focused view disappeared or is no longer focusable, repair focus.
	a.ensureValidFocus()

	// Layout overlays after main tree.
	a.overlays.layoutOverlays(a.size)

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
		processedPosts := false

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
			processedPosts = true

			if !a.running.Load() {
				a.setClosed()

				return nil
			}
		}

		// Posted callbacks run on the app loop and may invalidate, relayout,
		// or change overlay/focus state. Render that work before blocking for
		// the next external wake/event so Post-driven updates are not delayed
		// by loop timing.
		if processedPosts {
			a.render()
			a.lastRenderTime = time.Now()
		}

		select {
		case e, ok := <-a.eventCh:
			if !ok {
				a.setClosed()

				return nil
			}

			for _, qe := range a.compactEventBatch(a.collectEventBatch(e)) {
				a.handleEvent(qe)

				if !a.running.Load() {
					a.setClosed()

					return nil
				}
			}

			if a.shouldSkipRender() {
				continue
			}

			a.render()
			a.lastRenderTime = time.Now()

		case <-a.wakeCh:
			a.render()
			a.lastRenderTime = time.Now()
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
		e := a.host.ReadEvent()
		if e == nil {
			// Backend shutdown/EOF.
			// Contract: Backend.ReadEvent() returns nil only on shutdown/EOF.
			return
		}

		a.eventCh <- e
	}
}

func (a *App) collectEventBatch(first Event) []Event {
	batch := []Event{first}

	for {
		select {
		case e, ok := <-a.eventCh:
			if !ok {
				return batch
			}

			batch = append(batch, e)
		default:
			return batch
		}
	}
}

func (a *App) compactEventBatch(batch []Event) []Event {
	out := make([]Event, 0, len(batch))

	for i := 0; i < len(batch); {
		if me, ok := batch[i].(MouseEvent); ok {
			switch {
			case me.Action == event.MouseMove || me.Action == event.MouseDrag:
				latest := me
				i++

				for i < len(batch) {
					next, ok := batch[i].(MouseEvent)
					if !ok || (next.Action != event.MouseMove && next.Action != event.MouseDrag) {
						break
					}

					latest = next
					i++
				}

				out = append(out, latest)

				continue
			case isWheelMouseEvent(me):
				net := signedWheelDelta(me)
				latest := me
				i++

				for i < len(batch) {
					next, ok := batch[i].(MouseEvent)
					if !ok || !isWheelMouseEvent(next) {
						break
					}

					latest = next
					net += signedWheelDelta(next)
					i++
				}

				if net != 0 {
					out = append(out, wheelEventFromNet(latest, net))
				}

				continue
			}
		}

		out = append(out, batch[i])
		i++
	}

	return out
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
	case ClipboardResponseEvent:
		a.handleClipboardResponseEvent(evt)
	}
}

func isWheelMouseEvent(me MouseEvent) bool {
	return me.Action == event.MousePress &&
		(me.Button == event.MouseButtonWheelUp || me.Button == event.MouseButtonWheelDown)
}

func signedWheelDelta(me MouseEvent) int {
	delta := me.WheelDelta
	if delta == 0 {
		delta = 1
	}

	if me.Button == event.MouseButtonWheelUp {
		return -delta
	}

	return delta
}

func wheelEventFromNet(base MouseEvent, net int) MouseEvent {
	if net < 0 {
		base.Button = event.MouseButtonWheelUp
		base.WheelDelta = -net
	} else {
		base.Button = event.MouseButtonWheelDown
		base.WheelDelta = net
	}

	return base
}

// handleKeyEvent processes a key event by dispatching through the root.
// If overlays are present, the topmost overlay gets first crack.
// If ResolveAction is set and resolves a semantic action, dispatch via
// HandleAction on the focused view first. Falls back to raw key dispatch.
func (a *App) handleKeyEvent(e KeyEvent) {
	if a.root == nil {
		return
	}

	ctx := a.mkCtx(a.root)
	ctx.Mod = e.Mod

	// Try semantic action resolution if configured.
	// Route the resolved action to the focused view first, then bubble up
	// through the view tree to the root so that container/root-level
	// ActionHandlers (e.g. quit) can catch unhandled actions.
	// Resolution works even with no focused view (nil is safe for the
	// resolver — it defaults to KeyCtxGlobal), enabling global actions
	// like quit to work before anything has focus.
	if a.opts.ResolveAction != nil {
		focused := a.findFocusedView() // walks live tree as fallback
		if action, ok := a.opts.ResolveAction(e, focused); ok {
			// If overlays are present and ActionCancel, dismiss the topmost overlay
			if a.overlays.HasOverlays() && action == actionCancel {
				a.DismissOverlay()

				return
			}

			if a.dispatchAction(action, ctx) {
				return
			}
			// App-level fallback for focus navigation (scope-aware).
			switch action {
			case actionFocusNext:
				a.focusNextInScope()

				return
			case actionFocusPrev:
				a.focusPrevInScope()

				return
			}
		}
	}

	// Route to topmost overlay if present
	if top := a.overlays.TopOverlay(); top != nil {
		if top.root.Handle(e, ctx) {
			return
		}
		// Modal overlay blocks propagation to main tree
		if top.modal {
			return
		}
	}

	// Fall back to raw key dispatch on main tree
	a.root.Handle(e, ctx)
}

// findFocusedView returns the currently focused view, or nil.
func (a *App) findFocusedView() View {
	if a.focusedID == 0 || a.root == nil {
		return nil
	}

	if entry, ok := a.nodes[a.focusedID]; ok {
		return entry.view
	}
	// Fallback: walk the live tree. This handles views that were added
	// dynamically (e.g. Tabs switching content) and not yet in nodes
	// (e.g. before next rebuildTree call).
	return a.findViewInTree(a.root, a.focusedID)
}

// findViewInTree walks the view tree looking for a view with the given ID.
func (a *App) findViewInTree(v View, id ID) View {
	if v.ID() == id {
		return v
	}

	if c, ok := v.(viewChildren); ok {
		for _, child := range c.Children() {
			if found := a.findViewInTree(child, id); found != nil {
				return found
			}
		}
	}

	return nil
}

// dispatchAction routes a semantic action through the view tree: focused view
// first, then ancestors up to the root. Returns true if any handler consumed it.
func (a *App) dispatchAction(action int, ctx *Ctx) bool {
	// Try focused view first.
	focused := a.findFocusedView()
	if focused != nil {
		if ah, ok := focused.(actionHandler); ok {
			if ah.HandleAction(action, ctx) {
				return true
			}
		}
	}

	// Walk ancestors of focused view toward root using parent pointers.
	// This is O(depth) rather than a full-tree DFS.
	if focused != nil {
		for _, ancestorID := range a.ancestorIDs(focused.ID()) {
			entry, ok := a.nodes[ancestorID]
			if !ok {
				continue
			}

			if ah, ok := entry.view.(actionHandler); ok {
				if ah.HandleAction(action, ctx) {
					return true
				}
			}
		}

		return false
	}

	// No focused view: fall back to full-tree DFS so global handlers (e.g.
	// quit) still fire before anything has focus.
	var walk func(v View) bool

	walk = func(v View) bool {
		if ah, ok := v.(actionHandler); ok {
			if ah.HandleAction(action, ctx) {
				return true
			}
		}

		if c, ok := v.(viewChildren); ok {
			if slices.ContainsFunc(c.Children(), walk) {
				return true
			}
		}

		return false
	}

	if a.root != nil {
		return walk(a.root)
	}

	return false
}

// handleMouseEvent dispatches a mouse event via hit-testing the view tree.
// Overlays are tested first (top to bottom); a modal overlay blocks the main tree.
// The deepest hit-tested view receives the event directly (flat dispatch).
func (a *App) handleMouseEvent(e MouseEvent) {
	if a.root == nil {
		return
	}

	// Enrich the event with drag/click state.
	a.enrichMouseEvent(&e)

	ctx := a.mkCtx(a.root)

	// Check overlays first
	if target, blocked := a.overlays.overlayHitTest(e.X, e.Y); target != nil {
		target.Handle(e, ctx)

		return
	} else if blocked {
		return // modal overlay blocked
	}

	if target := a.hitTest(a.root, e.X, e.Y); target != nil {
		target.Handle(e, ctx)
	}
}

const doubleClickTimeout = 500 * time.Millisecond

// enrichMouseEvent updates the mouse state machine and sets ClickCount / MouseDrag.
func (a *App) enrichMouseEvent(e *MouseEvent) {
	switch e.Action {
	case event.MousePress:
		a.mouse.pressButton = e.Button
		a.mouse.pressX = e.X
		a.mouse.pressY = e.Y
		a.mouse.dragging = false

		// Check for multi-click.
		now := time.Now()
		samePos := abs(e.X-a.mouse.lastClickX) <= 1 && abs(e.Y-a.mouse.lastClickY) <= 1
		sameButton := e.Button == a.mouse.lastClickButton
		withinTime := now.Sub(a.mouse.lastClickTime) < doubleClickTimeout

		if samePos && sameButton && withinTime {
			a.mouse.clickCount++
		} else {
			a.mouse.clickCount = 1
		}

		e.ClickCount = a.mouse.clickCount

	case event.MouseMove:
		// Promote to drag if a button is held.
		if a.mouse.pressButton != event.MouseButtonNone {
			a.mouse.dragging = true
			e.Action = event.MouseDrag
			e.Button = a.mouse.pressButton
		}

	case event.MouseRelease:
		if !a.mouse.dragging {
			// Record for multi-click tracking.
			a.mouse.lastClickTime = time.Now()
			a.mouse.lastClickX = e.X
			a.mouse.lastClickY = e.Y
			a.mouse.lastClickButton = e.Button
		}

		a.mouse.pressButton = event.MouseButtonNone
		a.mouse.dragging = false
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}

	return x
}

// hitTest returns the deepest view containing (x, y). Views implementing
// mouseOpaque stop recursion — they receive all mouse events for their rect
// and are responsible for delegating to children themselves.
// Uses clipRect from nodes for an early-out when the point is outside the
// clipped region.
func (a *App) hitTest(v View, x, y int) View {
	r := v.Rect()
	if x < r.X || x >= r.X+r.W || y < r.Y || y >= r.Y+r.H {
		return nil
	}

	// Clip-rect short-circuit: if the node's clipped region is known and the
	// point falls outside it, skip the entire subtree.
	if entry, ok := a.nodes[v.ID()]; ok {
		cr := entry.clipRect
		if x < cr.X || x >= cr.X+cr.W || y < cr.Y || y >= cr.Y+cr.H {
			return nil
		}
	}

	if _, ok := v.(mouseOpaque); ok {
		return v
	}

	if c, ok := v.(viewChildren); ok {
		children := c.Children()
		for i := len(children) - 1; i >= 0; i-- {
			if target := a.hitTest(children[i], x, y); target != nil {
				return target
			}
		}
	}

	return v
}

// handlePasteEvent dispatches a paste event to the focused view.
// If a modal overlay is present, it receives the event exclusively.
func (a *App) handlePasteEvent(e PasteEvent) {
	if a.root == nil {
		return
	}

	ctx := a.mkCtx(a.root)

	if top := a.overlays.TopOverlay(); top != nil {
		top.root.Handle(e, ctx)

		if top.modal {
			return
		}
	}

	// Dispatch directly to focused view — container Handle methods
	// typically only forward KeyEvents, not PasteEvents.
	focused := a.findFocusedView()
	if focused != nil {
		focused.Handle(e, ctx)

		return
	}

	a.root.Handle(e, ctx)
}

// handleClipboardResponseEvent dispatches a clipboard response to the focused view.
func (a *App) handleClipboardResponseEvent(e ClipboardResponseEvent) {
	if a.root == nil {
		return
	}

	ctx := a.mkCtx(a.root)

	if top := a.overlays.TopOverlay(); top != nil {
		top.root.Handle(e, ctx)

		if top.modal {
			return
		}
	}

	focused := a.findFocusedView()
	if focused != nil {
		if focused.Handle(e, ctx) {
			return
		}
	}

	a.root.Handle(e, ctx)
}

// handleResizeEvent processes a resize event.
func (a *App) handleResizeEvent(e ResizeEvent) {
	// Resize buffers
	a.resizeBuffers(e.W, e.H)

	// Clear front buffer so every cell diffs as changed, forcing full redraw.
	// The terminal garbles content during resize (reflow), so the front buffer
	// no longer reflects what's actually on screen.
	a.renderer.ResetFrontBuffer()
	a.presenter.ResetCursor()

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
	ctx := &Ctx{
		Theme:            a.resolvedTheme,
		Cap:              a.capability,
		Invalidate:       func(r geom.Rect) { a.Invalidate(r) },
		InvalidateAll:    func() { a.InvalidateAll() },
		InvalidateLayout: func(id ID) { a.InvalidateLayout(id) },
		RequestFocus:     func(id ID) { a.setRequestFocus(id) },
		Quit:             func() { a.Quit() },
		FocusedID:        a.focusedID,
		InputCaps:        a.inputCaps,
		ShowOverlay:      func(opts OverlayOpts) *Overlay { return a.ShowOverlay(opts) },
		DismissOverlay:   func() *Overlay { return a.DismissOverlay() },
	}
	if a.host != nil {
		ctx.ClipboardWrite = func(s string) {
			_ = a.host.ClipboardWrite(s)
		}
	}

	if a.host != nil && a.inputCaps.ClipboardRead && a.host.CanClipboardReadAsync() {
		ctx.ClipboardRead = func() {
			_ = a.host.ClipboardReadRequest()
		}
	}

	return ctx
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

	// Update scope memory for the new focus target.
	if id != 0 {
		if entry, ok := a.nodes[id]; ok {
			scope := entry.focusScopeID
			if a.scopeMemory[scope] == nil {
				a.scopeMemory[scope] = &scopeState{}
			}

			a.scopeMemory[scope].lastFocused = id
		}
	}

	// Helper: look up rect from nodes map.
	rectFor := func(nodeID ID) (geom.Rect, bool) {
		if entry, ok := a.nodes[nodeID]; ok {
			return entry.rect, true
		}

		return geom.Rect{}, false
	}

	// Clearing focus is a real state transition. Only the old focus needs to
	// be invalidated (if known). Do not fall back to full-screen invalidation.
	if id == 0 {
		if oldRect, ok := rectFor(old); ok && !oldRect.Empty() {
			a.Invalidate(oldRect)
		} else {
			a.InvalidateAll()
		}

		return
	}

	oldRect, okOld := rectFor(old)
	newRect, okNew := rectFor(id)

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

// shouldSkipRender returns true when more events are backlogged and we
// rendered recently enough that deferring this frame won't cause stutter.
// This lets the event loop drain backlogs at full speed instead of
// blocking on a render (and potentially vsync) between every small batch.
func (a *App) shouldSkipRender() bool {
	if len(a.eventCh) == 0 {
		return false
	}

	return time.Since(a.lastRenderTime) < maxFrameInterval
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
	frameRects := a.invalidRects
	a.invalidRects = a.invalidRects[:0]

	// If still no damage, nothing to do
	if !a.beginFrame(frameRects) {
		return
	}

	a.paintFramePass()

	// Views may call Invalidate during Paint (e.g. FocusRing detecting a
	// focus-state change). Process follow-up invalidations so the update
	// lands in the same frame. Cap iterations to guard against loops.
	for range 2 {
		if len(a.invalidRects) == 0 {
			break
		}

		a.renderer.AddDamageRects(a.invalidRects)
		a.invalidRects = a.invalidRects[:0]

		// Clear only the newly damaged spans, then repaint
		a.paintFramePass()
	}

	a.presentFrame()
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

// ShowOverlay pushes an overlay onto the stack. The overlay is laid out
// immediately and the screen is invalidated. If modal, focus is saved and
// moved to the first focusable view in the overlay subtree.
// Must be called from the app loop (or via App.Post).
func (a *App) ShowOverlay(opts OverlayOpts) *Overlay {
	o := a.overlays.PushOverlay(opts, a.focusedID)
	o.rect = o.place.Resolve(o.root, a.size)
	o.root.Layout(o.rect)
	a.rebuildTree()
	a.updateBounds()
	a.Invalidate(o.rect)

	// Focus first focusable in overlay
	a.focusFirstIn(o.root)

	return o
}

// DismissOverlay removes the topmost overlay. Restores saved focus if modal.
// Returns the dismissed overlay, or nil if no overlays exist.
func (a *App) DismissOverlay() *Overlay {
	o := a.overlays.PopOverlay()
	if o == nil {
		return nil
	}

	if o.onDismiss != nil {
		o.onDismiss()
	}
	// Clean up scope memory for the dismissed overlay's scope.
	delete(a.scopeMemory, o.id)
	a.rebuildTree()
	a.updateBounds()
	// Restore focus
	a.setRequestFocus(o.savedFocus)
	a.Invalidate(o.rect)

	return o
}

// DismissOverlayByID removes a specific overlay by ID.
func (a *App) DismissOverlayByID(id ID) *Overlay {
	o := a.overlays.PopOverlayByID(id)
	if o == nil {
		return nil
	}

	if o.onDismiss != nil {
		o.onDismiss()
	}

	a.rebuildTree()
	a.updateBounds()
	a.setRequestFocus(o.savedFocus)
	a.Invalidate(o.rect)

	return o
}

// focusFirstIn sets focus to the first focusable descendant of v.
func (a *App) focusFirstIn(v View) {
	type composite interface {
		Children() []View
	}

	var walk func(View) bool

	walk = func(v View) bool {
		if f, ok := v.(viewFocusable); ok && f.Focusable() {
			a.setRequestFocus(v.ID())

			return true
		}

		if c, ok := v.(composite); ok {
			for _, child := range c.Children() {
				if walk(child) {
					return true
				}
			}
		}

		return false
	}
	walk(v)
}

// Focus sets the keyboard focus to the view with the given ID.
// Must be called from the app loop goroutine (e.g., via App.Post).
func (a *App) Focus(id ID) {
	a.setRequestFocus(id)
}

// SetTheme changes the application theme at runtime.
// Must be called from the app loop goroutine (e.g., via App.Post).
// Triggers full repaint with new theme.
func (a *App) SetTheme(theme Theme) {
	a.opts.Theme = theme
	a.resolvedTheme = theme.Resolved(a.capability)
	a.presenter.ResetStyle() // Force re-emission of all SGR codes with new theme
	a.InvalidateAll()
}
