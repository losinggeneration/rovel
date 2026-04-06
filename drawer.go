package tui

import (
	"github.com/losinggeneration/tui/geom"
	"github.com/losinggeneration/tui/style"
)

// Drawer is a backend-neutral drawing surface for widget paint code.
//
// The current implementation adapts the existing cell-oriented Painter. It is
// a compatibility layer used to pressure-test the v0.2 draw abstraction before
// changing the public View contract.
//
// DrawText is a single-line primitive: it writes text starting at pos and
// advances horizontally by rune width. It does not interpret '\n' as a line
// break; callers that need multiline layout must split lines themselves.
type Drawer interface {
	FillRect(r geom.Rect, st style.Style)
	DrawText(pos geom.Point, text string, st style.Style)
	DrawBorder(r geom.Rect, bs BoxStyle)

	ClipRect() geom.Rect
	WithClip(r geom.Rect, fn func(Drawer))
	WithOffset(x, y int, fn func(Drawer))
}

// CellDrawer is the preferred escape hatch for widgets that need precise cell
// control on the current cell renderer path without depending on Painter.
type CellDrawer interface {
	Drawer
	SetCell(x, y int, r rune, s style.Style)
}

type painterDrawer struct {
	p *Painter
}

// NewDrawer adapts a Painter to the Drawer interface.
func NewDrawer(p *Painter) Drawer {
	return &painterDrawer{p: p}
}

// CellDrawerOf returns a cell-specific drawer when the current Drawer supports
// exact cell writes. This is the preferred path for grid-oriented widgets that
// need rune-by-rune painting while remaining on the Drawer abstraction.
func CellDrawerOf(d Drawer) (CellDrawer, bool) {
	cd, ok := d.(CellDrawer)

	return cd, ok
}

func (d *painterDrawer) FillRect(r geom.Rect, st style.Style) {
	d.p.Fill(r, ' ', st)
}

func (d *painterDrawer) DrawText(pos geom.Point, text string, st style.Style) {
	d.p.Text(pos.X, pos.Y, text, st)
}

func (d *painterDrawer) DrawBorder(r geom.Rect, bs BoxStyle) {
	d.p.BoxStyled(r, bs)
}

func (d *painterDrawer) SetCell(x, y int, r rune, s style.Style) {
	d.p.SetCell(x, y, r, s)
}

func (d *painterDrawer) ClipRect() geom.Rect {
	return d.p.ClipRect()
}

func (d *painterDrawer) WithClip(r geom.Rect, fn func(Drawer)) {
	d.p.WithClip(r, func(p *Painter) {
		fn(&painterDrawer{p: p})
	})
}

func (d *painterDrawer) WithOffset(x, y int, fn func(Drawer)) {
	d.p.WithOffset(x, y, func(p *Painter) {
		fn(&painterDrawer{p: p})
	})
}
