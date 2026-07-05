package tui

import (
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/losinggeneration/tui/backend/headless"
	"github.com/losinggeneration/tui/event"
	"github.com/losinggeneration/tui/geom"
)

func TestPost_ErrClosed(t *testing.T) {
	app, _ := New(AppOpts{})
	app.setClosed()

	err := app.Post(func(ctx *UpdateCtx) {})
	if !errors.Is(err, ErrClosed) {
		t.Errorf("Post after closed: got %v, want ErrClosed", err)
	}
}

func TestPost_FromGoroutine(t *testing.T) {
	app, _ := New(AppOpts{})

	var (
		wg     sync.WaitGroup
		posted atomic.Bool
	)

	wg.Add(1)

	go func() {
		defer wg.Done()

		err := app.Post(func(ctx *UpdateCtx) {
			posted.Store(true)
		})
		if err != nil {
			t.Errorf("Post failed: %v", err)
		}
	}()

	wg.Wait()

	app.postMu.Lock()
	queued := len(app.postQueue)
	app.postMu.Unlock()

	if queued != 1 {
		t.Errorf("Expected 1 queued post, got %d", queued)
	}
}

func TestPost_OrderingFIFO(t *testing.T) {
	app, _ := New(AppOpts{})

	var (
		order []int
		mu    sync.Mutex
	)

	for i := range 5 {
		if err := app.Post(func(ctx *UpdateCtx) {
			mu.Lock()
			order = append(order, i)
			mu.Unlock()
		}); err != nil {
			t.Fatalf("Post failed: %v", err)
		}
	}

	ctx := app.mkUpdateCtx()
	app.postMu.Lock()

	for _, fn := range app.postQueue {
		fn(ctx)
	}

	app.postQueue = nil
	app.postMu.Unlock()

	mu.Lock()
	defer mu.Unlock()

	for i, v := range order {
		if v != i {
			t.Errorf("Order mismatch at %d: got %d, want %d", i, v, i)
		}
	}
}

func TestPost_NeverInline(t *testing.T) {
	app, _ := New(AppOpts{})

	var innerExecuted atomic.Bool

	if err := app.Post(func(ctx *UpdateCtx) {
		if err := app.Post(func(ctx *UpdateCtx) {
			innerExecuted.Store(true)
		}); err != nil {
			t.Errorf("inner Post failed: %v", err)
		}

		if innerExecuted.Load() {
			t.Error("Inner Post executed inline, should be deferred")
		}
	}); err != nil {
		t.Fatalf("outer Post failed: %v", err)
	}

	ctx := app.mkUpdateCtx()
	app.postMu.Lock()

	if len(app.postQueue) > 0 {
		fn := app.postQueue[0]
		app.postQueue = app.postQueue[1:]
		app.postMu.Unlock()
		fn(ctx)
	} else {
		app.postMu.Unlock()
	}

	app.postMu.Lock()

	if len(app.postQueue) != 1 {
		t.Errorf("Expected 1 queued inner post, got %d", len(app.postQueue))
	}

	app.postMu.Unlock()
}

func TestPost_NilFunc(t *testing.T) {
	app, _ := New(AppOpts{})

	err := app.Post(nil)
	if err != nil {
		t.Errorf("Post(nil) should return nil, got %v", err)
	}

	app.postMu.Lock()
	queued := len(app.postQueue)
	app.postMu.Unlock()

	if queued != 0 {
		t.Errorf("Post(nil) should not enqueue, got %d queued", queued)
	}
}

func TestCtx_Quit(t *testing.T) {
	app, _ := New(AppOpts{})
	app.running.Store(true)

	ctx := app.mkCtx(nil)

	if ctx.Quit == nil {
		t.Fatal("Ctx.Quit is nil")
	}

	ctx.Quit()

	if app.running.Load() {
		t.Error("Quit did not set running to false")
	}
}

func TestUpdateCtx_Quit(t *testing.T) {
	app, _ := New(AppOpts{})
	app.running.Store(true)

	ctx := app.mkUpdateCtx()

	if ctx.Quit == nil {
		t.Fatal("UpdateCtx.Quit is nil")
	}

	ctx.Quit()

	if app.running.Load() {
		t.Error("Quit did not set running to false")
	}
}

