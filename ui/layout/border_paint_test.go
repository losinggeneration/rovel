package layout

import (
	"testing"

	"github.com/losinggeneration/tui"
	"github.com/losinggeneration/tui/geom"
	"github.com/losinggeneration/tui/render"
	"github.com/losinggeneration/tui/style"
)

type paintChildView struct {
	id   tui.ID
	rect tui.Rect
}

func (v *paintChildView) ID() tui.ID        { return v.id }
func (v *paintChildView) Rect() tui.Rect    { return v.rect }
func (v *paintChildView) Layout(r tui.Rect) { v.rect = r }
func (v *paintChildView) MinSize() tui.Size { return tui.Size{W: 1, H: 1} }
func (v *paintChildView) Paint(p *tui.Painter, _ *tui.Ctx) {
	p.Text(v.rect.X, v.rect.Y, "X", style.Style{})
}
func (v *paintChildView) Handle(tui.Event, *tui.Ctx) bool { return false }

func TestBorderPaint_DrawerAdapter(t *testing.T) {
	child := &paintChildView{id: tui.NewID()}
	border := NewBorder(child)
	border.SetTitle("Title")
	border.Layout(geom.Rect{X: 0, Y: 0, W: 12, H: 4})

	base := style.Style{}
	buf := render.NewBuffer(12, 4)
	buf.Clear(render.Cell{R: ' ', Style: base})

	rp := render.NewPainter(buf, geom.Rect{X: 0, Y: 0, W: 12, H: 4}, base)
	p := tui.NewPainter(rp, base)
	ctx := &tui.Ctx{Theme: tui.DefaultTheme()}

	border.Paint(p, ctx)

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
