package virtual

import (
	"github.com/losinggeneration/tui"
	"github.com/losinggeneration/tui/geom"
	"github.com/losinggeneration/tui/render"
	"github.com/losinggeneration/tui/style"
)

// mockRenderRow returns a simple RenderRowFunc that renders a numbered prefix.
func mockRenderRow(prefix string) RenderRowFunc {
	return func(
		i int,
		selected bool,
		focused bool,
		p *tui.Painter,
		r geom.Rect,
	) {
		if r.W <= 0 || r.H <= 0 {
			return
		}
		var st tui.Style
		if selected && focused {
			st = tui.Style{Attr: style.AttrReverse}
		} else if selected {
			st = tui.Style{FG: style.ColorWhite, BG: style.ColorBlue}
		}
		// Simple text rendering - truncate to fit
		text := prefix
		maxLen := r.W
		if len(text) > maxLen {
			text = text[:maxLen]
		}
		p.Text(r.X, r.Y, text, st)
	}
}

// benchCtx returns a minimal tui.Ctx for benchmarking.
func benchCtx() *tui.Ctx {
	return &tui.Ctx{
		Invalidate:       func(r geom.Rect) {},
		InvalidateAll:    func() {},
		InvalidateLayout: func(id tui.ID) {},
		RequestFocus:     func(id tui.ID) {},
	}
}

// benchPainter creates a new tui.Painter with a buffer of the given size.
func benchPainter(w, h int) *tui.Painter {
	buf := render.NewBuffer(w, h)
	clip := geom.Rect{X: 0, Y: 0, W: w, H: h}
	baseStyle := style.Style{}
	rp := render.NewPainter(buf, clip, baseStyle)
	return tui.NewPainter(rp, baseStyle)
}

// setupBenchmarkList creates a VirtualList configured for benchmarking.
func setupBenchmarkList(itemCount, rowHeight, width, height int) *VirtualList {
	v := NewVirtualList(VirtualListOpts{
		RowHeight: rowHeight,
		Count:     func() int { return itemCount },
		RenderRow: mockRenderRow("Item"),
	})
	v.Layout(geom.Rect{X: 0, Y: 0, W: width, H: height})
	return v
}