func TestQuit_Idempotent(t *testing.T) {
	app, _ := New(AppOpts{})
	app.running.Store(true)

	app.Quit()
	app.Quit()
	app.Quit()

	if app.running.Load() {
		t.Error("Quit should be idempotent")
	}
}

func TestQuit_FromClosedApp(t *testing.T) {
	app, _ := New(AppOpts{})
	app.running.Store(true)
	app.setClosed()

	app.Quit()

	app.closeMu.RLock()
	closed := app.closed
	app.closeMu.RUnlock()

	if !closed {
		t.Error("Quit should preserve closed state")
	}
}

func TestUpdateCtx_Invalidate(t *testing.T) {
	app, _ := New(AppOpts{})

	ctx := app.mkUpdateCtx()

	ctx.Invalidate(geom.Rect{X: 0, Y: 0, W: 10, H: 10})

	if len(app.invalidRects) != 1 {
		t.Errorf("Expected 1 invalid rect, got %d", len(app.invalidRects))
	}
}

func TestUpdateCtx_InvalidateAll(t *testing.T) {
	app, _ := New(AppOpts{})
	app.size = geom.Size{W: 80, H: 24}

	ctx := app.mkUpdateCtx()

	ctx.InvalidateAll()

	if len(app.invalidRects) != 1 {
		t.Errorf("Expected 1 invalid rect, got %d", len(app.invalidRects))
	}

	expected := geom.Rect{X: 0, Y: 0, W: 80, H: 24}
	if app.invalidRects[0] != expected {
		t.Errorf("Expected %v, got %v", expected, app.invalidRects[0])
	}
}

func TestUpdateCtx_InvalidateLayout(t *testing.T) {
	app, _ := New(AppOpts{})

	ctx := app.mkUpdateCtx()

	ctx.InvalidateLayout(ID(123))

	if !app.layoutDirty {
		t.Error("InvalidateLayout did not set layoutDirty")
	}
}

func TestUpdateCtx_RequestFocus(t *testing.T) {
	app, _ := New(AppOpts{})
	app.nodes[ID(1)] = &nodeEntry{id: ID(1), rect: geom.Rect{X: 0, Y: 0, W: 10, H: 1}}
	app.nodes[ID(2)] = &nodeEntry{id: ID(2), rect: geom.Rect{X: 0, Y: 1, W: 10, H: 1}}
	app.focusedID = ID(1)

	ctx := app.mkUpdateCtx()

	ctx.RequestFocus(ID(2))

	if app.focusedID != ID(2) {
		t.Errorf("Expected focusedID=2, got %d", app.focusedID)
	}
}

func TestFocus(t *testing.T) {
	app, _ := New(AppOpts{})
	app.nodes[ID(1)] = &nodeEntry{id: ID(1), rect: geom.Rect{X: 0, Y: 0, W: 10, H: 1}}
	app.nodes[ID(2)] = &nodeEntry{id: ID(2), rect: geom.Rect{X: 0, Y: 1, W: 10, H: 1}}
	app.focusedID = ID(1)

	app.Focus(ID(2))

	if app.focusedID != ID(2) {
		t.Errorf("Expected focusedID=2, got %d", app.focusedID)
	}
}

func TestPost_BoundedBatch(t *testing.T) {
	app, _ := New(AppOpts{})

	var executedCount int32

	for range 100 {
		if err := app.Post(func(ctx *UpdateCtx) {
			atomic.AddInt32(&executedCount, 1)
		}); err != nil {
			t.Fatalf("Post failed: %v", err)
		}
	}

	ctx := app.mkUpdateCtx()

	app.postMu.Lock()

	count := 0
	for len(app.postQueue) > 0 && count < 64 {
		fn := app.postQueue[0]
		app.postQueue = app.postQueue[1:]
		count++

		app.postMu.Unlock()
		fn(ctx)
		app.postMu.Lock()
	}

	app.postMu.Unlock()

	if atomic.LoadInt32(&executedCount) != 64 {
		t.Errorf("Expected 64 posts executed in first batch, got %d", executedCount)
	}

	app.postMu.Lock()
	remaining := len(app.postQueue)
	app.postMu.Unlock()

	if remaining != 36 {
		t.Errorf("Expected 36 posts remaining, got %d", remaining)
	}
}

