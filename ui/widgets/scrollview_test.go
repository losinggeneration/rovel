package widgets

import (
	"testing"

	"github.com/losinggeneration/tui"
	"github.com/losinggeneration/tui/event"
	"github.com/losinggeneration/tui/geom"
	"github.com/losinggeneration/tui/render"
	"github.com/losinggeneration/tui/style"
)

type mockView struct {
	id          tui.ID
	minSize     geom.Size
	rect        geom.Rect
	focusable   bool
	paintCalled int
}

func (m *mockView) ID() tui.ID         { return m.id }
func (m *mockView) Rect() geom.Rect    { return m.rect }
func (m *mockView) MinSize() geom.Size { return m.minSize }
func (m *mockView) Focusable() bool    { return m.focusable }
func (m *mockView) Layout(r geom.Rect) { m.rect = r }
func (m *mockView) Paint(p *tui.Painter, ctx *tui.Ctx) {
	m.paintCalled++
}
func (m *mockView) Handle(e tui.Event, ctx *tui.Ctx) bool { return false }

func TestScrollView_New(t *testing.T) {
	child := &mockView{id: tui.NewID(), minSize: geom.Size{W: 10, H: 50}}
	sv := NewScrollView(ScrollViewOpts{
		Child:     child,
		Focusable: true,
	})

	if sv.ID() == 0 {
		t.Error("expected non-zero ID")
	}

	if !sv.Focusable() {
		t.Error("expected Focusable to be true")
	}
}

func TestScrollView_MinSize(t *testing.T) {
	child := &mockView{id: tui.NewID(), minSize: geom.Size{W: 10, H: 50}}
	sv := NewScrollView(ScrollViewOpts{
		Child:     child,
		Focusable: false,
	})

	minSize := sv.MinSize()
	if minSize.W != 10 || minSize.H != 50 {
		t.Errorf("expected MinSize{10, 50}, got %v", minSize)
	}
}

func TestScrollView_Layout(t *testing.T) {
	child := &mockView{id: tui.NewID(), minSize: geom.Size{W: 10, H: 50}}
	sv := NewScrollView(ScrollViewOpts{
		Child:     child,
		Focusable: false,
		Scrollbar: ScrollbarHidden,
	})

	rect := geom.Rect{X: 5, Y: 10, W: 20, H: 15}
	sv.Layout(rect)

	if sv.Rect() != rect {
		t.Errorf("expected rect %v, got %v", rect, sv.Rect())
	}

	if child.Rect() != rect {
		t.Errorf("expected child rect %v, got %v", rect, child.Rect())
	}
}

func TestScrollView_clampScroll(t *testing.T) {
	child := &mockView{id: tui.NewID(), minSize: geom.Size{W: 10, H: 50}}
	sv := NewScrollView(ScrollViewOpts{
		Child:     child,
		Focusable: false,
	})

	// Layout with viewport smaller than content
	sv.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 10})

	// Content is 50 rows, viewport is 10, so max scroll should be 40
	if sv.maxScrollY() != 40 {
		t.Errorf("expected maxScrollY 40, got %d", sv.maxScrollY())
	}

	// Scroll past bounds should be clamped
	sv.scrollY = 100
	sv.clampScroll()

	if sv.scrollY != 40 {
		t.Errorf("expected scrollY clamped to 40, got %d", sv.scrollY)
	}

	// Negative should clamp to 0
	sv.scrollY = -5
	sv.clampScroll()

	if sv.scrollY != 0 {
		t.Errorf("expected scrollY clamped to 0, got %d", sv.scrollY)
	}
}

