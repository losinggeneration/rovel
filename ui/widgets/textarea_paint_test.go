package widgets

import (
	"testing"

	"github.com/losinggeneration/tui"
	"github.com/losinggeneration/tui/geom"
	"github.com/losinggeneration/tui/render"
	"github.com/losinggeneration/tui/style"
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
	p := tui.NewPainter(rp, base)
	ctx := &tui.Ctx{
		Theme:     tui.DefaultTheme(),
		FocusedID: ta.ID(),
	}

	ta.Paint(p, ctx)

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

	focusSt := tui.DefaultTheme().Palette.Focus
	if got := buf.At(3, 1).Style; got != focusSt {
		t.Fatalf("cursor style = %+v, want %+v", got, focusSt)
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
	p := tui.NewPainter(rp, base)
	ctx := &tui.Ctx{
		Theme:     tui.DefaultTheme(),
		FocusedID: ta.ID(),
	}

	ta.Paint(p, ctx)

	selectionSt := tui.DefaultTheme().Palette.Selection
	if got := buf.At(0, 0).Style; got != selectionSt {
		t.Fatalf("selected cell style = %+v, want %+v", got, selectionSt)
	}

	if got := buf.At(1, 0).R; got != 'b' {
		t.Fatalf("cell (1,0) = %q, want %q", got, 'b')
	}
}