type testView struct {
	id        ID
	rect      geom.Rect
	focusable bool
	children  []View
}

type testPlacement struct {
	rect geom.Rect
}

func (p testPlacement) Resolve(root View, screenSize geom.Size) geom.Rect {
	return p.rect
}

func newTestView(focusable bool) *testView {
	return &testView{id: NewID(), focusable: focusable}
}

func (v *testView) ID() ID                   { return v.id }
func (v *testView) MinSize() geom.Size       { return geom.Size{W: 1, H: 1} }
func (v *testView) Layout(r geom.Rect)       { v.rect = r }
func (v *testView) Rect() geom.Rect          { return v.rect }
func (v *testView) Paint(d Drawer, ctx *Ctx) {}
func (v *testView) Handle(e Event, ctx *Ctx) bool {
	return false
}
func (v *testView) Focusable() bool  { return v.focusable }
func (v *testView) Children() []View { return v.children }

type testRoot struct {
	id       ID
	rect     geom.Rect
	children []View

	layoutFirstChildRect geom.Rect
}

func newTestRoot(children ...View) *testRoot {
	return &testRoot{id: NewID(), children: children}
}

func (r *testRoot) ID() ID             { return r.id }
func (r *testRoot) MinSize() geom.Size { return geom.Size{W: 1, H: 1} }
func (r *testRoot) Layout(gr geom.Rect) {
	r.rect = gr
	if len(r.children) > 0 {
		r.children[0].Layout(r.layoutFirstChildRect)
	}
}
func (r *testRoot) Rect() geom.Rect               { return r.rect }
func (r *testRoot) Paint(d Drawer, ctx *Ctx)      {}
func (r *testRoot) Handle(e Event, ctx *Ctx) bool { return false }
func (r *testRoot) Children() []View              { return r.children }

func containsRect(rects []geom.Rect, want geom.Rect) bool {
	for _, r := range rects {
		if r == want {
			return true
		}
	}

	return false
}

func TestLayout_InvalidatesOldAndNewRectsOnMove(t *testing.T) {
	app, _ := New(AppOpts{})
	app.size = geom.Size{W: 20, H: 10}

	child := newTestView(true)
	root := newTestRoot(child)
	root.layoutFirstChildRect = geom.Rect{X: 0, Y: 0, W: 5, H: 1}

	app.SetRoot(root)
	app.layout()
	app.invalidRects = nil

	oldRect := root.layoutFirstChildRect
	root.layoutFirstChildRect = geom.Rect{X: 0, Y: 1, W: 5, H: 1}
	newRect := root.layoutFirstChildRect

	app.layoutDirty = true
	app.layout()

	if !containsRect(app.invalidRects, oldRect) {
		t.Errorf("expected old rect to be invalidated: %v", oldRect)
	}

	if !containsRect(app.invalidRects, newRect) {
		t.Errorf("expected new rect to be invalidated: %v", newRect)
	}
}

func TestLayout_FocusRepair_WhenFocusedViewRemoved(t *testing.T) {
	app, _ := New(AppOpts{})
	app.size = geom.Size{W: 20, H: 10}

	a := newTestView(true)
	b := newTestView(true)
	root := newTestRoot(a, b)
	root.layoutFirstChildRect = geom.Rect{X: 0, Y: 0, W: 5, H: 1}

	app.SetRoot(root)
	app.layout()
	app.invalidRects = nil

	// Focus the first child, then remove it from the mounted tree.
	app.focusedID = a.ID()
	root.children = []View{b}

	app.layoutDirty = true
	app.layout()

	if app.focusedID != b.ID() {
		t.Fatalf("expected focus repaired to remaining focusable view, got %v want %v", app.focusedID, b.ID())
	}
}

