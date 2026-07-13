package layout

import (
	"github.com/losinggeneration/rovel"
)

// Padding is a wrapper that adds insets around a single child.
type Padding struct {
	id     rovel.ID
	child  rovel.View
	rect   rovel.Rect
	Left   int
	Top    int
	Right  int
	Bottom int
}

// NewPadding creates a new padding wrapper.
func NewPadding(child rovel.View) *Padding {
	return &Padding{
		id:    rovel.NewID(),
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
func (p *Padding) ID() rovel.ID {
	return p.id
}

// Rect returns the padding's current rect.
func (p *Padding) Rect() rovel.Rect {
	return p.rect
}

// Layout positions the padding within the given rect.
func (p *Padding) Layout(r rovel.Rect) {
	p.rect = r
	inner := InsetRect(r, p.Left, p.Top, p.Right, p.Bottom)
	p.child.Layout(inner)
}

// MinSize returns the minimum size needed for the padding.
func (p *Padding) MinSize() rovel.Size {
	childMin := p.child.MinSize()

	return rovel.Size{
		W: childMin.W + p.Left + p.Right,
		H: childMin.H + p.Top + p.Bottom,
	}
}

// Paint renders the padding and its child.
func (p *Padding) Paint(d rovel.Drawer, ctx *rovel.Ctx) {
	inner := InsetRect(p.rect, p.Left, p.Top, p.Right, p.Bottom)
	d.WithClip(inner, func(d rovel.Drawer) {
		p.child.Paint(d, ctx)
	})
}

// Handle processes events - delegates to child.
func (p *Padding) Handle(e rovel.Event, ctx *rovel.Ctx) bool {
	return p.child.Handle(e, ctx)
}

// Focusable returns false - padding is a wrapper, not a focus target.
func (p *Padding) Focusable() bool {
	return false
}

// Children returns the child view for traversal consistency.
func (p *Padding) Children() []rovel.View {
	return []rovel.View{p.child}
}
