package widgets

import (
	"testing"

	"github.com/losinggeneration/tui"
	"github.com/losinggeneration/tui/geom"
	"github.com/losinggeneration/tui/render"
	"github.com/losinggeneration/tui/style"
)

func TestTextInputPaint_DrawerAdapterCursorAndFill(t *testing.T) {
	input := NewTextInput()
	input.Layout(geom.Rect{X: 0, Y: 0, W: 4, H: 1})
	input.SetText(nil, "hi")

	base := style.Style{}
	buf := render.NewBuffer(4, 1)
	buf.Clear(render.Cell{R: ' ', Style: base})
	rp := render.NewPainter(buf, geom.Rect{X: 0, Y: 0, W: 4, H: 1}, base)
	p := tui.NewPainter(rp, base)
	ctx := &tui.Ctx{
		Theme:     tui.DefaultTheme(),
		FocusedID: input.ID(),
	}

	input.Paint(p, ctx)

	if got := buf.At(0, 0).R; got != 'h' {
		t.Fatalf("cell (0,0) = %q, want %q", got, 'h')
	}

	if got := buf.At(1, 0).R; got != 'i' {
		t.Fatalf("cell (1,0) = %q, want %q", got, 'i')
	}

	if got := buf.At(2, 0).R; got != ' ' {
		t.Fatalf("cursor cell = %q, want blank cursor cell", got)
	}

	focusSt := tui.DefaultTheme().Palette.Focus
	if got := buf.At(2, 0).Style; got != focusSt {
		t.Fatalf("cursor style = %+v, want %+v", got, focusSt)
	}

	if got := buf.At(3, 0).R; got != ' ' {
		t.Fatalf("tail fill cell = %q, want %q", got, ' ')
	}
}

func TestTextInputPaint_DrawerAdapterSelectionStyle(t *testing.T) {
	input := NewTextInput()
	input.Layout(geom.Rect{X: 0, Y: 0, W: 4, H: 1})
	input.SetText(nil, "ab")
	input.cursor = 1
	input.anchor = 0

	base := style.Style{}
	buf := render.NewBuffer(4, 1)
	buf.Clear(render.Cell{R: ' ', Style: base})
	rp := render.NewPainter(buf, geom.Rect{X: 0, Y: 0, W: 4, H: 1}, base)
	p := tui.NewPainter(rp, base)
	ctx := &tui.Ctx{
		Theme:     tui.DefaultTheme(),
		FocusedID: input.ID(),
	}

	input.Paint(p, ctx)

	selectionSt := tui.DefaultTheme().Palette.Selection
	if got := buf.At(0, 0).Style; got != selectionSt {
		t.Fatalf("selected cell style = %+v, want %+v", got, selectionSt)
	}

	if got := buf.At(1, 0).R; got != 'b' {
		t.Fatalf("cell (1,0) = %q, want %q", got, 'b')
	}
}
