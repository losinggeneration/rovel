package widgets

import (
	"github.com/losinggeneration/tui"
	"github.com/losinggeneration/tui/ui"
	"github.com/losinggeneration/tui/ui/layout"
)

// FocusRing is a decorator that draws a focus border around its child
// when the child or any of its descendants is focused.
type FocusRing struct {
	id    tui.ID
	child tui.View
	rect  tui.Rect
}

func NewFocusRing(child tui.View) *FocusRing {
	return &FocusRing{
		id:    tui.NewID(),
		child: child,
	}
}

func (f *FocusRing) ID() tui.ID {
	return f.id
}

func (f *FocusRing) Rect() tui.Rect {
	return f.rect
}

func (f *FocusRing) Layout(r tui.Rect) {
	f.rect = r
	// Inset by 1 on all sides for the border
	inner := layout.InsetRect(r, 1, 1, 1, 1)
	f.child.Layout(inner)
}

func (f *FocusRing) MinSize() tui.Size {
	childMin := f.child.MinSize()

	return tui.Size{W: childMin.W + 2, H: childMin.H + 2}
}

// Paint renders the focus ring and its child.
func (f *FocusRing) Paint(p *tui.Painter, ctx *tui.Ctx) {
	inner := layout.InsetRect(f.rect, 1, 1, 1, 1)

	// Check if focused (directly or in subtree)
	focused := f.isFocused(ctx.FocusedID)

	// Draw focus border if focused
	if focused {
		p.WithClip(f.rect, func(p *tui.Painter) {
			focusStyle := ctx.Theme.Palette.Focus
			if focusStyle == (tui.Style{}) {
				focusStyle = ctx.Theme.Base
			}

			chrome := ctx.Theme.Chrome.FocusRing.Effective(ctx.Theme)
			p.BoxStyled(f.rect, chrome.BoxStyle(focusStyle))
		})
	}

	// Paint child in inner rect
	p.WithClip(inner, func(p *tui.Painter) {
		f.child.Paint(p, ctx)
	})
}

func (f *FocusRing) Handle(e tui.Event, ctx *tui.Ctx) bool {
	return f.child.Handle(e, ctx)
}

func (f *FocusRing) Focusable() bool {
	return false
}

// Children returns the child view for traversal consistency.
func (f *FocusRing) Children() []tui.View {
	return []tui.View{f.child}
}

// isFocused checks if the given ID is focused within this focus ring's subtree.
func (f *FocusRing) isFocused(focusedID tui.ID) bool {
	// Check if child is directly focused
	if focusedID == f.child.ID() {
		return true
	}

	// Check if focus is in child's subtree
	if composite, ok := f.child.(ui.Composite); ok {
		visited := make(map[tui.ID]struct{})

		return f.hasFocusInSubtree(composite, focusedID, visited)
	}

	return false
}

// hasFocusInSubtree recursively checks if focusedID is in the subtree.
func (f *FocusRing) hasFocusInSubtree(c ui.Composite, focusedID tui.ID, visited map[tui.ID]struct{}) bool {
	for _, child := range c.Children() {
		id := child.ID()
		if _, ok := visited[id]; ok {
			continue
		}

		visited[id] = struct{}{}

		if id == focusedID {
			return true
		}

		if composite, ok := child.(ui.Composite); ok {
			if f.hasFocusInSubtree(composite, focusedID, visited) {
				return true
			}
		}
	}

	return false
}
