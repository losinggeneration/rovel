package rovel_test

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/losinggeneration/rovel"
	"github.com/losinggeneration/rovel/backend/headless"
	"github.com/losinggeneration/rovel/backend/memory"
	"github.com/losinggeneration/rovel/event"
	"github.com/losinggeneration/rovel/geom"
	"github.com/losinggeneration/rovel/style"
	"github.com/losinggeneration/rovel/ui"
	"github.com/losinggeneration/rovel/ui/widgets"
)

type fixedPlacement struct {
	rect geom.Rect
}

func (p fixedPlacement) Resolve(root rovel.View, screenSize geom.Size) geom.Rect {
	return p.rect
}

// integrationView is a test view that tracks paint calls and handles focus/input.
type integrationView struct {
	id          rovel.ID
	rect        geom.Rect
	paintCount  atomic.Int32
	handleCount atomic.Int32
	focusable   bool
	children    []rovel.View
}

func newIntView(focusable bool) *integrationView {
	return &integrationView{id: rovel.NewID(), focusable: focusable}
}

func (v *integrationView) ID() rovel.ID           { return v.id }
func (v *integrationView) MinSize() geom.Size     { return geom.Size{W: 1, H: 1} }
func (v *integrationView) Layout(r geom.Rect)     { v.rect = r }
func (v *integrationView) Rect() geom.Rect        { return v.rect }
func (v *integrationView) Focusable() bool        { return v.focusable }
func (v *integrationView) Children() []rovel.View { return v.children }

func (v *integrationView) Paint(d rovel.Drawer, ctx *rovel.Ctx) {
	v.paintCount.Add(1)
}

func (v *integrationView) Handle(e rovel.Event, ctx *rovel.Ctx) bool {
	v.handleCount.Add(1)

	return false
}

// integrationRoot wraps children and handles layout distribution.
type integrationRoot struct {
	id         rovel.ID
	rect       geom.Rect
	children   []rovel.View
	paintCount atomic.Int32
}

func newIntRoot(children ...rovel.View) *integrationRoot {
	return &integrationRoot{id: rovel.NewID(), children: children}
}

func (r *integrationRoot) ID() rovel.ID           { return r.id }
func (r *integrationRoot) MinSize() geom.Size     { return geom.Size{W: 1, H: 1} }
func (r *integrationRoot) Rect() geom.Rect        { return r.rect }
func (r *integrationRoot) Children() []rovel.View { return r.children }

func (r *integrationRoot) Layout(gr geom.Rect) {
	r.rect = gr
	for i, c := range r.children {
		c.Layout(geom.Rect{X: 0, Y: i, W: gr.W, H: 1})
	}
}

func (r *integrationRoot) Paint(d rovel.Drawer, ctx *rovel.Ctx) {
	r.paintCount.Add(1)

	for _, c := range r.children {
		c.Paint(d, ctx)
	}
}

func (r *integrationRoot) Handle(e rovel.Event, ctx *rovel.Ctx) bool {
	for _, c := range r.children {
		if c.Handle(e, ctx) {
			return true
		}
	}

	return false
}

type asyncInvalidateView struct {
	id    rovel.ID
	rect  geom.Rect
	value atomic.Int32
}

func newAsyncInvalidateView(initial int32) *asyncInvalidateView {
	v := &asyncInvalidateView{id: rovel.NewID()}
	v.value.Store(initial)

	return v
}

func (v *asyncInvalidateView) ID() rovel.ID       { return v.id }
func (v *asyncInvalidateView) MinSize() geom.Size { return geom.Size{W: 1, H: 1} }
func (v *asyncInvalidateView) Rect() geom.Rect    { return v.rect }
func (v *asyncInvalidateView) Layout(r geom.Rect) { v.rect = r }
func (v *asyncInvalidateView) Paint(d rovel.Drawer, ctx *rovel.Ctx) {
	d.FillRect(v.rect, rovel.Style{})
	d.DrawText(
		rovel.Point{X: v.rect.X, Y: v.rect.Y},
		string(rune('0'+v.value.Load())),
		rovel.Style{},
	)
}
func (v *asyncInvalidateView) Handle(e rovel.Event, ctx *rovel.Ctx) bool { return false }