func TestScrollView_ScrollTo(t *testing.T) {
	child := &mockView{id: tui.NewID(), minSize: geom.Size{W: 10, H: 50}}
	sv := NewScrollView(ScrollViewOpts{
		Child:     child,
		Focusable: false,
	})

	sv.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 10})

	// Track invalidation
	var invalidated bool

	ctx := &tui.Ctx{
		Invalidate: func(r geom.Rect) {
			if r == sv.Rect() {
				invalidated = true
			}
		},
	}

	// Scroll to middle
	invalidated = false

	sv.ScrollTo(ctx, 20)

	if sv.scrollY != 20 {
		t.Errorf("expected scrollY 20, got %d", sv.scrollY)
	}

	if !invalidated {
		t.Error("expected invalidation on scroll change")
	}

	// Scroll to same position should not invalidate
	invalidated = false

	sv.ScrollTo(ctx, 20)

	if invalidated {
		t.Error("should not invalidate when scroll position unchanged")
	}

	// Scroll past bounds should clamp
	sv.ScrollTo(ctx, 100)

	if sv.scrollY != 40 {
		t.Errorf("expected scrollY clamped to 40, got %d", sv.scrollY)
	}
}

func TestScrollView_ScrollBy(t *testing.T) {
	child := &mockView{id: tui.NewID(), minSize: geom.Size{W: 10, H: 50}}
	sv := NewScrollView(ScrollViewOpts{
		Child:     child,
		Focusable: false,
	})

	sv.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 10})

	ctx := &tui.Ctx{
		Invalidate: func(r geom.Rect) {},
	}

	sv.scrollY = 10
	sv.ScrollBy(ctx, 5)

	if sv.scrollY != 15 {
		t.Errorf("expected scrollY 15, got %d", sv.scrollY)
	}

	sv.ScrollBy(ctx, -20)

	if sv.scrollY != 0 {
		t.Errorf("expected scrollY 0 after negative scroll, got %d", sv.scrollY)
	}
}

func TestScrollView_Handle(t *testing.T) {
	child := &mockView{id: tui.NewID(), minSize: geom.Size{W: 10, H: 50}}
	sv := NewScrollView(ScrollViewOpts{
		Child:     child,
		Focusable: true,
	})

	sv.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 10})

	ctx := &tui.Ctx{
		Invalidate: func(r geom.Rect) {},
	}

	// Test KeyUp
	sv.scrollY = 10
	sv.Handle(event.KeyEvent{Key: event.KeyUp}, ctx)

	if sv.scrollY != 9 {
		t.Errorf("expected scrollY 9 after KeyUp, got %d", sv.scrollY)
	}

	// Test KeyDown
	sv.scrollY = 10
	sv.Handle(event.KeyEvent{Key: event.KeyDown}, ctx)

	if sv.scrollY != 11 {
		t.Errorf("expected scrollY 11 after KeyDown, got %d", sv.scrollY)
	}

	// Test KeyPageUp
	sv.scrollY = 20
	sv.Handle(event.KeyEvent{Key: event.KeyPageUp}, ctx)
	// page up = -(H-1) = -9, so 20 - 9 = 11
	if sv.scrollY != 11 {
		t.Errorf("expected scrollY 11 after KeyPageUp, got %d", sv.scrollY)
	}

	// Test KeyPageDown
	sv.scrollY = 20
	sv.Handle(event.KeyEvent{Key: event.KeyPageDown}, ctx)
	// page down = H-1 = 9, so 20 + 9 = 29
	if sv.scrollY != 29 {
		t.Errorf("expected scrollY 29 after KeyPageDown, got %d", sv.scrollY)
	}

	// Test KeyHome
	sv.scrollY = 20
	sv.Handle(event.KeyEvent{Key: event.KeyHome}, ctx)

	if sv.scrollY != 0 {
		t.Errorf("expected scrollY 0 after KeyHome, got %d", sv.scrollY)
	}

	// Test KeyEnd
	sv.scrollY = 20
	sv.Handle(event.KeyEvent{Key: event.KeyEnd}, ctx)

	if sv.scrollY != 40 {
		t.Errorf("expected scrollY 40 after KeyEnd, got %d", sv.scrollY)
	}

	// Test non-scroll key
	sv.scrollY = 10

	handled := sv.Handle(event.KeyEvent{Key: event.KeyTab}, ctx)
	if handled {
		t.Error("expected KeyTab to not be handled")
	}

	if sv.scrollY != 10 {
		t.Error("expected scrollY unchanged for unhandled key")
	}
}

