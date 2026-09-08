package rovel_test

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/losinggeneration/rovel"
	"github.com/losinggeneration/rovel/backend/memory"
	"github.com/losinggeneration/rovel/event"
	"github.com/losinggeneration/rovel/geom"
	"github.com/losinggeneration/rovel/style"
	"github.com/losinggeneration/rovel/ui/overlay"
)

// The movable-window tests compose a window the way an embedder would — a
// root view with a draggable title bar over a body — and drive the full app
// loop through the memory backend, asserting on painted cells only.

// desktopView fills its rect with '.' so restored (formerly covered) cells are
// distinguishable from window content.
type desktopView struct {
	id   rovel.ID
	rect geom.Rect
}

func newDesktopView() *desktopView { return &desktopView{id: rovel.NewID()} }

func (v *desktopView) ID() rovel.ID           { return v.id }
func (v *desktopView) MinSize() geom.Size     { return geom.Size{W: 1, H: 1} }
func (v *desktopView) Rect() geom.Rect        { return v.rect }
func (v *desktopView) Layout(r geom.Rect)     { v.rect = r }
func (v *desktopView) Children() []rovel.View { return nil }

func (v *desktopView) Paint(d rovel.Drawer, ctx *rovel.Ctx) {
	for y := v.rect.Y; y < v.rect.Y+v.rect.H; y++ {
		d.DrawText(geom.Point{X: v.rect.X, Y: y}, repeatRune('.', v.rect.W), rovel.Style{})
	}
}

func (v *desktopView) Handle(e rovel.Event, ctx *rovel.Ctx) bool { return false }

func repeatRune(r rune, n int) string {
	out := make([]rune, n)
	for i := range out {
		out[i] = r
	}
	return string(out)
}

// titleBarView paints its title at its rect origin, raises its window's
// overlay on press, and drags the shared Floating placement while the button
// is held.
type titleBarView struct {
	id       rovel.ID
	rect     geom.Rect
	title    string
	place    *overlay.Floating
	raiseID  rovel.ID // set after ShowOverlay; press raises the overlay
	raised   atomic.Pointer[rovel.Overlay]
	dragging bool
	anchor   geom.Point
}

func (t *titleBarView) ID() rovel.ID           { return t.id }
func (t *titleBarView) MinSize() geom.Size     { return geom.Size{W: 1, H: 1} }
func (t *titleBarView) Rect() geom.Rect        { return t.rect }
func (t *titleBarView) Layout(r geom.Rect)     { t.rect = r }
func (t *titleBarView) Children() []rovel.View { return nil }

func (t *titleBarView) Paint(d rovel.Drawer, ctx *rovel.Ctx) {
	d.DrawText(geom.Point{X: t.rect.X, Y: t.rect.Y}, t.title, rovel.Style{})
}

func (t *titleBarView) Handle(e rovel.Event, ctx *rovel.Ctx) bool {
	me, ok := e.(rovel.MouseEvent)
	if !ok {
		return false
	}

	switch me.Action {
	case rovel.MousePress:
		if t.raiseID != 0 && ctx != nil && ctx.RaiseOverlay != nil {
			t.raised.Store(ctx.RaiseOverlay(t.raiseID))
		}

		t.dragging = true
		t.anchor = geom.Point{X: me.X, Y: me.Y}

		return true
	case rovel.MouseDrag:
		if !t.dragging {
			return false
		}

		t.place.MoveBy(me.X-t.anchor.X, me.Y-t.anchor.Y)
		t.anchor = geom.Point{X: me.X, Y: me.Y}
		ctx.InvalidateLayout()

		return true
	case rovel.MouseRelease:
		t.dragging = false

		return true
	}

	return false
}

// bodyView fills its rect with 'B'.
type bodyView struct {
	id   rovel.ID
	rect geom.Rect
}

func (b *bodyView) ID() rovel.ID           { return b.id }
func (b *bodyView) MinSize() geom.Size     { return geom.Size{W: 1, H: 1} }
func (b *bodyView) Rect() geom.Rect        { return b.rect }
func (b *bodyView) Layout(r geom.Rect)     { b.rect = r }
func (b *bodyView) Children() []rovel.View { return nil }

func (b *bodyView) Paint(d rovel.Drawer, ctx *rovel.Ctx) {
	for y := b.rect.Y; y < b.rect.Y+b.rect.H; y++ {
		d.DrawText(geom.Point{X: b.rect.X, Y: y}, repeatRune('B', b.rect.W), rovel.Style{})
	}
}

