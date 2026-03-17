package tui_test

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/losinggeneration/tui"
	"github.com/losinggeneration/tui/backend/headless"
	"github.com/losinggeneration/tui/event"
	"github.com/losinggeneration/tui/geom"
	"github.com/losinggeneration/tui/style"
	"github.com/losinggeneration/tui/ui"
)

// integrationView is a test view that tracks paint calls and handles focus/input.
type integrationView struct {
	id          tui.ID
	rect        geom.Rect
	paintCount  atomic.Int32
	handleCount atomic.Int32
	focusable   bool
	children    []tui.View
}

func newIntView(focusable bool) *integrationView {
	return &integrationView{id: tui.NewID(), focusable: focusable}
}

func (v *integrationView) ID() tui.ID           { return v.id }
func (v *integrationView) MinSize() geom.Size   { return geom.Size{W: 1, H: 1} }
func (v *integrationView) Layout(r geom.Rect)   { v.rect = r }
func (v *integrationView) Rect() geom.Rect      { return v.rect }
func (v *integrationView) Focusable() bool      { return v.focusable }
func (v *integrationView) Children() []tui.View { return v.children }

func (v *integrationView) Paint(p *tui.Painter, ctx *tui.Ctx) {
	v.paintCount.Add(1)
}

func (v *integrationView) Handle(e tui.Event, ctx *tui.Ctx) bool {
	v.handleCount.Add(1)
	return false
}

// integrationRoot wraps children and handles layout distribution.
type integrationRoot struct {
	id         tui.ID
	rect       geom.Rect
	children   []tui.View
	paintCount atomic.Int32
}

func newIntRoot(children ...tui.View) *integrationRoot {
	return &integrationRoot{id: tui.NewID(), children: children}
}

func (r *integrationRoot) ID() tui.ID           { return r.id }
func (r *integrationRoot) MinSize() geom.Size   { return geom.Size{W: 1, H: 1} }
func (r *integrationRoot) Rect() geom.Rect      { return r.rect }
func (r *integrationRoot) Children() []tui.View { return r.children }

func (r *integrationRoot) Layout(gr geom.Rect) {
	r.rect = gr
	for i, c := range r.children {
		c.Layout(geom.Rect{X: 0, Y: i, W: gr.W, H: 1})
	}
}

func (r *integrationRoot) Paint(p *tui.Painter, ctx *tui.Ctx) {
	r.paintCount.Add(1)

	for _, c := range r.children {
		c.Paint(p, ctx)
	}
}

func (r *integrationRoot) Handle(e tui.Event, ctx *tui.Ctx) bool {
	for _, c := range r.children {
		if c.Handle(e, ctx) {
			return true
		}
	}

	return false
}

func TestIntegration_FullLifecycle(t *testing.T) {
	be := headless.New(geom.Size{W: 80, H: 24})
	cap := style.Capability{HasBasic: true}
	opts := tui.AppOpts{
		Backend:       be,
		Capability:    &cap,
		Theme:         tui.DefaultTheme(),
		ResolveAction: ui.DefaultAppResolver(),
	}

	app, err := tui.New(opts)
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
	_ = app.Post(func(ctx *tui.UpdateCtx) {
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

	err = app.Post(func(ctx *tui.UpdateCtx) {
		postRan.Store(true)
		ctx.InvalidateAll()
	})
	if err != nil {
		t.Fatalf("Post: %v", err)
	}

	waitFor("post callback", func() bool { return postRan.Load() })

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
	err = app.Post(func(ctx *tui.UpdateCtx) {})
	if err != tui.ErrClosed {
		t.Errorf("Post after shutdown: got %v, want ErrClosed", err)
	}
}