func TestSetRequestFocus_ClearFocusDoesNotInvalidateAll(t *testing.T) {
	app, _ := New(AppOpts{})
	app.size = geom.Size{W: 20, H: 10}

	oldID := ID(1)
	oldRect := geom.Rect{X: 1, Y: 2, W: 3, H: 1}
	app.nodes[oldID] = &nodeEntry{id: oldID, rect: oldRect}
	app.focusedID = oldID

	app.setRequestFocus(0)

	if len(app.invalidRects) != 1 {
		t.Fatalf("expected 1 invalid rect, got %d", len(app.invalidRects))
	}

	if app.invalidRects[0] != oldRect {
		t.Fatalf("expected invalid rect %v, got %v", oldRect, app.invalidRects[0])
	}
}

func TestEnrichMouseEvent_WheelDoesNotPoison(t *testing.T) {
	app, _ := New(AppOpts{})

	// Simulate a wheel-up press (no corresponding release ever arrives).
	wheel := MouseEvent{
		Button: MouseButtonWheelUp,
		Action: event.MousePress,
		X:      5, Y: 5,
	}
	app.enrichMouseEvent(&wheel)

	// pressButton must remain None — wheel events should not set it.
	if app.mouse.pressButton != event.MouseButtonNone {
		t.Fatalf("pressButton = %v after wheel press, want MouseButtonNone",
			app.mouse.pressButton)
	}

	// A subsequent mouse move must stay a move, not be promoted to drag.
	move := MouseEvent{Action: event.MouseMove, X: 10, Y: 10}
	app.enrichMouseEvent(&move)

	if move.Action != event.MouseMove {
		t.Fatalf("move.Action = %v after wheel press, want MouseMove", move.Action)
	}

	if move.Button != event.MouseButtonNone {
		t.Fatalf("move.Button = %v after wheel press, want MouseButtonNone", move.Button)
	}
}

func TestCompactEventBatch_NetsQueuedWheelBacklog(t *testing.T) {
	app, _ := New(AppOpts{})

	batch := []Event{
		MouseEvent{Button: MouseButtonWheelDown, Action: event.MousePress, WheelDelta: 100},
		MouseEvent{Button: MouseButtonWheelUp, Action: event.MousePress, WheelDelta: 50},
		MouseEvent{Button: MouseButtonWheelDown, Action: event.MousePress, WheelDelta: 25},
		KeyEvent{Key: event.KeyTab},
	}

	got := app.compactEventBatch(batch)
	if len(got) != 2 {
		t.Fatalf("len(compacted) = %d, want 2", len(got))
	}

	me, ok := got[0].(MouseEvent)
	if !ok {
		t.Fatalf("got[0] = %T, want MouseEvent", got[0])
	}

	if me.Button != MouseButtonWheelDown || me.WheelDelta != 75 {
		t.Fatalf("compacted wheel = (%v,%d), want (%v,75)", me.Button, me.WheelDelta, MouseButtonWheelDown)
	}

	ke, ok := got[1].(KeyEvent)
	if !ok || ke.Key != event.KeyTab {
		t.Fatalf("got[1] = %#v, want KeyTab", got[1])
	}
}

func TestUpdateCtx_OverlayOperations(t *testing.T) {
	app, _ := New(AppOpts{})
	app.running.Store(true)

	ctx := app.mkUpdateCtx()

	// Test that overlay operations are wired up
	if ctx.ShowOverlay == nil {
		t.Fatal("UpdateCtx.ShowOverlay is nil")
	}

	if ctx.DismissOverlay == nil {
		t.Fatal("UpdateCtx.DismissOverlay is nil")
	}

	if ctx.DismissOverlayByID == nil {
		t.Fatal("UpdateCtx.DismissOverlayByID is nil")
	}
}