func TestScrollView_ScrollTopBottom(t *testing.T) {
	child := &mockView{id: tui.NewID(), minSize: geom.Size{W: 10, H: 50}}
	sv := NewScrollView(ScrollViewOpts{
		Child:     child,
		Focusable: false,
	})

	sv.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 10})

	ctx := &tui.Ctx{
		Invalidate: func(r geom.Rect) {},
	}

	sv.scrollY = 30
	sv.ScrollTop(ctx)

	if sv.scrollY != 0 {
		t.Errorf("expected scrollY 0, got %d", sv.scrollY)
	}

	sv.ScrollBottom(ctx)

	if sv.scrollY != 40 {
		t.Errorf("expected scrollY 40, got %d", sv.scrollY)
	}
}

type paintRowsView struct {
	id   tui.ID
	rect geom.Rect
}

func (v *paintRowsView) ID() tui.ID         { return v.id }
func (v *paintRowsView) Rect() geom.Rect    { return v.rect }
func (v *paintRowsView) MinSize() geom.Size { return geom.Size{W: 1, H: 4} }
func (v *paintRowsView) Layout(r geom.Rect) { v.rect = r }
func (v *paintRowsView) Handle(tui.Event, *tui.Ctx) bool {
	return false
}

func (v *paintRowsView) Paint(p *tui.Painter, _ *tui.Ctx) {
	for i, ch := range []rune{'0', '1', '2', '3'} {
		p.Text(v.rect.X, v.rect.Y+i, string(ch), style.Style{})
	}
}

func TestScrollViewPaint_DrawerAdapterOffsetAndClip(t *testing.T) {
	child := &paintRowsView{id: tui.NewID()}
	sv := NewScrollView(ScrollViewOpts{
		Child:     child,
		Scrollbar: ScrollbarHidden,
	})
	sv.Layout(geom.Rect{X: 0, Y: 0, W: 3, H: 2})
	sv.scrollY = 1

	base := style.Style{}
	buf := render.NewBuffer(3, 2)
	buf.Clear(render.Cell{R: ' ', Style: base})
	rp := render.NewPainter(buf, geom.Rect{X: 0, Y: 0, W: 3, H: 2}, base)
	p := tui.NewPainter(rp, base)

	sv.Paint(p, &tui.Ctx{Theme: tui.DefaultTheme()})

	if got := buf.At(0, 0).R; got != '1' {
		t.Fatalf("row 0 = %q, want %q", got, '1')
	}

	if got := buf.At(0, 1).R; got != '2' {
		t.Fatalf("row 1 = %q, want %q", got, '2')
	}

	if got := buf.At(1, 0).R; got != ' ' {
		t.Fatalf("unexpected write outside text width at (1,0): %q", got)
	}
}

func TestScrollView_contentHeight(t *testing.T) {
	child := &mockView{id: tui.NewID(), minSize: geom.Size{W: 10, H: 50}}
	sv := NewScrollView(ScrollViewOpts{
		Child:     child,
		Focusable: false,
	})

	if sv.contentHeight() != 50 {
		t.Errorf("expected contentHeight 50, got %d", sv.contentHeight())
	}
}

func TestScrollView_noChild(t *testing.T) {
	sv := NewScrollView(ScrollViewOpts{
		Focusable: false,
	})

	minSize := sv.MinSize()
	if minSize.W != 1 || minSize.H != 1 {
		t.Errorf("expected MinSize{1, 1}, got %v", minSize)
	}

	// Should not panic on layout/paint
	sv.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 10})
}
