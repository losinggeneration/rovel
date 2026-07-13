package layout

import (
	"github.com/losinggeneration/rovel"
	"github.com/losinggeneration/rovel/style"
	"github.com/losinggeneration/rovel/text"
)

// Border is a wrapper that draws a box border around its child.
// It is purely visual and has no focus logic.
type Border struct {
	id    rovel.ID
	child rovel.View
	rect  rovel.Rect
	title string
}

// NewBorder creates a new border wrapper.
func NewBorder(child rovel.View) *Border {
	return &Border{
		id:    rovel.NewID(),
		child: child,
	}
}

// SetTitle sets the border title.
func (b *Border) SetTitle(title string) {
	b.title = title
}

// ID returns the border's unique ID.
func (b *Border) ID() rovel.ID {
	return b.id
}

// Rect returns the border's current rect.
func (b *Border) Rect() rovel.Rect {
	return b.rect
}

// Layout positions the border within the given rect.
func (b *Border) Layout(r rovel.Rect) {
	b.rect = r
	inner := InsetRect(r, 1, 1, 1, 1)
	b.child.Layout(inner)
}

// MinSize returns the minimum size needed for the border.
func (b *Border) MinSize() rovel.Size {
	childMin := b.child.MinSize()

	return rovel.Size{W: childMin.W + 2, H: childMin.H + 2}
}

// Paint renders the border and its child.
func (b *Border) Paint(d rovel.Drawer, ctx *rovel.Ctx) {
	r := b.rect
	inner := InsetRect(r, 1, 1, 1, 1)

	// Only draw box if rect is large enough
	if r.W >= 2 && r.H >= 2 {
		d.WithClip(r, func(d rovel.Drawer) {
			// Use theme border style
			borderStyle := ctx.Theme.Palette.Border
			if borderStyle == (style.Style{}) {
				borderStyle = ctx.Theme.Base
			}

			chrome := ctx.Theme.Chrome.Border.Effective(ctx.Theme)
			d.DrawBorder(r, chrome.BoxStyle(borderStyle))

			// Draw title if provided
			if b.title != "" && r.W > 4 && (chrome.Edges&rovel.BoxEdgeTop) != 0 {
				// Use theme text style
				textStyle := ctx.Theme.Palette.Text
				if textStyle == (style.Style{}) {
					textStyle = ctx.Theme.Base
				}

				titleX := r.X + 2
				titleY := r.Y
				maxTitleW := r.W - 4

				end, titleW, _ := text.FitPrefix(b.title, maxTitleW)
				truncatedTitle := b.title[:end]

				// Clear title background
				d.FillRect(rovel.Rect{X: titleX, Y: titleY, W: min(titleW, maxTitleW), H: 1}, borderStyle)

				d.DrawText(rovel.Point{X: titleX, Y: titleY}, truncatedTitle, textStyle)
			}
		})
	}

	d.WithClip(inner, func(d rovel.Drawer) {
		b.child.Paint(d, ctx)
	})
}

// Handle processes events - delegates to child.
func (b *Border) Handle(e rovel.Event, ctx *rovel.Ctx) bool {
	return b.child.Handle(e, ctx)
}

// Focusable returns false - border is a wrapper, not a focus target.
func (b *Border) Focusable() bool {
	return false
}

// Children returns the child view for traversal consistency.
func (b *Border) Children() []rovel.View {
	return []rovel.View{b.child}
}