func TestUpdateCtx_ShowOverlayViaPost(t *testing.T) {
	app, _ := New(AppOpts{})

	var overlayShown bool
	var overlayID ID

	err := app.Post(func(ctx *UpdateCtx) {
		o := ctx.ShowOverlay(OverlayOpts{
			Root:  &testView{id: NewID(), focusable: true},
			Modal: true,
			Place: testPlacement{rect: geom.Rect{X: 0, Y: 0, W: 10, H: 5}},
		})
		if o != nil {
			overlayShown = true
			overlayID = o.ID()
		}
	})
	if err != nil {
		t.Fatalf("Post failed: %v", err)
	}

	// Execute the posted callback
	app.postMu.Lock()
	for _, fn := range app.postQueue {
		fn(app.mkUpdateCtx())
	}
	app.postQueue = nil
	app.postMu.Unlock()

	if !overlayShown {
		t.Error("overlay was not shown via UpdateCtx.ShowOverlay")
	}

	if overlayID == 0 {
		t.Error("overlay ID should not be zero")
	}

	// Verify overlay is on stack
	if app.overlays.OverlayCount() != 1 {
		t.Errorf("overlay count = %d, want 1", app.overlays.OverlayCount())
	}
}

func TestUpdateCtx_DismissOverlayViaPost(t *testing.T) {
	app, _ := New(AppOpts{})

	// First show an overlay via app directly
	o := app.ShowOverlay(OverlayOpts{
		Root:  &testView{id: NewID(), focusable: true},
		Modal: true,
		Place: testPlacement{rect: geom.Rect{X: 0, Y: 0, W: 10, H: 5}},
	})
	overlayID := o.ID()

	if app.overlays.OverlayCount() != 1 {
		t.Fatalf("setup: expected 1 overlay, got %d", app.overlays.OverlayCount())
	}

	// Dismiss via UpdateCtx
	var dismissed bool
	err := app.Post(func(ctx *UpdateCtx) {
		ctx.DismissOverlay()
		dismissed = true
	})
	if err != nil {
		t.Fatalf("Post failed: %v", err)
	}

	// Execute the posted callback
	app.postMu.Lock()
	for _, fn := range app.postQueue {
		fn(app.mkUpdateCtx())
	}
	app.postQueue = nil
	app.postMu.Unlock()

	if !dismissed {
		t.Error("overlay was not dismissed via UpdateCtx.DismissOverlay")
	}

	if app.overlays.OverlayCount() != 0 {
		t.Errorf("overlay count = %d, want 0", app.overlays.OverlayCount())
	}

	_ = overlayID // suppress unused warning
}

func TestUpdateCtx_DismissOverlayByIDViaPost(t *testing.T) {
	app, _ := New(AppOpts{})

	// Show two overlays
	o1 := app.ShowOverlay(OverlayOpts{
		Root:  &testView{id: NewID(), focusable: true},
		Modal: true,
		Place: testPlacement{rect: geom.Rect{X: 0, Y: 0, W: 10, H: 5}},
	})
	o2 := app.ShowOverlay(OverlayOpts{
		Root:  &testView{id: NewID(), focusable: true},
		Modal: true,
		Place: testPlacement{rect: geom.Rect{X: 0, Y: 0, W: 10, H: 5}},
	})

	if app.overlays.OverlayCount() != 2 {
		t.Fatalf("setup: expected 2 overlays, got %d", app.overlays.OverlayCount())
	}

	// Dismiss bottom overlay by ID via UpdateCtx
	var dismissed bool
	targetID := o1.ID()
	err := app.Post(func(ctx *UpdateCtx) {
		ctx.DismissOverlayByID(targetID)
		dismissed = true
	})
	if err != nil {
		t.Fatalf("Post failed: %v", err)
	}

	// Execute the posted callback
	app.postMu.Lock()
	for _, fn := range app.postQueue {
		fn(app.mkUpdateCtx())
	}
	app.postQueue = nil
	app.postMu.Unlock()

	if !dismissed {
		t.Error("overlay was not dismissed via UpdateCtx.DismissOverlayByID")
	}

	// Should have one overlay remaining (the top one)
	if app.overlays.OverlayCount() != 1 {
		t.Errorf("overlay count = %d, want 1", app.overlays.OverlayCount())
	}

	// The remaining overlay should be o2
	remaining := app.overlays.TopOverlay()
	if remaining == nil || remaining.ID() != o2.ID() {
		t.Error("wrong overlay remaining after dismiss by ID")
	}
}

