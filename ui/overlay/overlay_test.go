package overlay

import (
	"testing"

	"github.com/losinggeneration/rovel"
	"github.com/losinggeneration/rovel/geom"
)

type stubView struct {
	id      rovel.ID
	minSize geom.Size
	pref    geom.Size
	rect    geom.Rect
}

func (v *stubView) ID() rovel.ID       { return v.id }
func (v *stubView) MinSize() geom.Size { return v.minSize }
func (v *stubView) PreferredSize() geom.Size {
	if v.pref != (geom.Size{}) {
		return v.pref
	}

	return v.minSize
}
func (v *stubView) Layout(r geom.Rect)                        { v.rect = r }
func (v *stubView) Rect() geom.Rect                           { return v.rect }
func (v *stubView) Paint(d rovel.Drawer, ctx *rovel.Ctx)      {}
func (v *stubView) Handle(e rovel.Event, ctx *rovel.Ctx) bool { return false }

func TestCentered(t *testing.T) {
	view := &stubView{minSize: geom.Size{W: 20, H: 10}}
	screen := geom.Size{W: 80, H: 24}

	got := Centered{}.Resolve(view, screen)

	want := geom.Rect{X: 30, Y: 7, W: 20, H: 10}
	if got != want {
		t.Errorf("Centered.Resolve = %v, want %v", got, want)
	}
}

func TestCentered_ClampedToScreen(t *testing.T) {
	view := &stubView{minSize: geom.Size{W: 100, H: 30}}
	screen := geom.Size{W: 80, H: 24}

	got := Centered{}.Resolve(view, screen)
	if got.W != 80 || got.H != 24 {
		t.Errorf("Centered should clamp to screen: got %v", got)
	}
}

func TestAnchored_Below(t *testing.T) {
	view := &stubView{minSize: geom.Size{W: 15, H: 5}}
	screen := geom.Size{W: 80, H: 24}
	anchor := geom.Rect{X: 10, Y: 3, W: 15, H: 1}

	got := Anchored{Anchor: anchor}.Resolve(view, screen)
	// Should appear below anchor
	if got.Y != anchor.Y+anchor.H {
		t.Errorf("expected Y=%d (below anchor), got Y=%d", anchor.Y+anchor.H, got.Y)
	}

	if got.X != anchor.X {
		t.Errorf("expected X=%d, got X=%d", anchor.X, got.X)
	}
}

func TestAnchored_FlipsAbove(t *testing.T) {
	view := &stubView{minSize: geom.Size{W: 15, H: 5}}
	screen := geom.Size{W: 80, H: 24}
	// Anchor near bottom — not enough room below.
	anchor := geom.Rect{X: 10, Y: 20, W: 15, H: 1}

	got := Anchored{Anchor: anchor}.Resolve(view, screen)
	// Should appear above anchor
	if got.Y != anchor.Y-5 {
		t.Errorf("expected Y=%d (above anchor), got Y=%d", anchor.Y-5, got.Y)
	}
}

func TestAnchored_ClampsX(t *testing.T) {
	view := &stubView{minSize: geom.Size{W: 20, H: 3}}
	screen := geom.Size{W: 80, H: 24}
	// Anchor near right edge.
	anchor := geom.Rect{X: 70, Y: 5, W: 5, H: 1}

	got := Anchored{Anchor: anchor}.Resolve(view, screen)
	if got.X+got.W > screen.W {
		t.Errorf("overlay extends beyond screen: X=%d, W=%d, screenW=%d", got.X, got.W, screen.W)
	}
}

func TestFullscreen(t *testing.T) {
	view := &stubView{minSize: geom.Size{W: 10, H: 5}}
	screen := geom.Size{W: 80, H: 24}

	got := Fullscreen{}.Resolve(view, screen)

	want := geom.Rect{X: 0, Y: 0, W: 80, H: 24}
	if got != want {
		t.Errorf("Fullscreen.Resolve = %v, want %v", got, want)
	}
}