func TestIntegration_FullLifecycle(t *testing.T) {
	be := headless.New(geom.Size{W: 80, H: 24})
	c := style.Capability{HasBasic: true}
	opts := rovel.AppOpts{
		Backend:       be,
		Capability:    &c,
		Theme:         rovel.DefaultTheme(),
		ResolveAction: ui.DefaultAppResolver(),
	}

	app, err := rovel.New(opts)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	child1 := newIntView(true)
	child2 := newIntView(true)
	root := newIntRoot(child1, child2)

	app.SetRoot(root)

	if err := app.Enable(); err != nil {
		t.Fatalf("Enable: %v", err)
	}

	// Run the app in a goroutine.
	done := make(chan error, 1)

	go func() { done <- app.Run() }()

	// Helper to wait for an atomic condition with timeout.
	waitFor := func(name string, check func() bool) {
		t.Helper()

		deadline := time.After(2 * time.Second)

		for {
			if check() {
				return
			}

			select {
			case <-deadline:
				t.Fatalf("timeout waiting for %s", name)
			default:
				time.Sleep(5 * time.Millisecond)
			}
		}
	}

	// 1. Initial paint should have happened.
	waitFor("initial paint", func() bool { return root.paintCount.Load() > 0 })

	// 2. Send a key event and wait for it to be handled.
	be.SendEvent(event.KeyEvent{Key: event.KeyRune, Rune: 'a'})
	waitFor("key event", func() bool {
		return child1.handleCount.Load() > 0 || child2.handleCount.Load() > 0
	})

	// 3. Send a resize event.
	paintsBefore := root.paintCount.Load()

	be.SendResize(100, 30)
	waitFor("resize repaint", func() bool { return root.paintCount.Load() > paintsBefore })

	// Verify size via Post to avoid racing with the app loop.
	sizeCh := make(chan geom.Size, 1)
	_ = app.Post(func(ctx *rovel.UpdateCtx) {
		sizeCh <- app.Size()
	})

	select {
	case sz := <-sizeCh:
		if sz.W != 100 || sz.H != 30 {
			t.Errorf("size after resize: got %v, want {100 30}", sz)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout getting size")
	}

	// 4. Test Post from background goroutine.
	var postRan atomic.Bool

	err = app.Post(func(ctx *rovel.UpdateCtx) {
		postRan.Store(true)
		ctx.InvalidateAll()
	})
	if err != nil {
		t.Fatalf("Post: %v", err)
	}

	waitFor("post callback", postRan.Load)

	// 5. Shut down.
	app.Quit()
	be.Close()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for Run to return")
	}

	// 6. Verify restore was called.
	if err := app.Restore(); err != nil {
		t.Fatalf("Restore: %v", err)
	}

	if !be.Restored() {
		t.Error("backend Restore was not called")
	}

	// 7. Post after shutdown should return ErrClosed.
	err = app.Post(func(ctx *rovel.UpdateCtx) {})
	if !errors.Is(err, rovel.ErrClosed) {
		t.Errorf("Post after shutdown: got %v, want ErrClosed", err)
	}
}

