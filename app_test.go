package tui

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/losinggeneration/tui/geom"
)

func TestPost_ErrClosed(t *testing.T) {
	app, _ := New(AppOpts{})
	app.setClosed()

	err := app.Post(func(ctx *UpdateCtx) {})
	if err != ErrClosed {
		t.Errorf("Post after closed: got %v, want ErrClosed", err)
	}
}

func TestPost_FromGoroutine(t *testing.T) {
	app, _ := New(AppOpts{})

	var wg sync.WaitGroup
	var posted atomic.Bool

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

	var order []int
	var mu sync.Mutex

	for i := 0; i < 5; i++ {
		i := i
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
	app.rectByID[ID(1)] = geom.Rect{X: 0, Y: 0, W: 10, H: 1}
	app.rectByID[ID(2)] = geom.Rect{X: 0, Y: 1, W: 10, H: 1}
	app.focusedID = ID(1)

	ctx := app.mkUpdateCtx()

	ctx.RequestFocus(ID(2))

	if app.focusedID != ID(2) {
		t.Errorf("Expected focusedID=2, got %d", app.focusedID)
	}
}

func TestPost_BoundedBatch(t *testing.T) {
	app, _ := New(AppOpts{})

	var executedCount int32

	for i := 0; i < 100; i++ {
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
