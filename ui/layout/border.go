package layout

import (
	"github.com/losinggeneration/tui"
)

// Border is a wrapper that draws a box border around its child.
// It is purely visual and has no focus logic.
type Border struct {
	id    tui.ID
	child tui.View
	rect  tui.Rect
	title string
}

// NewBorder creates a new border wrapper.
func NewBorder(child tui.View) *Border {
	return &Border{
		id:    tui.NewID(),
		child: child,
	}
}

// SetTitle sets the border title.
func (b *Border) SetTitle(title string) {
	b.title = title
}

// ID returns the border's unique ID.
func (b *Border) ID() tui.ID {
	return b.id
}

// Rect returns the border's current rect.
func (b *Border) Rect() tui.Rect {
	return b.rect
}

// Layout positions the border within the given rect.
func (b *Border) Layout(r tui.Rect) {
	b.rect = r
	inner := InsetRect(r, 1, 1, 1, 1)
	b.child.Layout(inner)
}

// MinSize returns the minimum size needed for the border.
func (b *Border) MinSize() tui.Size {
	childMin := b.child.MinSize()
	return tui.Size{W: childMin.W + 2, H: childMin.H + 2}
}

// Paint renders the border and its child.
func (b *Border) Paint(p *tui.Painter, ctx *tui.Ctx) {
	r := b.rect
	inner := InsetRect(r, 1, 1, 1, 1)

	// Only draw box if rect is large enough
	if r.W >= 2 && r.H >= 2 {
		p.WithClip(r, func(p *tui.Painter) {
			// Draw box border
			p.Box(r, ctx.Theme.Base)

			// Draw title if provided
			if b.title != "" && r.W > 4 {
				titleX := r.X + 2
				titleY := r.Y
				maxTitleW := r.W - 4

				// Truncate title if needed
				titleRunes := []rune(b.title)
				titleW := 0
				titleIdx := 0
				for i, ru := range titleRunes {
					rw := tui.RuneWidth(ru)
					if titleW+rw > maxTitleW {
						break
					}
					titleW += rw
					titleIdx = i + 1
				}

				// Clear top border where title will go
				for i := 0; i < titleW && i < maxTitleW; i++ {
					p.SetCell(titleX+i, titleY, ' ', ctx.Theme.Base)
				}

				// Draw title
				p.Text(titleX, titleY, string(titleRunes[:titleIdx]), ctx.Theme.Base)
			}
		})
	}

	// Paint child in inner rect
	p.WithClip(inner, func(p *tui.Painter) {
		b.child.Paint(p, ctx)
	})
}

// Handle processes events - delegates to child.
func (b *Border) Handle(e tui.Event, ctx *tui.Ctx) bool {
	return b.child.Handle(e, ctx)
}

// Focusable returns false - border is a wrapper, not a focus target.
func (b *Border) Focusable() bool {
	return false
}

// Children returns the child view for traversal consistency.
func (b *Border) Children() []tui.View {
	return []tui.View{b.child}
}