func (b *bodyView) Handle(e rovel.Event, ctx *rovel.Ctx) bool { return false }

// windowView composes title bar + body: title bar occupies the top row.
type windowView struct {
	id       rovel.ID
	rect     geom.Rect
	titleBar *titleBarView
	body     *bodyView
	keyCount atomic.Int32
}

func newWindowView(title string, place *overlay.Floating) *windowView {
	return &windowView{
		id:       rovel.NewID(),
		titleBar: &titleBarView{id: rovel.NewID(), title: title, place: place},
		body:     &bodyView{id: rovel.NewID()},
	}
}

func (w *windowView) ID() rovel.ID       { return w.id }
func (w *windowView) MinSize() geom.Size { return geom.Size{W: 4, H: 2} }
func (w *windowView) Rect() geom.Rect    { return w.rect }

func (w *windowView) Layout(r geom.Rect) {
	w.rect = r
	w.titleBar.Layout(geom.Rect{X: r.X, Y: r.Y, W: r.W, H: 1})
	w.body.Layout(geom.Rect{X: r.X, Y: r.Y + 1, W: r.W, H: max(r.H-1, 0)})
}

func (w *windowView) Children() []rovel.View { return []rovel.View{w.titleBar, w.body} }

func (w *windowView) Paint(d rovel.Drawer, ctx *rovel.Ctx) {
	w.titleBar.Paint(d, ctx)
	w.body.Paint(d, ctx)
}

func (w *windowView) Handle(e rovel.Event, ctx *rovel.Ctx) bool {
	if _, ok := e.(rovel.KeyEvent); ok {
		w.keyCount.Add(1)

		return true
	}

	return w.titleBar.Handle(e, ctx)
}