func TestIntegration_PostInvalidate_RepaintsThroughRunLoop(t *testing.T) {
	be := memory.New(geom.Size{W: 12, H: 3})
	opts := rovel.AppOpts{Backend: be}

	app, err := rovel.New(opts)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	view := newAsyncInvalidateView(1)
	app.SetRoot(view)

	if err := app.Enable(); err != nil {
		t.Fatalf("Enable: %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- app.Run() }()

	waitFor := func(name string, check func() bool) {
		t.Helper()

		deadline := time.After(2 * time.Second)
		for {
			if check() {
				return
			}

			select {
			case <-deadline:
				t.Fatalf("timeout waiting for %s", name)
			default:
				time.Sleep(5 * time.Millisecond)
			}
		}
	}

	frame, ok := be.LastFrame()
	if !ok {
		t.Fatal("expected initial frame")
	}
	if got := string(frame.RowRunes(0)[:1]); got != "1" {
		t.Fatalf("initial top-left rune = %q, want %q", got, "1")
	}

	framesBefore := be.FrameCount()
	go func() {
		view.value.Store(2)
		_ = app.PostInvalidate(view.Rect())
	}()

	waitFor("post invalidate repaint", func() bool {
		return be.FrameCount() > framesBefore
	})

	frame, ok = be.LastFrame()
	if !ok {
		t.Fatal("expected frame after PostInvalidate")
	}
	if got := string(frame.RowRunes(0)[:1]); got != "2" {
		t.Fatalf("updated top-left rune = %q, want %q", got, "2")
	}

	app.Quit()
	be.Close()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for Run to return")
	}
}

func TestIntegration_MemoryBackendCapturesLogicalFrames(t *testing.T) {
	be := memory.New(geom.Size{W: 12, H: 4})
	c := style.Capability{HasBasic: true}

	app, err := rovel.New(rovel.AppOpts{
		Backend:    be,
		Capability: &c,
		Theme:      rovel.DefaultTheme(),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	label := widgets.NewLabel("hello")
	app.SetRoot(label)

	if err := app.Enable(); err != nil {
		t.Fatalf("Enable: %v", err)
	}

	frame, ok := be.LastFrame()
	if !ok {
		t.Fatal("expected an initial logical frame")
	}

	if frame.W != 12 || frame.H != 4 {
		t.Fatalf("frame size = %dx%d, want 12x4", frame.W, frame.H)
	}

	got := make([]rune, 5)
	for i := range got {
		got[i] = frame.Cells[i].R
	}

	if string(got) != "hello" {
		t.Fatalf("top row prefix = %q, want %q", string(got), "hello")
	}

	done := make(chan error, 1)

	go func() { done <- app.Run() }()

	waitFor := func(name string, check func() bool) {
		t.Helper()

		deadline := time.After(2 * time.Second)

		for {
			if check() {
				return
			}

			select {
			case <-deadline:
				t.Fatalf("timeout waiting for %s", name)
			default:
				time.Sleep(5 * time.Millisecond)
			}
		}
	}

	initialFrames := be.FrameCount()
	be.SendResize(14, 5)
	waitFor("memory resize frame", func() bool {
		return be.FrameCount() > initialFrames
	})

	frame, ok = be.LastFrame()
	if !ok {
		t.Fatal("expected frame after resize")
	}

	if frame.W != 14 || frame.H != 5 {
		t.Fatalf("resized frame size = %dx%d, want 14x5", frame.W, frame.H)
	}

	app.Quit()
	be.Close()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for Run to return")
	}
}

func TestIntegration_MemoryBackendCapturesPostedUpdatesAndOverlays(t *testing.T) {
	be := memory.New(geom.Size{W: 16, H: 6})
	c := style.Capability{HasBasic: true}

	app, err := rovel.New(rovel.AppOpts{
		Backend:    be,
		Capability: &c,
		Theme:      rovel.DefaultTheme(),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	label := widgets.NewLabel("base")
	app.SetRoot(label)

	if err := app.Enable(); err != nil {
		t.Fatalf("Enable: %v", err)
	}

	done := make(chan error, 1)

	go func() { done <- app.Run() }()

	waitFor := func(name string, check func() bool) {
		t.Helper()

		deadline := time.After(2 * time.Second)

		for {
			if check() {
				return
			}

			select {
			case <-deadline:
				t.Fatalf("timeout waiting for %s", name)
			default:
				time.Sleep(5 * time.Millisecond)
			}
		}
	}

	initialFrames := be.FrameCount()

	err = app.Post(func(ctx *rovel.UpdateCtx) {
		label.SetText(nil, "updated")
		ctx.Invalidate(label.Rect())
	})
	if err != nil {
		t.Fatalf("Post update: %v", err)
	}

	waitFor("posted update frame", func() bool {
		return be.FrameCount() > initialFrames
	})

	frame, ok := be.LastFrame()
	if !ok {
		t.Fatal("expected frame after posted update")
	}

	if got := string(frame.RowRunes(0)[:7]); got != "updated" {
		t.Fatalf("updated top row prefix = %q, want %q", got, "updated")
	}

	framesBeforeOverlay := be.FrameCount()

	err = app.Post(func(ctx *rovel.UpdateCtx) {
		o := ctx.ShowOverlay(rovel.OverlayOpts{
			Root:  widgets.NewLabel("OVR"),
			Modal: false,
			Place: fixedPlacement{rect: geom.Rect{X: 2, Y: 2, W: 3, H: 1}},
		})
		if o != nil {
			ctx.Invalidate(o.Rect())
		}
	})
	if err != nil {
		t.Fatalf("Post overlay: %v", err)
	}

	waitFor("overlay frame", func() bool {
		return be.FrameCount() > framesBeforeOverlay
	})

	frame, ok = be.LastFrame()
	if !ok {
		t.Fatal("expected frame after overlay")
	}

	if got := string(frame.RowRunes(2)[2:5]); got != "OVR" {
		t.Fatalf("overlay row segment = %q, want %q", got, "OVR")
	}

	if got := string(frame.RowRunes(0)[:7]); got != "updated" {
		t.Fatalf("base content after overlay = %q, want %q", got, "updated")
	}

	app.Quit()
	be.Close()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for Run to return")
	}
}