func TestDismissOverlayByID_NonTopPreservesFocus(t *testing.T) {
	app, _ := New(AppOpts{})

	// Show two overlays
	overlay1Focus := &testView{id: NewID(), focusable: true}
	o1 := app.ShowOverlay(OverlayOpts{
		Root:  overlay1Focus,
		Modal: true,
		Place: testPlacement{rect: geom.Rect{X: 0, Y: 0, W: 10, H: 5}},
	})
	overlay2Focus := &testView{id: NewID(), focusable: true}
	o2 := app.ShowOverlay(OverlayOpts{
		Root:  overlay2Focus,
		Modal: true,
		Place: testPlacement{rect: geom.Rect{X: 0, Y: 0, W: 10, H: 5}},
	})

	if app.focusedID != overlay2Focus.ID() {
		t.Fatalf("focusedID after showing top overlay = %v, want %v", app.focusedID, overlay2Focus.ID())
	}

	// Dismiss bottom overlay by ID
	// Focus should NOT be restored since o2 is still on top
	targetID := o1.ID()
	dismissed := app.DismissOverlayByID(targetID)

	if dismissed == nil {
		t.Fatal("expected non-nil dismissed overlay")
	}

	// Should have one overlay remaining (the top one)
	if app.overlays.OverlayCount() != 1 {
		t.Errorf("overlay count = %d, want 1", app.overlays.OverlayCount())
	}

	// The remaining overlay should be o2
	remaining := app.overlays.TopOverlay()
	if remaining == nil || remaining.ID() != o2.ID() {
		t.Error("wrong overlay remaining after dismiss by ID")
	}

	if app.focusedID != overlay2Focus.ID() {
		t.Fatalf("focusedID after dismissing non-top overlay = %v, want %v", app.focusedID, overlay2Focus.ID())
	}

	if app.focusedID == overlay1Focus.ID() {
		t.Fatal("dismissing a lower overlay restored focus to the wrong overlay")
	}
}

func TestDismissOverlayByID_TopRestoresFocus(t *testing.T) {
	app, _ := New(AppOpts{})

	baseFocus := NewID()
	app.focusedID = baseFocus

	overlayFocus := &testView{id: NewID(), focusable: true}
	o := app.ShowOverlay(OverlayOpts{
		Root:  overlayFocus,
		Modal: true,
		Place: testPlacement{rect: geom.Rect{X: 0, Y: 0, W: 10, H: 5}},
	})

	if app.focusedID != overlayFocus.ID() {
		t.Fatalf("focusedID after showing modal overlay = %v, want %v", app.focusedID, overlayFocus.ID())
	}

	dismissed := app.DismissOverlayByID(o.ID())
	if dismissed == nil {
		t.Fatal("expected non-nil dismissed overlay")
	}

	if app.focusedID != baseFocus {
		t.Fatalf("focusedID after dismissing top overlay by ID = %v, want %v", app.focusedID, baseFocus)
	}
}

func TestDismissOverlayByID_CleansUpScopeMemory(t *testing.T) {
	app, _ := New(AppOpts{})

	// Show an overlay
	o := app.ShowOverlay(OverlayOpts{
		Root:  &testView{id: NewID(), focusable: true},
		Modal: true,
		Place: testPlacement{rect: geom.Rect{X: 0, Y: 0, W: 10, H: 5}},
	})

	overlayID := o.ID()

	// Dismiss by ID
	dismissed := app.DismissOverlayByID(overlayID)

	if dismissed == nil {
		t.Fatal("expected non-nil dismissed overlay")
	}

	// Scope memory should be cleaned up
	if _, exists := app.scopeMemory[overlayID]; exists {
		t.Error("scope memory not cleaned up after DismissOverlayByID")
	}
}

