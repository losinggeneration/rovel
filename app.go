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
//		_ = app.Post(func(ctx *rovel.UpdateCtx) {
//			model.items = append(model.items, result)
//			ctx.Invalidate(list.Rect())
//		})
//	}()
//
// For repaint-only work, PostInvalidate and PostInvalidateAll provide a small
// goroutine-safe convenience layer. Posted callbacks can also manipulate
// overlays through UpdateCtx:
//
//	go func() {
//		time.Sleep(time.Second)
//
//		_ = app.Post(func(ctx *rovel.UpdateCtx) {
//			ctx.ShowOverlay(OverlayOpts{
//				Root:  dialog,
//				Modal: true,
//				Place: overlay.Centered{},
//			})
//		})
//	}()
//
// Posted callbacks are serialized with input handling and run on the app loop.
// Multiple posted updates coalesce into bounded rendering work.
package rovel

import (
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/losinggeneration/rovel/action"
	"github.com/losinggeneration/rovel/backend"
	"github.com/losinggeneration/rovel/event"
	"github.com/losinggeneration/rovel/geom"
	"github.com/losinggeneration/rovel/internal/errbuf"
	"github.com/losinggeneration/rovel/style"
)

// App represents a TUI application.
type App struct {
	opts         AppOpts
	host         runtimeHost
	size         geom.Size // render region size (may be <= terminalSize)
	terminalSize geom.Size // physical terminal size

	renderer  runtimeRenderer
	presenter runtimePresenter
	frame     runtimeFrame

	errs *errbuf.ErrorBuffer

	root     View
	nodes    map[ID]*nodeEntry
	overlays OverlayManager

	layoutDirty bool

	invalidRects []geom.Rect

	focusedID     ID
	onFocusChange func(FocusChange)
	scopeMemory   map[ID]*scopeState

	eventCh chan Event

	// signalCh delivers terminal lifecycle signals (suspend/terminate) from
	// the backend. Nil when the backend does not catch lifecycle signals or
	// signal handling is disabled; a nil channel in the Run select never fires.
	signalCh <-chan backend.LifecycleSignal

	postMu    sync.Mutex
	postQueue []func(*UpdateCtx)

	wakeCh chan struct{}

	running atomic.Bool
	closed  bool
	closeMu sync.RWMutex
	done    chan struct{} // closed once when the app loop exits; unblocks readEvents' send

	mouse          mouseState
	lastRenderTime time.Time

	resolvedTheme Theme
	capability    style.Capability
	inputCaps     backend.InputCapabilities

	// ctx is the shared per-App dispatch context, built once (its closures and
	// host-derived fields never change) and reused across every event and paint
	// pass. mkCtx refreshes only the volatile fields. All access is on the app
	// loop goroutine, so no synchronization is needed.
	ctx *Ctx
}

// mouseState tracks press/drag/click state for mouse event enrichment.
type mouseState struct {
	pressButton event.MouseButton
	pressX      int
	pressY      int
	dragging    bool

	// grab is the hit-tested target of the current press. While non-nil,
	// drag and release events route to it directly (no re-hit-test), so a
	// drag keeps working when the pointer outruns the view — e.g. moving a
	// floating window. Cleared on release.
	grab View

	lastClickTime   time.Time
	lastClickX      int
	lastClickY      int
	lastClickButton event.MouseButton
	clickCount      int
}

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

// transportWriter adapts a backend.ANSITransport to io.Writer.
type transportWriter struct {
	t backend.ANSITransport
}

type actionHandler interface {
	HandleAction(act action.Action, ctx *Ctx) bool
}

func (w *transportWriter) Write(p []byte) (int, error) {
	return w.t.Write(p)
}