// runWindowApp starts an app with a desktop root and one floating window,
// returns the backend and the window's placement. quit stops the app.
func runWindowApp(t *testing.T, screen geom.Size, winRect geom.Rect) (be *memory.Backend, place *overlay.Floating) {
	t.Helper()

	be = memory.New(screen)
	caps := style.Capability{HasBasic: true}

	app, err := rovel.New(rovel.AppOpts{Backend: be, Capability: &caps, Theme: rovel.DefaultTheme()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	app.SetRoot(newDesktopView())

	if err := app.Enable(); err != nil {
		t.Fatalf("Enable: %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- app.Run() }()

	t.Cleanup(func() {
		app.Quit()
		be.Close()

		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("timeout waiting for Run to return")
		}
	})

	place = &overlay.Floating{Rect: winRect}

	err = app.Post(func(ctx *rovel.UpdateCtx) {
		ctx.ShowOverlay(rovel.OverlayOpts{Root: newWindowView("WIN", place), Place: place})
	})
	if err != nil {
		t.Fatalf("Post ShowOverlay: %v", err)
	}

	return be, place
}

func waitForFrames(t *testing.T, be *memory.Backend, count int) {
	t.Helper()

	deadline := time.After(2 * time.Second)

	for be.FrameCount() < count {
		select {
		case <-deadline:
			t.Fatalf("timeout waiting for %d frames, got %d", count, be.FrameCount())
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}
}

func frameRuneAt(t *testing.T, be *memory.Backend, x, y int) rune {
	t.Helper()

	frame, ok := be.LastFrame()
	if !ok {
		t.Fatal("expected a frame")
	}

	return frame.RowRunes(y)[x]
}

// waitRuneAt polls the last frame until the cell shows the wanted rune.
func waitRuneAt(t *testing.T, be *memory.Backend, x, y int, want rune, what string) {
	t.Helper()

	deadline := time.After(2 * time.Second)

	for {
		if got := frameRuneAt(t, be, x, y); got == want {
			return
		}

		select {
		case <-deadline:
			t.Fatalf("timeout waiting for %q at (%d,%d), got %q", what, x, y, frameRuneAt(t, be, x, y))
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}
}

func pressDragRelease(be *memory.Backend, pressX, pressY int, drags [][2]int) {
	be.SendEvent(event.MouseEvent{X: pressX, Y: pressY, Button: event.MouseButtonLeft, Action: event.MousePress})

	for _, d := range drags {
		be.SendEvent(event.MouseEvent{X: pressX + d[0], Y: pressY + d[1], Button: event.MouseButtonLeft, Action: event.MouseMove})
	}

	be.SendEvent(event.MouseEvent{X: pressX, Y: pressY, Button: event.MouseButtonLeft, Action: event.MouseRelease})
}

func TestMovableWindow_DragMovesWindow(t *testing.T) {
	screen := geom.Size{W: 40, H: 12}
	winRect := geom.Rect{X: 2, Y: 2, W: 10, H: 5}

	be, _ := runWindowApp(t, screen, winRect)
	waitForFrames(t, be, 2)

	if got := frameRuneAt(t, be, 2, 2); got != 'W' {
		t.Fatalf("initial title rune at (2,2) = %q, want 'W'", got)
	}

	frames := be.FrameCount()
	pressDragRelease(be, 3, 2, [][2]int{{5, 2}})
	waitForFrames(t, be, frames+1)

	// Title moved by the drag delta.
	if got := frameRuneAt(t, be, 7, 4); got != 'W' {
		t.Fatalf("title rune after drag at (7,4) = %q, want 'W'", got)
	}

	// Old area restored to desktop — no trails.
	if got := frameRuneAt(t, be, 2, 2); got != '.' {
		t.Fatalf("old title cell (2,2) = %q, want '.' (desktop restored)", got)
	}

	// Body moved with the window.
	if got := frameRuneAt(t, be, 7, 5); got != 'B' {
		t.Fatalf("body rune after drag at (7,5) = %q, want 'B'", got)
	}
}

func TestMovableWindow_DragContinuesPastViewRect(t *testing.T) {
	screen := geom.Size{W: 40, H: 12}
	winRect := geom.Rect{X: 2, Y: 2, W: 10, H: 5}

	be, _ := runWindowApp(t, screen, winRect)
	waitForFrames(t, be, 2)

	// Press the title bar, then drag the pointer down 4 rows. The title bar
	// is only 1 cell tall at y=2, so the drag position is outside it (on the
	// body). Without the implicit grab, re-hit-testing would hand the event to
	// the body/desktop and the window would never move.
	frames := be.FrameCount()
	pressDragRelease(be, 3, 2, [][2]int{{0, 4}})
	waitForFrames(t, be, frames+1)

	if got := frameRuneAt(t, be, 2, 6); got != 'W' {
		t.Fatalf("title rune after off-titlebar drag at (2,6) = %q, want 'W'", got)
	}
}

func TestMovableWindow_ClampsToScreen(t *testing.T) {
	screen := geom.Size{W: 40, H: 12}
	winRect := geom.Rect{X: 2, Y: 2, W: 10, H: 5}

	be, _ := runWindowApp(t, screen, winRect)
	waitForFrames(t, be, 2)

	// Drag far past the bottom-right corner: clamped, still fully visible.
	frames := be.FrameCount()
	pressDragRelease(be, 3, 2, [][2]int{{200, 200}})
	waitForFrames(t, be, frames+1)

	wantX, wantY := screen.W-winRect.W, screen.H-winRect.H

	if got := frameRuneAt(t, be, wantX, wantY); got != 'W' {
		t.Fatalf("title rune at clamp (%d,%d) = %q, want 'W'", wantX, wantY, got)
	}

	if got := frameRuneAt(t, be, screen.W-1, screen.H-1); got != 'B' {
		t.Fatalf("bottom-right cell = %q, want 'B' (window fully on screen)", got)
	}

	// Drag far past the top-left corner: clamped to origin.
	frames = be.FrameCount()
	pressDragRelease(be, wantX+1, wantY, [][2]int{{-400, -400}})
	waitForFrames(t, be, frames+1)

	if got := frameRuneAt(t, be, 0, 0); got != 'W' {
		t.Fatalf("title rune at origin = %q, want 'W'", got)
	}
}

func TestMovableWindow_ReleaseEndsGrab(t *testing.T) {
	screen := geom.Size{W: 40, H: 12}
	winRect := geom.Rect{X: 2, Y: 2, W: 10, H: 5}

	be, _ := runWindowApp(t, screen, winRect)
	waitForFrames(t, be, 2)

	// Press title, drag, release — window moves to (7,4).
	pressDragRelease(be, 3, 2, [][2]int{{5, 2}})

	start := time.Now()

	for {
		frame, _ := be.LastFrame()
		if frame.RowRunes(4)[7] == 'W' {
			break
		}

		if time.Since(start) > 2*time.Second {
			t.Fatal("timeout waiting for window to move to (7,4)")
		}

		time.Sleep(5 * time.Millisecond)
	}

	frames := be.FrameCount()

	// A later press+drag on the desktop where the title bar used to be hits
	// the desktop, not the window: normal hit-testing resumed.
	be.SendEvent(event.MouseEvent{X: 3, Y: 2, Button: event.MouseButtonLeft, Action: event.MousePress})
	be.SendEvent(event.MouseEvent{X: 8, Y: 4, Button: event.MouseButtonLeft, Action: event.MouseMove})
	be.SendEvent(event.MouseEvent{X: 8, Y: 4, Button: event.MouseButtonLeft, Action: event.MouseRelease})

	// No repaint is expected (desktop ignores mouse); give the loop a moment,
	// then assert the window did not move.
	time.Sleep(150 * time.Millisecond)

	if got := frameRuneAt(t, be, 7, 4); got != 'W' {
		t.Fatalf("title rune after desktop drag = %q at (7,4), want 'W' (window must not move)", got)
	}

	if be.FrameCount() > frames {
		t.Errorf("desktop drag caused %d unexpected repaints", be.FrameCount()-frames)
	}
}

func TestMovableWindow_DismissDuringDragDropsGrab(t *testing.T) {
	screen := geom.Size{W: 40, H: 12}

	be := memory.New(screen)
	caps := style.Capability{HasBasic: true}

	app, err := rovel.New(rovel.AppOpts{Backend: be, Capability: &caps, Theme: rovel.DefaultTheme()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	app.SetRoot(newDesktopView())

	if err := app.Enable(); err != nil {
		t.Fatalf("Enable: %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- app.Run() }()

	t.Cleanup(func() {
		app.Quit()
		be.Close()

		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("timeout waiting for Run to return")
		}
	})

	place := &overlay.Floating{Rect: geom.Rect{X: 2, Y: 2, W: 10, H: 5}}

	err = app.Post(func(ctx *rovel.UpdateCtx) {
		ctx.ShowOverlay(rovel.OverlayOpts{Root: newWindowView("WIN", place), Place: place})
	})
	if err != nil {
		t.Fatalf("Post ShowOverlay: %v", err)
	}

	// Wait for the window to actually paint before pressing it.
	for deadline := time.Now().Add(2 * time.Second); frameRuneAt(t, be, 2, 2) != 'W'; {
		if time.Now().After(deadline) {
			t.Fatal("timeout waiting for the window to paint")
		}

		time.Sleep(5 * time.Millisecond)
	}

	// Press the title bar and drag once, so the grab is proven live before
	// the dismissal (Post and event delivery race otherwise).
	be.SendEvent(event.MouseEvent{X: 3, Y: 2, Button: event.MouseButtonLeft, Action: event.MousePress})
	be.SendEvent(event.MouseEvent{X: 4, Y: 3, Button: event.MouseButtonLeft, Action: event.MouseMove})

	for deadline := time.Now().Add(2 * time.Second); frameRuneAt(t, be, 3, 3) != 'W'; {
		if time.Now().After(deadline) {
			t.Fatal("timeout waiting for the drag to move the window (grab not live)")
		}

		time.Sleep(5 * time.Millisecond)
	}

	// Now dismiss the overlay mid-drag, with the button still held.
	dismissed := make(chan struct{})

	err = app.Post(func(ctx *rovel.UpdateCtx) {
		ctx.DismissOverlay()
		close(dismissed)
	})
	if err != nil {
		t.Fatalf("Post DismissOverlay: %v", err)
	}

	select {
	case <-dismissed:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for dismissal")
	}

	// Desktop shows through where the window was.
	for deadline := time.Now().Add(2 * time.Second); frameRuneAt(t, be, 3, 3) != '.'; {
		if time.Now().After(deadline) {
			t.Fatal("timeout waiting for desktop to be restored after dismissal")
		}

		time.Sleep(5 * time.Millisecond)
	}

	// The drag must not reach the unmounted title bar.
	be.SendEvent(event.MouseEvent{X: 20, Y: 8, Button: event.MouseButtonLeft, Action: event.MouseMove})
	be.SendEvent(event.MouseEvent{X: 20, Y: 8, Button: event.MouseButtonLeft, Action: event.MouseRelease})
	time.Sleep(150 * time.Millisecond)

	if got := (geom.Rect{X: 3, Y: 3, W: 10, H: 5}); place.Rect != got {
		t.Fatalf("placement moved after dismissal: %v, want %v (grab must be dropped)", place.Rect, got)
	}
}

// showWindowAt shows one floating window whose title bar raises on press.
// The returned channel delivers the overlay ID once shown (race-free read
// from the test goroutine).
func showWindowAt(t *testing.T, app *rovel.App, title string, rect geom.Rect) (*windowView, *overlay.Floating, <-chan rovel.ID) {
	t.Helper()

	place := &overlay.Floating{Rect: rect}
	win := newWindowView(title, place)
	idCh := make(chan rovel.ID, 1)

	err := app.Post(func(ctx *rovel.UpdateCtx) {
		o := ctx.ShowOverlay(rovel.OverlayOpts{Root: win, Place: place})
		win.titleBar.raiseID = o.ID()
		idCh <- o.ID()
	})
	if err != nil {
		t.Fatalf("Post ShowOverlay: %v", err)
	}

	return win, place, idCh
}

// recvID receives an overlay ID from a showWindowAt channel.
func recvID(t *testing.T, ch <-chan rovel.ID) rovel.ID {
	t.Helper()

	select {
	case id := <-ch:
		return id
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for overlay id")
		return 0
	}
}

func TestMovableWindow_RaiseOnClick(t *testing.T) {
	be := memory.New(geom.Size{W: 40, H: 12})
	caps := style.Capability{HasBasic: true}

	app, err := rovel.New(rovel.AppOpts{Backend: be, Capability: &caps, Theme: rovel.DefaultTheme()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	app.SetRoot(newDesktopView())

	if err := app.Enable(); err != nil {
		t.Fatalf("Enable: %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- app.Run() }()

	t.Cleanup(func() {
		app.Quit()
		be.Close()

		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("timeout waiting for Run to return")
		}
	})

	// Back window below, front window above, overlapping. onDismiss fires on
	// dismissal only — raising must never trigger it.
	var dismissals atomic.Int32

	backPlace := &overlay.Floating{Rect: geom.Rect{X: 2, Y: 2, W: 10, H: 5}}
	back := newWindowView("BACK", backPlace)
	backIDCh := make(chan rovel.ID, 1)

	err = app.Post(func(ctx *rovel.UpdateCtx) {
		o := ctx.ShowOverlay(rovel.OverlayOpts{Root: back, Place: backPlace, OnDismiss: func() { dismissals.Add(1) }})
		back.titleBar.raiseID = o.ID()
		backIDCh <- o.ID()
	})
	if err != nil {
		t.Fatalf("Post back: %v", err)
	}

	front, _, _ := showWindowAt(t, app, "FRNT", geom.Rect{X: 6, Y: 3, W: 10, H: 5})
	backID := recvID(t, backIDCh)

	// Overlap cell (7,3) is on the front window's title row while front is
	// on top ("FRNT" starts at x=6, so (7,3) is its 'R').
	waitRuneAt(t, be, 7, 3, 'R', "front window on top before raise")

	// Press the back window's exposed title bar cell (3,2): front starts at
	// x=6, so this cell belongs to the back window only.
	be.SendEvent(event.MouseEvent{X: 3, Y: 2, Button: event.MouseButtonLeft, Action: event.MousePress})
	be.SendEvent(event.MouseEvent{X: 3, Y: 2, Button: event.MouseButtonLeft, Action: event.MouseRelease})

	// Back window now paints over the front one.
	waitRuneAt(t, be, 7, 3, 'B', "back window raised above front")

	// Identity preserved: RaiseOverlay returned the same overlay ID.
	if raised := back.titleBar.raised.Load(); raised == nil || raised.ID() != backID {
		t.Fatalf("RaiseOverlay returned %v, want overlay %d", raised, backID)
	}

	// No dismiss side effect fired.
	if n := dismissals.Load(); n != 0 {
		t.Fatalf("onDismiss fired %d times during raise, want 0", n)
	}

	// Keys now route to the raised window, not the covered one.
	be.SendEvent(event.KeyEvent{Key: event.KeyRune, Rune: 'k'})

	deadline := time.After(2 * time.Second)

	for back.keyCount.Load() == 0 {
		select {
		case <-deadline:
			t.Fatal("timeout waiting for key on raised window")
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}

	if n := front.keyCount.Load(); n != 0 {
		t.Fatalf("covered window received %d key events, want 0", n)
	}
}

func TestMovableWindow_ModalBlocksAfterRaise(t *testing.T) {
	be := memory.New(geom.Size{W: 40, H: 12})
	caps := style.Capability{HasBasic: true}

	app, err := rovel.New(rovel.AppOpts{Backend: be, Capability: &caps, Theme: rovel.DefaultTheme()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	app.SetRoot(newDesktopView())

	if err := app.Enable(); err != nil {
		t.Fatalf("Enable: %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- app.Run() }()

	t.Cleanup(func() {
		app.Quit()
		be.Close()

		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("timeout waiting for Run to return")
		}
	})

	back, _, backIDCh := showWindowAt(t, app, "BACK", geom.Rect{X: 2, Y: 2, W: 10, H: 5})
	showWindowAt(t, app, "FRNT", geom.Rect{X: 6, Y: 3, W: 10, H: 5})

	// Lower modal, overlapping the back window: a raise must stop below it,
	// not merely below the topmost modal.
	lowModalPlace := &overlay.Floating{Rect: geom.Rect{X: 4, Y: 4, W: 10, H: 3}}
	lowModal := newWindowView("MDLA", lowModalPlace)

	err = app.Post(func(ctx *rovel.UpdateCtx) {
		ctx.ShowOverlay(rovel.OverlayOpts{Root: lowModal, Place: lowModalPlace, Modal: true})
	})
	if err != nil {
		t.Fatalf("Post lower modal: %v", err)
	}

	// Topmost modal dialog at the bottom-right, away from both windows.
	modalPlace := &overlay.Floating{Rect: geom.Rect{X: 28, Y: 8, W: 10, H: 3}}
	modal := newWindowView("MODL", modalPlace)

	err = app.Post(func(ctx *rovel.UpdateCtx) {
		ctx.ShowOverlay(rovel.OverlayOpts{Root: modal, Place: modalPlace, Modal: true})
	})
	if err != nil {
		t.Fatalf("Post modal: %v", err)
	}

	// Wait for all three overlays to land.
	waitRuneAt(t, be, 7, 3, 'R', "front window on top before modal click")

	// A click on the back window's exposed title bar never arrives: the modal
	// blocks all clicks to lower overlays, even on a miss.
	be.SendEvent(event.MouseEvent{X: 3, Y: 2, Button: event.MouseButtonLeft, Action: event.MousePress})
	be.SendEvent(event.MouseEvent{X: 3, Y: 2, Button: event.MouseButtonLeft, Action: event.MouseRelease})
	time.Sleep(150 * time.Millisecond)

	if back.titleBar.raised.Load() != nil {
		t.Fatal("click reached a lower overlay while a modal overlay was active")
	}

	if got := frameRuneAt(t, be, 7, 3); got != 'R' {
		t.Fatalf("z-order changed under a modal: overlap cell = %q, want 'R' (front on top)", got)
	}

	// Raise the back window programmatically; the guard keeps it below the
	// modal, so keys still route to the modal.
	backID := recvID(t, backIDCh)

	err = app.Post(func(ctx *rovel.UpdateCtx) {
		ctx.RaiseOverlay(backID)
	})
	if err != nil {
		t.Fatalf("Post raise: %v", err)
	}

	// The guarded raise repaints the overlap with the back window on top of
	// the front one (but below the modal).
	waitRuneAt(t, be, 7, 3, 'B', "back raised above front, below modal")

	// ...but still below the lower modal it was under: the lower modal's
	// title row survives the raise.
	if got := frameRuneAt(t, be, 5, 4); got != 'D' {
		t.Fatalf("raise crossed a lower modal: cell (5,4) = %q, want 'D' (MDLA on top)", got)
	}

	be.SendEvent(event.KeyEvent{Key: event.KeyRune, Rune: 'k'})

	deadline := time.After(2 * time.Second)

	for modal.keyCount.Load() == 0 {
		select {
		case <-deadline:
			t.Fatal("timeout waiting for key on modal")
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}

	if n := back.keyCount.Load(); n != 0 {
		t.Fatalf("raised non-modal window received %d keys under a modal, want 0", n)
	}
}