func TestDismissOverlayByID_UnknownIDIsHarmless(t *testing.T) {
	app, _ := New(AppOpts{})

	// Show an overlay
	app.ShowOverlay(OverlayOpts{
		Root:  &testView{id: NewID(), focusable: true},
		Modal: true,
		Place: testPlacement{rect: geom.Rect{X: 0, Y: 0, W: 10, H: 5}},
	})

	// Try to dismiss unknown ID
	dismissed := app.DismissOverlayByID(ID(999999))

	if dismissed != nil {
		t.Error("expected nil for unknown ID")
	}

	// Existing overlay should still be there
	if app.overlays.OverlayCount() != 1 {
		t.Errorf("overlay count = %d, want 1", app.overlays.OverlayCount())
	}
}

func TestQuit_WakesEventLoop(t *testing.T) {
	app, _ := New(AppOpts{})
	app.running.Store(true)

	// Quit should wake the loop (test by checking it sends to wakeCh)
	select {
	case app.wakeCh <- struct{}{}:
		// Already has pending wakeup, that's fine
	default:
	}

	// Drain the channel
	select {
	case <-app.wakeCh:
	default:
	}

	// Quit should wake
	app.Quit()

	select {
	case <-app.wakeCh:
		// Good: Quit woke the loop
	default:
		t.Error("Quit did not wake the event loop")
	}
}

func TestShowOverlay_NonModalDoesNotStealFocus(t *testing.T) {
	app, _ := New(AppOpts{})
	baseFocus := NewID()
	app.focusedID = baseFocus

	overlayFocus := &testView{id: NewID(), focusable: true}
	app.ShowOverlay(OverlayOpts{
		Root:  overlayFocus,
		Modal: false,
		Place: testPlacement{rect: geom.Rect{X: 0, Y: 0, W: 10, H: 5}},
	})

	if app.focusedID != baseFocus {
		t.Fatalf("focusedID after showing non-modal overlay = %v, want %v", app.focusedID, baseFocus)
	}
}

