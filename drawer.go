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
type Drawer interface {
	FillRect(r geom.Rect, st style.Style)
	DrawText(pos geom.Point, text string, st style.Style)
	DrawBorder(r geom.Rect, bs BoxStyle)

	ClipRect() geom.Rect
	WithClip(r geom.Rect, fn func(Drawer))
	WithOffset(x, y int, fn func(Drawer))
}

// CellDrawer is an optional escape hatch for widgets that need precise cell
// control on the current cell renderer path.
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

// PainterFromDrawer returns the underlying Painter for Drawer implementations
// backed by the current compatibility adapter. It returns nil for other Drawer
// implementations.
func PainterFromDrawer(d Drawer) *Painter {
	pd, ok := d.(*painterDrawer)
	if !ok {
		return nil
	}

	return pd.p
}

// CellDrawerOf returns a cell-specific drawer when the current Drawer supports
// exact cell writes.
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
