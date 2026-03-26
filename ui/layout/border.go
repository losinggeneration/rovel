package layout

import (
	"github.com/losinggeneration/tui"
	"github.com/losinggeneration/tui/style"
	"github.com/losinggeneration/tui/text"
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
	b.PaintDrawer(tui.NewDrawer(p), ctx)
	inner := InsetRect(b.rect, 1, 1, 1, 1)
	p.WithClip(inner, func(p *tui.Painter) {
		tui.PaintView(b.child, p, ctx)
	})
}

func (b *Border) PaintDrawer(d tui.Drawer, ctx *tui.Ctx) {
	r := b.rect
	inner := InsetRect(r, 1, 1, 1, 1)

	// Only draw box if rect is large enough
	if r.W >= 2 && r.H >= 2 {
		d.WithClip(r, func(d tui.Drawer) {
			// Use theme border style
			borderStyle := ctx.Theme.Palette.Border
			if borderStyle == (style.Style{}) {
				borderStyle = ctx.Theme.Base
			}

			chrome := ctx.Theme.Chrome.Border.Effective(ctx.Theme)
			d.DrawBorder(r, chrome.BoxStyle(borderStyle))

			// Draw title if provided
			if b.title != "" && r.W > 4 && (chrome.Edges&tui.BoxEdgeTop) != 0 {
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
				d.FillRect(tui.Rect{X: titleX, Y: titleY, W: min(titleW, maxTitleW), H: 1}, borderStyle)

				d.DrawText(tui.Point{X: titleX, Y: titleY}, truncatedTitle, textStyle)
			}
		})
	}

	d.WithClip(inner, func(d tui.Drawer) {
		tui.PaintViewDrawer(b.child, d, ctx)
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}

	return b
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
