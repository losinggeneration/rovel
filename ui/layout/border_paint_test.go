package layout

import (
	"testing"

	"github.com/losinggeneration/rovel"
	"github.com/losinggeneration/rovel/geom"
	"github.com/losinggeneration/rovel/render"
	"github.com/losinggeneration/rovel/style"
)

type paintChildView struct {
	id   rovel.ID
	rect rovel.Rect
}

func (v *paintChildView) ID() rovel.ID        { return v.id }
func (v *paintChildView) Rect() rovel.Rect    { return v.rect }
func (v *paintChildView) Layout(r rovel.Rect) { v.rect = r }
func (v *paintChildView) MinSize() rovel.Size { return rovel.Size{W: 1, H: 1} }
func (v *paintChildView) Paint(d rovel.Drawer, _ *rovel.Ctx) {
	d.DrawText(rovel.Point{X: v.rect.X, Y: v.rect.Y}, "X", style.Style{})
}
func (v *paintChildView) Handle(rovel.Event, *rovel.Ctx) bool { return false }

func TestBorderPaint_DrawerAdapter(t *testing.T) {
	child := &paintChildView{id: rovel.NewID()}
	border := NewBorder(child)
	border.SetTitle("Title")
	border.Layout(geom.Rect{X: 0, Y: 0, W: 12, H: 4})

	base := style.Style{}
	buf := render.NewBuffer(12, 4)
	buf.Clear(render.Cell{R: ' ', Style: base})

	rp := render.NewPainter(buf, geom.Rect{X: 0, Y: 0, W: 12, H: 4}, base)
	p := rovel.NewPainter(rp, base)
	ctx := &rovel.Ctx{Theme: rovel.DefaultTheme()}

	border.Paint(rovel.NewDrawer(p), ctx)

	assertRune := func(x, y int, want rune) {
		t.Helper()

		got := buf.At(x, y).R
		if got != want {
			t.Fatalf("cell (%d,%d) = %q, want %q", x, y, got, want)
		}
	}

	assertRune(0, 0, '+')
	assertRune(11, 0, '+')
	assertRune(0, 3, '+')
	assertRune(11, 3, '+')
	assertRune(2, 0, 'T')
	assertRune(6, 0, 'e')
	assertRune(1, 1, 'X')
}
