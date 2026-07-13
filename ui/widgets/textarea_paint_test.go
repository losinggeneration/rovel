package widgets

import (
	"testing"

	"github.com/losinggeneration/rovel"
	"github.com/losinggeneration/rovel/geom"
	"github.com/losinggeneration/rovel/render"
	"github.com/losinggeneration/rovel/style"
)

func TestTextAreaPaint_DrawerAdapterScrollAndCursor(t *testing.T) {
	ta := NewTextArea()
	ta.Layout(geom.Rect{X: 0, Y: 0, W: 4, H: 2})
	ta.SetText(nil, "a\nbc\ndef")
	ta.scrollY = 1
	ta.cursor = ta.lines[2].endByte

	base := style.Style{}
	buf := render.NewBuffer(4, 2)
	buf.Clear(render.Cell{R: ' ', Style: base})
	rp := render.NewPainter(buf, geom.Rect{X: 0, Y: 0, W: 4, H: 2}, base)
	p := rovel.NewPainter(rp, base)
	ctx := &rovel.Ctx{
		Theme:     rovel.DefaultTheme(),
		FocusedID: ta.ID(),
	}

	ta.Paint(rovel.NewDrawer(p), ctx)

	if got := buf.At(0, 0).R; got != 'b' {
		t.Fatalf("row 0 col 0 = %q, want %q", got, 'b')
	}

	if got := buf.At(1, 0).R; got != 'c' {
		t.Fatalf("row 0 col 1 = %q, want %q", got, 'c')
	}

	if got := buf.At(0, 1).R; got != 'd' {
		t.Fatalf("row 1 col 0 = %q, want %q", got, 'd')
	}

	if got := buf.At(2, 1).R; got != 'f' {
		t.Fatalf("row 1 col 2 = %q, want %q", got, 'f')
	}

	focusSt := rovel.DefaultTheme().Palette.Focus
	if got := buf.At(3, 1).Style; got != focusSt {
		t.Fatalf("cursor style = %+v, want %+v", got, focusSt)
	}
}

func TestTextAreaPaint_HorizontalScroll(t *testing.T) {
	ta := NewTextArea()
	ta.Layout(geom.Rect{X: 0, Y: 0, W: 4, H: 1})
	ta.SetText(nil, "0123456789")
	ta.scrollX = 3 // view shows columns 3..6
	ta.cursor = 0  // off-screen; no cursor cell expected in the view

	base := style.Style{}
	buf := render.NewBuffer(4, 1)
	buf.Clear(render.Cell{R: ' ', Style: base})
	rp := render.NewPainter(buf, geom.Rect{X: 0, Y: 0, W: 4, H: 1}, base)
	p := rovel.NewPainter(rp, base)
	ctx := &rovel.Ctx{
		Theme:     rovel.DefaultTheme(),
		FocusedID: ta.ID(),
	}

	ta.Paint(rovel.NewDrawer(p), ctx)

	for i, w := range []rune{'3', '4', '5', '6'} {
		if got := buf.At(i, 0).R; got != w {
			t.Fatalf("col %d = %q, want %q", i, got, w)
		}
	}
}

func TestTextAreaPaint_DrawerAdapterSelectionStyle(t *testing.T) {
	ta := NewTextArea()
	ta.Layout(geom.Rect{X: 0, Y: 0, W: 4, H: 2})
	ta.SetText(nil, "ab\ncd")
	ta.anchor = 0
	ta.cursor = 1

	base := style.Style{}
	buf := render.NewBuffer(4, 2)
	buf.Clear(render.Cell{R: ' ', Style: base})
	rp := render.NewPainter(buf, geom.Rect{X: 0, Y: 0, W: 4, H: 2}, base)
	p := rovel.NewPainter(rp, base)
	ctx := &rovel.Ctx{
		Theme:     rovel.DefaultTheme(),
		FocusedID: ta.ID(),
	}

	ta.Paint(rovel.NewDrawer(p), ctx)

	selectionSt := rovel.DefaultTheme().Palette.Selection
	if got := buf.At(0, 0).Style; got != selectionSt {
		t.Fatalf("selected cell style = %+v, want %+v", got, selectionSt)
	}

	if got := buf.At(1, 0).R; got != 'b' {
		t.Fatalf("cell (1,0) = %q, want %q", got, 'b')
	}
}
