package tui

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"

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
}

func newTestView(focusable bool) *testView {
	return &testView{id: NewID(), focusable: focusable}
}

func (v *testView) ID() ID                     { return v.id }
func (v *testView) MinSize() geom.Size         { return geom.Size{W: 1, H: 1} }
func (v *testView) Layout(r geom.Rect)         { v.rect = r }
func (v *testView) Rect() geom.Rect            { return v.rect }
func (v *testView) Paint(p *Painter, ctx *Ctx) {}
func (v *testView) Handle(e Event, ctx *Ctx) bool {
	return false
}
func (v *testView) Focusable() bool { return v.focusable }

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
func (r *testRoot) Paint(p *Painter, ctx *Ctx)    {}
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