func TestQuit_UnblocksIdleRun(t *testing.T) {
	be := headless.New(geom.Size{W: 80, H: 24})
	app, err := New(AppOpts{Backend: be})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	app.SetRoot(&testRoot{id: NewID()})
	if err := app.Enable(); err != nil {
		t.Fatalf("Enable: %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- app.Run() }()

	time.Sleep(20 * time.Millisecond)
	app.Quit()
	be.Close()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after Quit while idle")
	}
}

func TestQuit_RepeatedCallsAreIdempotent(t *testing.T) {
	app, _ := New(AppOpts{})
	app.running.Store(true)

	// Drain channel
	select {
	case <-app.wakeCh:
	default:
	}

	// Multiple Quit calls should all work
	app.Quit()
	app.Quit()
	app.Quit()

	// All should have sent to wakeCh (but only one survives due to buffer)
	// This test just verifies no panic or deadlock
	if app.running.Load() {
		t.Error("running should be false after Quit")
	}
}

func TestQuit_RacesWithQueuedPosts(t *testing.T) {
	be := headless.New(geom.Size{W: 80, H: 24})
	app, err := New(AppOpts{Backend: be})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	app.SetRoot(&testRoot{id: NewID()})
	if err := app.Enable(); err != nil {
		t.Fatalf("Enable: %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- app.Run() }()

	var ran atomic.Int32
	postErr := make(chan error, 1)
	go func() {
		postErr <- app.Post(func(ctx *UpdateCtx) {
			ran.Add(1)
			ctx.InvalidateAll()
		})
		app.Quit()
	}()

	select {
	case err := <-postErr:
		if err != nil {
			t.Fatalf("Post failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for post result")
	}

	be.Close()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after Quit racing with queued posts")
	}

	if ran.Load() > 1 {
		t.Fatalf("queued post ran %d times, want at most 1", ran.Load())
	}
}

func TestPostInvalidate_SchedulesRepaint(t *testing.T) {
	app, _ := New(AppOpts{})

	err := app.PostInvalidate(geom.Rect{X: 0, Y: 0, W: 10, H: 10})
	if err != nil {
		t.Fatalf("PostInvalidate failed: %v", err)
	}

	// Verify it was queued
	app.postMu.Lock()
	queued := len(app.postQueue)
	app.postMu.Unlock()

	if queued != 1 {
		t.Errorf("expected 1 queued post, got %d", queued)
	}

	// Execute and verify invalidation
	app.postMu.Lock()
	for _, fn := range app.postQueue {
		fn(app.mkUpdateCtx())
	}
	app.postQueue = nil
	app.postMu.Unlock()

	// Check that the rect was invalidated
	found := false
	for _, r := range app.invalidRects {
		if r == (geom.Rect{X: 0, Y: 0, W: 10, H: 10}) {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected rect to be invalidated")
	}
}

func TestPostInvalidateAll_SchedulesFullRepaint(t *testing.T) {
	app, _ := New(AppOpts{})
	app.size = geom.Size{W: 80, H: 24}

	err := app.PostInvalidateAll()
	if err != nil {
		t.Fatalf("PostInvalidateAll failed: %v", err)
	}

	// Verify it was queued
	app.postMu.Lock()
	queued := len(app.postQueue)
	app.postMu.Unlock()

	if queued != 1 {
		t.Errorf("expected 1 queued post, got %d", queued)
	}

	// Execute and verify invalidation
	app.postMu.Lock()
	for _, fn := range app.postQueue {
		fn(app.mkUpdateCtx())
	}
	app.postQueue = nil
	app.postMu.Unlock()

	// Check that entire screen was invalidated
	found := false
	for _, r := range app.invalidRects {
		if r == (geom.Rect{X: 0, Y: 0, W: 80, H: 24}) {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected full screen to be invalidated")
	}
}

func TestPostInvalidateFromGoroutine(t *testing.T) {
	app, _ := New(AppOpts{})

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		err := app.PostInvalidate(geom.Rect{X: 5, Y: 5, W: 20, H: 10})
		if err != nil {
			t.Errorf("PostInvalidate from goroutine failed: %v", err)
		}
	}()

	wg.Wait()

	// Should have been queued
	app.postMu.Lock()
	queued := len(app.postQueue)
	app.postMu.Unlock()

	if queued != 1 {
		t.Errorf("expected 1 queued post, got %d", queued)
	}
}

// floodBackend is a backend.Backend whose ReadEvent never blocks: it returns an
// endless stream of key events. This lets a test force App.readEvents to fill
// eventCh and park on a send.
type floodBackend struct{}

func (floodBackend) Enable() (geom.Size, error) { return geom.Size{W: 80, H: 24}, nil }
func (floodBackend) Restore() error             { return nil }
func (floodBackend) ReadEvent() event.Event     { return event.KeyEvent{Key: event.KeyRune, Rune: 'x'} }
func (floodBackend) Size() geom.Size            { return geom.Size{W: 80, H: 24} }

// After the app loop exits, App.readEvents must not park forever on a send into
// an eventCh that no one drains. With an endless input stream the forwarder
// fills eventCh and blocks on a send; setClosed must then unblock it so the
// goroutine (and eventCh) can't leak.
func TestReadEvents_ForwarderExitsAfterClose(t *testing.T) {
	app, _ := New(AppOpts{Backend: floodBackend{}})
	app.host = newAppHost(floodBackend{})
	app.running.Store(true)

	go app.readEvents()

	// Wait until the forwarder has filled eventCh and is blocked on a send.
	deadline := time.Now().Add(2 * time.Second)
	for len(app.eventCh) < cap(app.eventCh) {
		if time.Now().After(deadline) {
			t.Fatal("forwarder never filled eventCh")
		}

		runtime.Gosched()
	}

	// Simulate the app loop exiting; this must release the blocked send.
	app.setClosed()

	closed := make(chan struct{})
	go func() {
		for range app.eventCh { //nolint:revive // drain until closed
		}

		close(closed)
	}()

	select {
	case <-closed:
		// Forwarder exited and closed eventCh: no leak.
	case <-time.After(2 * time.Second):
		t.Fatal("readEvents leaked: eventCh not closed after setClosed")
	}
}

// Placement helper tests
