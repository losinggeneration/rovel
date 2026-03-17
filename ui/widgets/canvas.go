package widgets

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

// CanvasOpts holds options for creating a Canvas.
type CanvasOpts struct {
	ID        tui.ID
	Paint     PaintCallback
	Handle    HandleCallback
	MinSize   geom.Size
	Focusable bool
}

// Canvas is a framework escape hatch for application-defined paint and input behavior.
// It remains a normal tui.View, so it composes naturally inside layout containers
// and obeys clipping, z-order, focus, and invalidation rules.
type Canvas struct {
	id        tui.ID
	rect      geom.Rect
	paint     PaintCallback
	handle    HandleCallback
	minSize   geom.Size
	focusable bool
}

// NewCanvas creates a new canvas with default options.
func NewCanvas() *Canvas {
	return NewCanvasOpts(CanvasOpts{})
}

// NewCanvasOpts creates a new canvas with the given options.
func NewCanvasOpts(opts CanvasOpts) *Canvas {
	id := opts.ID
	if isZeroID(id) {
		id = tui.NewID()
	}

	min := opts.MinSize
	if min.W <= 0 {
		min.W = 10
	}

	if min.H <= 0 {
		min.H = 10
	}

	return &Canvas{
		id:        id,
		paint:     opts.Paint,
		handle:    opts.Handle,
		minSize:   min,
		focusable: opts.Focusable,
	}
}

// ID returns the canvas's unique ID.
func (c *Canvas) ID() tui.ID {
	return c.id
}

// Rect returns the canvas's current rect.
func (c *Canvas) Rect() geom.Rect {
	return c.rect
}

// Layout positions the canvas within the given rect.
func (c *Canvas) Layout(r geom.Rect) {
	c.rect = r
}

// MinSize returns the minimum size needed for the canvas.
func (c *Canvas) MinSize() geom.Size {
	return c.minSize
}

// Focusable returns true if the canvas can receive focus.
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

// Handle processes events using the handle callback.
func (c *Canvas) Handle(e tui.Event, ctx *tui.Ctx) bool {
	if c.handle == nil {
		return false
	}

	return c.handle(e, ctx)
}

// SetPaintCallback sets the paint callback.
func (c *Canvas) SetPaintCallback(cb PaintCallback) {
	c.paint = cb
}

// SetHandleCallback sets the handle callback.
func (c *Canvas) SetHandleCallback(cb HandleCallback) {
	c.handle = cb
}

// SetMinSize sets the minimum size and triggers layout invalidation.
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

// SetFocusable sets whether the canvas can receive focus.
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
// The relative rect is offset by the canvas origin, intersected with the canvas rect,
// and the result is invalidated.
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

	abs = intersectRect(abs, c.rect)
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

// intersectRect returns the intersection of two rectangles.
func intersectRect(a, b geom.Rect) geom.Rect {
	x0 := maxInt(a.X, b.X)
	y0 := maxInt(a.Y, b.Y)
	x1 := minInt(a.X+a.W, b.X+b.W)
	y1 := minInt(a.Y+a.H, b.Y+b.H)

	if x1 <= x0 || y1 <= y0 {
		return geom.Rect{}
	}

	return geom.Rect{
		X: x0,
		Y: y0,
		W: x1 - x0,
		H: y1 - y0,
	}
}

// isZeroID returns true if the ID is the zero value.
func isZeroID(id tui.ID) bool {
	var zero tui.ID

	return id == zero
}

func minInt(a, b int) int {
	if a < b {
		return a
	}

	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}

	return b
}