// New creates a new App with the given options.
func New(opts AppOpts) (*App, error) {
	size := geom.Size{W: 80, H: 24} // Default, will be updated on Enable

	app := &App{
		opts:        opts,
		size:        size,
		renderer:    newCellRenderer(size),
		nodes:       make(map[ID]*nodeEntry),
		scopeMemory: make(map[ID]*scopeState),
		eventCh:     make(chan Event, 256),
		postQueue:   make([]func(*UpdateCtx), 0, 64),
		wakeCh:      make(chan struct{}, 1),
		done:        make(chan struct{}),
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

// resolveRenderSize chooses the render region size. Explicit RenderSize wins.
// In cbreak mode, fall back to the root's PreferredSize (or MinSize if that
// is unavailable). Otherwise use the full terminal size. All results are
// clamped to the terminal.
func (a *App) resolveRenderSize(terminal geom.Size) geom.Size {
	clamp := func(s geom.Size) geom.Size {
		if s.W > terminal.W {
			s.W = terminal.W
		}
		if s.H > terminal.H {
			s.H = terminal.H
		}
		return s
	}

	if a.opts.RenderSize.W > 0 && a.opts.RenderSize.H > 0 {
		return clamp(a.opts.RenderSize)
	}

	if a.opts.TerminalMode == backend.ModeCBreak && a.root != nil {
		if ps, ok := a.root.(PreferredSizer); ok {
			pref := ps.PreferredSize()
			if pref.W > 0 && pref.H > 0 {
				return clamp(pref)
			}
		}

		min := a.root.MinSize()
		if min.W > 0 && min.H > 0 {
			return clamp(min)
		}
	}

	return terminal
}

// Enable enables the terminal and starts the application.
func (a *App) Enable() error {
	// Create backend if not provided
	if a.opts.Backend == nil {
		b, err := defaultBackend(a.errs, a.opts)
		if err != nil {
			return err
		}

		a.opts.Backend = b
	}

	a.host = newAppHost(a.opts.Backend)

	// Enable the backend
	terminalSize, err := a.host.Enable()
	if err != nil {
		return err
	}

	// The backend is now live (raw mode on a real terminal). Any later
	// failure must restore it, or the user's shell is left raw.
	unwind := func(err error) error {
		a.errs.Add(a.host.Restore())

		return err
	}

	// Capture the lifecycle-signal channel. Nil when the backend does not
	// catch lifecycle signals (custom backend or handling disabled); the
	// corresponding Run select arm then never fires.
	a.signalCh = a.host.Signals()

	a.terminalSize = terminalSize
	a.size = a.resolveRenderSize(terminalSize)

	// Detect or use provided capability
	providedCap := a.opts.Capability
	if providedCap == nil {
		detected := style.DetectCapabilityFromEnv()
		providedCap = &detected
	}

	a.capability = *providedCap

	// Query backend capabilities
	a.inputCaps = a.host.InputCapabilities()

	// Enable requested input features
	features := backend.InputFeatures{
		Mouse:          a.opts.Input.Mouse && a.inputCaps.Mouse,
		BracketedPaste: a.opts.Input.BracketedPaste && a.inputCaps.BracketedPaste,
		ModifiedKeys:   a.opts.Input.ModifiedKeys && a.inputCaps.ModifiedKeys,
	}
	if err := a.host.SetInputFeatures(features); err != nil {
		return unwind(err)
	}

	// Resolve theme for capability
	a.resolvedTheme = a.opts.Theme.Resolved(a.capability)

	// Build the shared dispatch context now that host capabilities are known.
	a.ctx = a.buildCtx()

	// Resize buffers to the render region size.
	a.resizeBuffers(a.size.W, a.size.H)

	// Choose the presenter for the concrete backend.
	presenter, err := presenterForBackend(a.opts.Backend, a.opts.TerminalMode, a.opts.ClearOnExit)
	if err != nil {
		return unwind(err)
	}

	a.presenter = presenter

	if err := a.presenter.InitScreen(a.size); err != nil {
		return unwind(err)
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

// Restore restores the terminal to its original state. It is safe to call
// even when Enable never ran or failed partway (the usual `defer
// app.Restore()` pattern); whatever was not set up is skipped.
func (a *App) Restore() error {
	if a.presenter != nil {
		if err := a.presenter.RestoreScreen(); err != nil {
			return err
		}
	}

	if a.host != nil {
		return a.host.Restore()
	}

	return nil
}

// Run starts the main event loop and blocks until the application quits.
func (a *App) Run() (err error) {
	defer func() {
		if r := recover(); r != nil {
			// Signal the forwarder before restoring so it can't leak on a
			// blocked send while the panic unwinds.
			a.setClosed()
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

		// wakeCh holds at most one signal, so the wakes for a backlog larger
		// than one drain batch may already be consumed. Re-arm the wake when
		// posts remain queued or the leftovers stall until the next event.
		a.postMu.Lock()
		morePosts := len(a.postQueue) > 0
		a.postMu.Unlock()

		if morePosts {
			a.wake()
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

		case sig := <-a.signalCh:
			switch sig {
			case backend.SignalSuspend:
				if err := a.suspend(); err != nil {
					a.setClosed()

					return err
				}
			case backend.SignalTerminate:
				// Convert termination into a graceful quit so the terminal is
				// restored through the caller's normal Restore path.
				a.running.Store(false)
			}

		case <-a.wakeCh:
			a.render()
			a.lastRenderTime = time.Now()
		}
	}

	a.setClosed()

	return nil
}

// suspend runs the orchestrated suspend/resume sequence on the app-loop
// goroutine (the sole terminal writer). It tears down the screen, hands off to
// the backend to restore cooked termios and stop the process, then — on resume
// — re-inits the screen at the current size and repaints. It is shared by the
// SignalSuspend select arm and the public App.Suspend.
//
// On any error the terminal is in an unknown state; the caller (Run) stops the
// loop and returns the error rather than continuing to render.
func (a *App) suspend() error {
	if a.host == nil || !a.host.Suspendable() {
		return ErrSuspendUnsupported
	}

	// Tear down the screen: show cursor, reset SGR, leave/clear region.
	if err := a.presenter.RestoreScreen(); err != nil {
		return err
	}

	// Restore cooked termios, stop the process, and on resume re-enter raw
	// mode and refresh the cached terminal size. Blocks while stopped.
	if err := a.host.Suspend(); err != nil {
		return err
	}

	// Re-derive the render region from the (possibly changed) terminal size.
	a.terminalSize = a.host.Size()
	a.size = a.resolveRenderSize(a.terminalSize)
	a.resizeBuffers(a.size.W, a.size.H)

	// The terminal was cooked and the shell may have scrolled; treat the front
	// buffer as stale so every cell repaints.
	a.renderer.ResetFrontBuffer()

	// Re-init the screen (new inline region in cbreak / clear in raw) and
	// force a full repaint.
	if err := a.presenter.InitScreen(a.size); err != nil {
		return err
	}

	a.presenter.ResetCursor()
	a.layout()
	a.InvalidateAll()
	a.render()

	return nil
}

// Suspend suspends the application: it restores the terminal to its pre-Enable
// state, stops the process with an uncatchable stop signal (the job-control
// equivalent of Ctrl+Z), and on resume (fg/SIGCONT) re-enters raw mode and
// repaints.
//
// It must be called from the app loop — an event handler (via ctx.Suspend) or a
// posted callback. In raw mode Ctrl+Z arrives as a key event rather than a
// signal, so apps wanting the conventional behavior bind a key to it:
//
//	if ev.Key == someSuspendKey {
//		_ = ctx.Suspend()
//	}
//
// Returns ErrSuspendUnsupported if the backend does not support suspension.
func (a *App) Suspend() error {
	return a.suspend()
}

// Invalidate marks a rect as needing repaint.
func (a *App) Invalidate(r geom.Rect) {
	a.invalidRects = append(a.invalidRects, r)
}

// InvalidateAll marks the entire screen as needing repaint.
func (a *App) InvalidateAll() {
	a.Invalidate(geom.Rect{X: 0, Y: 0, W: a.size.W, H: a.size.H})
}

// InvalidateLayout marks that a layout pass is needed. Layout is currently
// global (the whole tree is re-laid out on the next frame), so no subtree
// argument is taken.
func (a *App) InvalidateLayout() {
	a.layoutDirty = true
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
// Wakes the event loop to ensure prompt termination.
func (a *App) Quit() {
	a.closeMu.Lock()
	defer a.closeMu.Unlock()

	if a.closed {
		return
	}

	a.running.Store(false)

	// Wake the event loop to ensure it processes the quit request promptly.
	// Without this, the loop may block on channel receives if no events are pending.
	select {
	case a.wakeCh <- struct{}{}:
	default:
	}
}

// Post schedules fn to run on the app loop.
// It is safe to call from any goroutine, including before Run().
// Returns ErrClosed if the app is shutting down or closed.
func (a *App) Post(fn func(ctx *UpdateCtx)) error {
	if fn == nil {
		return nil
	}

	// Hold the read lock across both the closed check and the enqueue so a
	// concurrent setClosed cannot flip closed between them and strand a
	// callback that Post reported as accepted. setClosed takes the write lock,
	// so it serializes against this whole section.
	a.closeMu.RLock()

	if a.closed {
		a.closeMu.RUnlock()

		return ErrClosed
	}

	a.postMu.Lock()
	a.postQueue = append(a.postQueue, fn)
	a.postMu.Unlock()

	a.closeMu.RUnlock()

	a.wake()

	return nil
}

// PostInvalidate schedules a repaint of the given rect on the app loop.
// It is safe to call from any goroutine.
// This is a convenience wrapper that avoids the need to write:
//
//	_ = app.Post(func(ctx *rovel.UpdateCtx) { ctx.Invalidate(r) })
func (a *App) PostInvalidate(r geom.Rect) error {
	return a.Post(func(ctx *UpdateCtx) {
		ctx.Invalidate(r)
	})
}

// PostInvalidateAll schedules a full repaint of the screen on the app loop.
// It is safe to call from any goroutine.
// This is a convenience wrapper that avoids the need to write:
//
//	_ = app.Post(func(ctx *rovel.UpdateCtx) { ctx.InvalidateAll() })
func (a *App) PostInvalidateAll() error {
	return a.Post(func(ctx *UpdateCtx) {
		ctx.InvalidateAll()
	})
}

// Size returns the render region size. This equals the terminal size in the
// common full-screen case, but is smaller when a reduced render region is in
// use (AppOpts.RenderSize or cbreak mode); use TerminalSize for the physical
// terminal dimensions.
func (a *App) Size() geom.Size {
	return a.size
}

// TerminalSize returns the physical terminal size, which may be larger than
// Size when a smaller render region is in use (RenderSize or cbreak mode).
func (a *App) TerminalSize() geom.Size {
	return a.terminalSize
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

	if o.modal {
		a.focusFirstIn(o.root)
	}

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
	if o.modal {
		a.setRequestFocus(o.savedFocus)
	}
	a.Invalidate(o.rect)

	return o
}

// DismissOverlayByID removes a specific overlay by ID.
// Returns the dismissed overlay, or nil if not found.
// If the overlay is the top modal overlay, focus is restored.
// If the overlay is not at the top, focus is unchanged (dismissing a lower
// overlay should not affect focus that was moved after it was pushed).
func (a *App) DismissOverlayByID(id ID) *Overlay {
	wasTop := a.overlays.TopOverlay() != nil && a.overlays.TopOverlay().id == id

	o := a.overlays.PopOverlayByID(id)
	if o == nil {
		return nil
	}

	if o.onDismiss != nil {
		o.onDismiss()
	}

	// Clean up scope memory for the dismissed overlay.
	delete(a.scopeMemory, o.id)

	a.rebuildTree()
	a.updateBounds()

	// Only restore focus if this was the top modal overlay.
	// Dismissing a lower overlay should not steal focus from the current top.
	if wasTop && o.modal {
		a.setRequestFocus(o.savedFocus)
	}

	a.Invalidate(o.rect)

	return o
}

// RaiseOverlay moves the overlay with the given id to the top of the stack
// (below any modal overlay), preserving its identity and firing no dismiss
// side effects or focus changes. The raised region is invalidated for
// repaint. Returns the raised overlay, or nil if the id is unknown.
// Must be called from the app loop (or via App.Post).
func (a *App) RaiseOverlay(id ID) *Overlay {
	o := a.overlays.RaiseOverlay(id)
	if o == nil {
		return nil
	}

	a.Invalidate(o.rect)

	return o
}

// Focus sets the keyboard focus to the view with the given ID.
// Must be called from the app loop goroutine (e.g., via App.Post).
func (a *App) Focus(id ID) {
	a.setRequestFocus(id)
}

// FocusChange reports a committed focus transition. An ID of 0 means "no
// view focused".
type FocusChange struct {
	From ID
	To   ID
}

// SetFocusObserver registers fn to be invoked on every committed focus
// transition, including startup auto-focus, explicit App.Focus requests,
// overlay push/pop focus moves, and focus repair after the focused view is
// removed or disabled. fn runs on the app loop goroutine and must not block.
// Passing nil removes the observer. Call SetFocusObserver before Run or from
// the app loop (e.g., via App.Post); it is not safe to call concurrently with
// a running app loop.
func (a *App) SetFocusObserver(fn func(FocusChange)) {
	a.onFocusChange = fn
}

func (a *App) notifyFocusChange(from, to ID) {
	if a.onFocusChange != nil {
		a.onFocusChange(FocusChange{From: from, To: to})
	}
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

// doInitialPaint performs the initial paint of the screen.
func (a *App) doInitialPaint() {
	defer func() { a.frame = nil }()

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

	// Layout overlays before the tree rebuild and rect diff so overlay
	// geometry changes (e.g. a moved floating window) are captured by the
	// diff, invalidating both the old and the new rects.
	a.overlays.layoutOverlays(a.size)

	// Rebuild tree structure (handles dynamic children like Tabs).
	a.rebuildTree()

	// Update geometry in the nodes map.
	a.updateBounds()

	// Invalidate regions that changed structurally.
	a.invalidateLayoutDiff(oldNodes)

	// If the focused view disappeared or is no longer focusable, repair focus.
	a.ensureValidFocus()

	a.layoutDirty = false
}

func (a *App) setClosed() {
	a.closeMu.Lock()
	defer a.closeMu.Unlock()

	if a.closed {
		return
	}

	a.closed = true
	// Unblock readEvents if it is parked on a send into a no-longer-drained
	// eventCh, so the forwarding goroutine (and eventCh) can't leak after Run
	// returns.
	close(a.done)
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

		// Abort a blocked send once the app loop has exited: after Run returns
		// nothing drains eventCh, so an undrained send would park this goroutine
		// forever.
		select {
		case a.eventCh <- e:
		case <-a.done:
			return
		}
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

	ctx := a.mkCtx()
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
		if act, ok := a.opts.ResolveAction(e, focused); ok {
			// If overlays are present and Cancel, dismiss the topmost overlay.
			if a.overlays.HasOverlays() && act == action.Cancel {
				a.DismissOverlay()

				return
			}

			if a.dispatchAction(act, ctx) {
				return
			}
			// App-level fallback for focus navigation (scope-aware).
			switch act {
			case action.FocusNext:
				a.focusNextInScope()

				return
			case action.FocusPrev:
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
	return a.findViewInTreeSeen(v, id, nil)
}

func (a *App) findViewInTreeSeen(v View, id ID, seen map[ID]struct{}) View {
	vID := v.ID()
	if vID == id {
		return v
	}

	if vID != 0 {
		if _, ok := seen[vID]; ok {
			return nil
		}
		if seen == nil {
			seen = make(map[ID]struct{})
		}
		seen[vID] = struct{}{}
	}

	if c, ok := v.(viewChildren); ok {
		for _, child := range c.Children() {
			if found := a.findViewInTreeSeen(child, id, seen); found != nil {
				return found
			}
		}
	}

	return nil
}

// dispatchAction routes a semantic action through the view tree: focused view
// first, then ancestors up to the root. Returns true if any handler consumed it.
func (a *App) dispatchAction(act action.Action, ctx *Ctx) bool {
	// Try focused view first.
	focused := a.findFocusedView()
	if focused != nil {
		if ah, ok := focused.(actionHandler); ok {
			if ah.HandleAction(act, ctx) {
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
				if ah.HandleAction(act, ctx) {
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
			if ah.HandleAction(act, ctx) {
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
// A press implicitly grabs the hit-tested target: until release, drag and
// release events route to that target without re-hit-testing.
func (a *App) handleMouseEvent(e MouseEvent) {
	if a.root == nil {
		return
	}

	// Enrich the event with drag/click state.
	a.enrichMouseEvent(&e)

	ctx := a.mkCtx()

	// Drop a grab whose target left the mounted tree (e.g. its overlay was
	// dismissed mid-drag): routing to an unmounted view would move a window
	// nobody can see, and would pin its subtree in memory until the next
	// release — which may never arrive.
	if a.mouse.grab != nil {
		if _, mounted := a.nodes[a.mouse.grab.ID()]; !mounted {
			a.mouse.grab = nil
		}
	}

	// Implicit grab: drags and release go to the pressed target even when
	// the pointer has left its rect.
	if a.mouse.grab != nil && (e.Action == event.MouseDrag || e.Action == event.MouseRelease) {
		a.mouse.grab.Handle(e, ctx)

		if e.Action == event.MouseRelease {
			a.mouse.grab = nil
		}

		return
	}

	// Check overlays first
	if target, blocked := a.overlays.overlayHitTest(e.X, e.Y); target != nil {
		a.tryGrab(e, target)
		target.Handle(e, ctx)

		return
	} else if blocked {
		return // modal overlay blocked
	}

	if target := a.hitTest(a.root, e.X, e.Y); target != nil {
		a.tryGrab(e, target)
		target.Handle(e, ctx)
	}
}

// tryGrab records the implicit grab target for a press. Wheel buttons are
// excluded: they are delivered as presses but never get a release.
func (a *App) tryGrab(e MouseEvent, target View) {
	if e.Action != event.MousePress || e.Button == event.MouseButtonWheelUp || e.Button == event.MouseButtonWheelDown {
		return
	}

	a.mouse.grab = target
}

const doubleClickTimeout = 500 * time.Millisecond

// enrichMouseEvent updates the mouse state machine and sets ClickCount / MouseDrag.
func (a *App) enrichMouseEvent(e *MouseEvent) {
	switch e.Action {
	case event.MousePress:
		// Wheel buttons are delivered as MousePress but never get a
		// corresponding MouseRelease. Skip press-state tracking for them
		// so that subsequent MouseMove events are not promoted to drags
		// with a stale wheel button.
		if e.Button == event.MouseButtonWheelUp || e.Button == event.MouseButtonWheelDown {
			return
		}

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
	case event.MouseDrag:
	default:
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

	ctx := a.mkCtx()

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

	ctx := a.mkCtx()

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
	a.terminalSize = geom.Size{W: e.W, H: e.H}

	// Re-derive the render region from the new terminal size. For explicit
	// RenderSize / cbreak mode, the region may stay the same; for full-screen
	// (raw) mode it tracks the terminal.
	newSize := a.resolveRenderSize(a.terminalSize)

	// Resize buffers to the new render region.
	a.resizeBuffers(newSize.W, newSize.H)

	// Clear front buffer so every cell diffs as changed, forcing full redraw.
	// The terminal garbles content during resize (reflow), so the front buffer
	// no longer reflects what's actually on screen.
	a.renderer.ResetFrontBuffer()
	a.presenter.ResetCursor()

	// Full layout pass
	a.layout()

	// Mark entire render region as damaged
	a.Invalidate(geom.Rect{X: 0, Y: 0, W: newSize.W, H: newSize.H})
}

// buildCtx constructs the shared dispatch context. Its closures capture the
// App and its host-derived fields (Suspend/Clipboard/InputCaps) reflect
// capabilities fixed at Enable time, so it is built once and cached in a.ctx.
func (a *App) buildCtx() *Ctx {
	ctx := &Ctx{
		Invalidate:         func(r geom.Rect) { a.Invalidate(r) },
		InvalidateAll:      func() { a.InvalidateAll() },
		InvalidateLayout:   func() { a.InvalidateLayout() },
		RequestFocus:       func(id ID) { a.setRequestFocus(id) },
		Quit:               func() { a.Quit() },
		InputCaps:          a.inputCaps,
		ShowOverlay:        func(opts OverlayOpts) *Overlay { return a.ShowOverlay(opts) },
		RaiseOverlay:       func(id ID) *Overlay { return a.RaiseOverlay(id) },
		DismissOverlay:     func() *Overlay { return a.DismissOverlay() },
		DismissOverlayByID: func(id ID) *Overlay { return a.DismissOverlayByID(id) },
	}

	if a.host != nil && a.host.Suspendable() {
		ctx.Suspend = func() error { return a.Suspend() }
	}

	if a.host != nil {
		ctx.ClipboardWrite = func(s string) {
			if err := a.host.ClipboardWrite(s); err != nil {
				a.errs.Add(err)
			}
		}
	}

	if a.host != nil && a.inputCaps.ClipboardRead && a.host.CanClipboardReadAsync() {
		ctx.ClipboardRead = func() {
			_ = a.host.ClipboardReadRequest()
		}
	}

	return ctx
}

// mkCtx returns the shared dispatch context with its volatile fields refreshed
// for the current event or paint pass. The context is built once (see buildCtx)
// and reused, so no allocation happens on the hot path. Callers that dispatch a
// key event set Mod afterward; other paths leave it zero. Not safe to retain
// across dispatches — the next mkCtx call reuses the same struct.
func (a *App) mkCtx() *Ctx {
	if a.ctx == nil {
		a.ctx = a.buildCtx()
	}

	a.ctx.Theme = a.resolvedTheme
	a.ctx.Cap = a.capability
	a.ctx.FocusedID = a.focusedID
	a.ctx.Mod = 0

	return a.ctx
}

// mkUpdateCtx creates an update context for posted callbacks.
func (a *App) mkUpdateCtx() *UpdateCtx {
	return &UpdateCtx{
		Invalidate:         func(r geom.Rect) { a.Invalidate(r) },
		InvalidateAll:      func() { a.InvalidateAll() },
		InvalidateLayout:   func() { a.InvalidateLayout() },
		RequestFocus:       func(id ID) { a.setRequestFocus(id) },
		Quit:               func() { a.Quit() },
		ShowOverlay:        func(opts OverlayOpts) *Overlay { return a.ShowOverlay(opts) },
		RaiseOverlay:       func(id ID) *Overlay { return a.RaiseOverlay(id) },
		DismissOverlay:     func() *Overlay { return a.DismissOverlay() },
		DismissOverlayByID: func(id ID) *Overlay { return a.DismissOverlayByID(id) },
	}
}

// setRequestFocus requests focus for a view, using bounded invalidation when
// rects are known. Every committed transition is reported to the focus
// observer registered with SetFocusObserver.
func (a *App) setRequestFocus(id ID) {
	old := a.focusedID
	if old == id {
		return
	}
	a.commitFocus(old, id)
	a.notifyFocusChange(old, id)
}

// commitFocus applies a focus transition from old to id. The caller has
// already verified old != id.
func (a *App) commitFocus(old, id ID) {
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
	defer func() { a.frame = nil }()

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

// wake signals the event loop to wake up.
// It is internal machinery, not part of the public API.
func (a *App) wake() {
	select {
	case a.wakeCh <- struct{}{}:
	default:
	}
}

// focusFirstIn sets focus to the first focusable descendant of v.
func (a *App) focusFirstIn(v View) {
	seen := make(map[ID]struct{})
	var walk func(View) bool

	walk = func(v View) bool {
		id := v.ID()
		if id != 0 {
			if _, ok := seen[id]; ok {
				return false
			}
			seen[id] = struct{}{}
		}

		if f, ok := v.(viewFocusable); ok && f.Focusable() {
			a.setRequestFocus(v.ID())

			return true
		}

		if c, ok := v.(viewChildren); ok {
			if slices.ContainsFunc(c.Children(), walk) {
				return true
			}
		}

		return false
	}
	walk(v)
}
