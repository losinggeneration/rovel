package tui

import (
	"github.com/losinggeneration/tui/geom"
	"github.com/losinggeneration/tui/render"
	"github.com/losinggeneration/tui/style"
)

// Painter provides an immediate-mode drawing API for views.
// It wraps the low-level render.Painter with a more convenient API.
type Painter struct {
	paint *render.Painter
	base  style.Style // Theme base style for Clear()
}

// NewPainter creates a new Painter wrapping a render.Painter.
func NewPainter(rp *render.Painter, baseStyle style.Style) *Painter {
	return &Painter{
		paint: rp,
		base:  baseStyle,
	}
}

// Clear fills a rect with spaces using the theme's base style.
// This is a convenience method for widgets that need to explicitly clear.
// Note: Paint Contract A already clears damaged regions before Paint(),
// so this is only needed for special cases.
func (p *Painter) Clear(r geom.Rect) {
	p.Fill(r, ' ', p.base)
}

// SetCell writes a single cell at the given position.
func (p *Painter) SetCell(x, y int, r rune, s style.Style) {
	p.paint.SetCell(x, y, r, s)
}

// Text writes a string at the given position.
func (p *Painter) Text(x, y int, s string, st style.Style) {
	p.paint.Text(x, y, s, st)
}

// Fill fills a rect with a repeated rune.
func (p *Painter) Fill(r geom.Rect, ch rune, st style.Style) {
	p.paint.Fill(r, ch, st)
}

// HLine draws a horizontal line.
func (p *Painter) HLine(x, y, w int, ch rune, st style.Style) {
	p.paint.HLine(x, y, w, ch, st)
}

// VLine draws a vertical line.
func (p *Painter) VLine(x, y, h int, ch rune, st style.Style) {
	p.paint.VLine(x, y, h, ch, st)
}

// Box draws a box border.
func (p *Painter) Box(r geom.Rect, st style.Style) {
	p.paint.Box(r, st)
}

// ClipRect returns the current clip rect for this painter.
func (p *Painter) ClipRect() geom.Rect {
	return p.paint.ClipRect()
}

// WithClip intersects the current clip with r, sets it for the duration of fn,
// then restores the previous clip. Skips calling fn if intersection is empty.
func (p *Painter) WithClip(r geom.Rect, fn func(p *Painter)) {
	cur := p.paint.ClipRect()
	next := cur.Intersect(r)
	if next.W <= 0 || next.H <= 0 {
		return
	}

	p.paint.SetClipRect(next)
	defer p.paint.SetClipRect(cur)

	fn(p)
}

// WithOffset sets a drawing offset for the duration of fn, then restores it.
// All drawing operations within fn have their coordinates offset by (x, y).
// This is useful for scroll containers where content needs to appear shifted.
func (p *Painter) WithOffset(x, y int, fn func(p *Painter)) {
	oldX, oldY := p.paint.Offset()
	p.paint.SetOffset(oldX+x, oldY+y)
	defer p.paint.SetOffset(oldX, oldY)

	fn(p)
}
