package layout

import (
	"github.com/losinggeneration/tui"
)

// Padding is a wrapper that adds insets around a single child.
type Padding struct {
	id     tui.ID
	child  tui.View
	rect   tui.Rect
	Left   int
	Top    int
	Right  int
	Bottom int
}

// NewPadding creates a new padding wrapper.
func NewPadding(child tui.View) *Padding {
	return &Padding{
		id:    tui.NewID(),
		child: child,
	}
}

// SetInsets sets all padding insets at once.
func (p *Padding) SetInsets(left, top, right, bottom int) {
	p.Left = left
	p.Top = top
	p.Right = right
	p.Bottom = bottom
}

// ID returns the padding's unique ID.
func (p *Padding) ID() tui.ID {
	return p.id
}

// Rect returns the padding's current rect.
func (p *Padding) Rect() tui.Rect {
	return p.rect
}

// Layout positions the padding within the given rect.
func (p *Padding) Layout(r tui.Rect) {
	p.rect = r
	inner := InsetRect(r, p.Left, p.Top, p.Right, p.Bottom)
	p.child.Layout(inner)
}

// MinSize returns the minimum size needed for the padding.
func (p *Padding) MinSize() tui.Size {
	childMin := p.child.MinSize()

	return tui.Size{
		W: childMin.W + p.Left + p.Right,
		H: childMin.H + p.Top + p.Bottom,
	}
}

// Paint renders the padding and its child.
func (p *Padding) Paint(painter *tui.Painter, ctx *tui.Ctx) {
	inner := InsetRect(p.rect, p.Left, p.Top, p.Right, p.Bottom)
	painter.WithClip(inner, func(painter *tui.Painter) {
		tui.PaintView(p.child, painter, ctx)
	})
}

// Handle processes events - delegates to child.
func (p *Padding) Handle(e tui.Event, ctx *tui.Ctx) bool {
	return p.child.Handle(e, ctx)
}

// Focusable returns false - padding is a wrapper, not a focus target.
func (p *Padding) Focusable() bool {
	return false
}

// Children returns the child view for traversal consistency.
func (p *Padding) Children() []tui.View {
	return []tui.View{p.child}
}
