package cell

import (
	"github.com/losinggeneration/tui"
	"github.com/losinggeneration/tui/geom"
)

// PaintCallback is the function signature for custom paint callbacks.
// It receives the painter, the canvas rect, and the render context.
type PaintCallback func(p *tui.Painter, rect geom.Rect, ctx *tui.Ctx)

// HandleCallback is the function signature for custom event handling callbacks.
// It receives the event and the render context, returning true if handled.
type HandleCallback func(e tui.Event, ctx *tui.Ctx) bool

// CanvasOpts holds options for creating a cell-specific Canvas.
type CanvasOpts struct {
	ID        tui.ID
	Paint     PaintCallback
	Handle    HandleCallback
	MinSize   geom.Size
	Focusable bool
}

// Canvas is the cell-renderer escape hatch for application-defined paint and
// input behavior. Its callback contract is intentionally Painter-based.
type Canvas struct {
	id        tui.ID
	rect      geom.Rect
	paint     PaintCallback
	handle    HandleCallback
	minSize   geom.Size
	focusable bool
}

func NewCanvas() *Canvas {
	return NewCanvasOpts(CanvasOpts{})
}

func NewCanvasOpts(opts CanvasOpts) *Canvas {
	id := opts.ID
	if id == 0 {
		id = tui.NewID()
	}

	minSize := opts.MinSize
	if minSize.W <= 0 {
		minSize.W = 10
	}

	if minSize.H <= 0 {
		minSize.H = 10
	}

	return &Canvas{
		id:        id,
		paint:     opts.Paint,
		handle:    opts.Handle,
		minSize:   minSize,
		focusable: opts.Focusable,
	}
}

func (c *Canvas) ID() tui.ID {
	return c.id
}

func (c *Canvas) Rect() geom.Rect {
	return c.rect
}

func (c *Canvas) Layout(r geom.Rect) {
	c.rect = r
}

func (c *Canvas) MinSize() geom.Size {
	return c.minSize
}

func (c *Canvas) Focusable() bool {
	return c.focusable
}

// Paint renders the canvas using the paint callback.
// The callback is clipped to the canvas rect.
func (c *Canvas) Paint(p *tui.Painter, ctx *tui.Ctx) {
	if c.paint == nil || c.rect.W <= 0 || c.rect.H <= 0 {
		return
	}

	p.WithClip(c.rect, func(cp *tui.Painter) {
		c.paint(cp, c.rect, ctx)
	})
}

// PaintDrawer preserves compatibility with DrawerPaintable container paths on
// the current cell-family renderer by delegating to the underlying Painter when
// available. Canvas remains explicitly cell-specific.
func (c *Canvas) PaintDrawer(d tui.Drawer, ctx *tui.Ctx) {
	if p := tui.PainterFromDrawer(d); p != nil {
		c.Paint(p, ctx)
	}
}

func (c *Canvas) Handle(e tui.Event, ctx *tui.Ctx) bool {
	if c.handle == nil {
		return false
	}

	return c.handle(e, ctx)
}

func (c *Canvas) SetPaintCallback(cb PaintCallback) {
	c.paint = cb
}

func (c *Canvas) SetHandleCallback(cb HandleCallback) {
	c.handle = cb
}

func (c *Canvas) SetMinSize(ctx *tui.Ctx, sz geom.Size) {
	if sz.W <= 0 {
		sz.W = 1
	}

	if sz.H <= 0 {
		sz.H = 1
	}

	if c.minSize == sz {
		return
	}

	c.minSize = sz

	if ctx != nil && ctx.InvalidateLayout != nil {
		ctx.InvalidateLayout(c.id)
	}
}

func (c *Canvas) SetFocusable(v bool) {
	c.focusable = v
}

// Invalidate marks the entire canvas rect as needing repaint.
func (c *Canvas) Invalidate(ctx *tui.Ctx) {
	if ctx == nil || ctx.Invalidate == nil {
		return
	}

	if c.rect.W <= 0 || c.rect.H <= 0 {
		return
	}

	ctx.Invalidate(c.rect)
}

// InvalidateRect marks a relative rect within the canvas as needing repaint.
func (c *Canvas) InvalidateRect(ctx *tui.Ctx, r geom.Rect) {
	if ctx == nil || ctx.Invalidate == nil {
		return
	}

	if c.rect.W <= 0 || c.rect.H <= 0 {
		return
	}

	if r.W <= 0 || r.H <= 0 {
		return
	}

	abs := geom.Rect{
		X: c.rect.X + r.X,
		Y: c.rect.Y + r.Y,
		W: r.W,
		H: r.H,
	}

	abs = abs.Intersect(c.rect)
	if abs.W <= 0 || abs.H <= 0 {
		return
	}

	ctx.Invalidate(abs)
}

// InvalidateRow marks a single row within the canvas as needing repaint.
func (c *Canvas) InvalidateRow(ctx *tui.Ctx, row int) {
	c.InvalidateRect(ctx, geom.Rect{
		X: 0,
		Y: row,
		W: c.rect.W,
		H: 1,
	})
}

// InvalidateRows marks a range of rows within the canvas as needing repaint.
func (c *Canvas) InvalidateRows(ctx *tui.Ctx, start, count int) {
	c.InvalidateRect(ctx, geom.Rect{
		X: 0,
		Y: start,
		W: c.rect.W,
		H: count,
	})
}