func TestTopCentered_UsesPreferredSizeByDefault(t *testing.T) {
	view := &stubView{minSize: geom.Size{W: 5, H: 2}, pref: geom.Size{W: 20, H: 3}}
	screen := geom.Size{W: 80, H: 24}

	got := TopCentered{}.Resolve(view, screen)
	want := geom.Rect{X: 30, Y: 0, W: 20, H: 3}
	if got != want {
		t.Fatalf("TopCentered.Resolve = %v, want %v", got, want)
	}
}

func TestTopCentered_ClampsExplicitSize(t *testing.T) {
	view := &stubView{minSize: geom.Size{W: 5, H: 2}}
	screen := geom.Size{W: 80, H: 24}

	got := TopCentered{W: 100, H: 30}.Resolve(view, screen)
	want := geom.Rect{X: 0, Y: 0, W: 80, H: 24}
	if got != want {
		t.Fatalf("TopCentered.Resolve = %v, want %v", got, want)
	}
}

func TestPointAnchored_DefaultBelowRight(t *testing.T) {
	view := &stubView{minSize: geom.Size{W: 8, H: 3}}
	screen := geom.Size{W: 80, H: 24}

	got := PointAnchored{Anchor: geom.Point{X: 10, Y: 5}}.Resolve(view, screen)
	want := geom.Rect{X: 10, Y: 5, W: 8, H: 3}
	if got != want {
		t.Fatalf("PointAnchored.Resolve = %v, want %v", got, want)
	}
}

func TestPointAnchored_FlipsNearEdges(t *testing.T) {
	view := &stubView{minSize: geom.Size{W: 12, H: 4}}
	screen := geom.Size{W: 80, H: 24}

	got := PointAnchored{
		Anchor: geom.Point{X: 78, Y: 23},
		Flip:   true,
	}.Resolve(view, screen)

	want := geom.Rect{X: 66, Y: 19, W: 12, H: 4}
	if got != want {
		t.Fatalf("PointAnchored.Resolve = %v, want %v", got, want)
	}
}

func TestPointAnchored_ClampsWithoutFlip(t *testing.T) {
	view := &stubView{minSize: geom.Size{W: 12, H: 4}}
	screen := geom.Size{W: 80, H: 24}

	got := PointAnchored{
		Anchor: geom.Point{X: 78, Y: 23},
	}.Resolve(view, screen)

	want := geom.Rect{X: 68, Y: 20, W: 12, H: 4}
	if got != want {
		t.Fatalf("PointAnchored.Resolve = %v, want %v", got, want)
	}
}

func TestFloating_ReturnsStoredRect(t *testing.T) {
	view := &stubView{minSize: geom.Size{W: 12, H: 4}}
	screen := geom.Size{W: 80, H: 24}
	want := geom.Rect{X: 5, Y: 7, W: 20, H: 6}

	f := &Floating{Rect: want}
	if got := f.Resolve(view, screen); got != want {
		t.Fatalf("Floating.Resolve = %v, want %v", got, want)
	}
}

func TestFloating_ClampsToScreen(t *testing.T) {
	view := &stubView{minSize: geom.Size{W: 1, H: 1}}
	screen := geom.Size{W: 40, H: 12}

	f := &Floating{Rect: geom.Rect{X: 100, Y: 100, W: 10, H: 5}}
	got := f.Resolve(view, screen)
	want := geom.Rect{X: 30, Y: 7, W: 10, H: 5}
	if got != want {
		t.Fatalf("Floating.Resolve = %v, want %v", got, want)
	}

	f.MoveBy(-500, -500)
	got = f.Resolve(view, screen)
	want = geom.Rect{X: 0, Y: 0, W: 10, H: 5}
	if got != want {
		t.Fatalf("Floating.Resolve after MoveBy = %v, want %v", got, want)
	}

	f.MoveBy(3, 4)
	if f.Rect.X != 3 || f.Rect.Y != 4 {
		t.Fatalf("Floating.MoveBy = %v, want origin (3,4)", f.Rect)
	}
}
