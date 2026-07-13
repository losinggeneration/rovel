package widgets

import (
	"github.com/losinggeneration/rovel"
	"github.com/losinggeneration/rovel/ui"
	"github.com/losinggeneration/rovel/ui/layout"
)

// FocusRing is a decorator that draws a focus border around its child
// when the child or any of its descendants is focused.
type FocusRing struct {
	id          rovel.ID
	child       rovel.View
	rect        rovel.Rect
	lastFocused bool
}

func NewFocusRing(child rovel.View) *FocusRing {
	return &FocusRing{
		id:    rovel.NewID(),
		child: child,
	}
}

func (f *FocusRing) ID() rovel.ID {
	return f.id
}

func (f *FocusRing) Rect() rovel.Rect {
	return f.rect
}

func (f *FocusRing) Layout(r rovel.Rect) {
	f.rect = r
	// Inset by 1 on all sides for the border
	inner := layout.InsetRect(r, 1, 1, 1, 1)
	f.child.Layout(inner)
}

func (f *FocusRing) MinSize() rovel.Size {
	childMin := f.child.MinSize()

	return rovel.Size{W: childMin.W + 2, H: childMin.H + 2}
}

// Paint renders the focus ring and its child.
func (f *FocusRing) Paint(d rovel.Drawer, ctx *rovel.Ctx) {
	inner := layout.InsetRect(f.rect, 1, 1, 1, 1)

	// Check if focused (directly or in subtree)
	focused := f.isFocused(ctx.FocusedID)

	// If focus state changed since last paint, schedule a
	// follow-up invalidation so the border repaints.
	if focused != f.lastFocused {
		ctx.Invalidate(f.rect)
		f.lastFocused = focused
	}

	// Draw focus border if focused
	if focused {
		d.WithClip(f.rect, func(d rovel.Drawer) {
			focusStyle := ctx.Theme.Palette.Focus
			if focusStyle == (rovel.Style{}) {
				focusStyle = ctx.Theme.Base
			}

			chrome := ctx.Theme.Chrome.FocusRing.Effective(ctx.Theme)
			d.DrawBorder(f.rect, chrome.BoxStyle(focusStyle))
		})
	}

	// Paint child in inner rect
	d.WithClip(inner, func(d rovel.Drawer) {
		f.child.Paint(d, ctx)
	})
}

func (f *FocusRing) Handle(e rovel.Event, ctx *rovel.Ctx) bool {
	return f.child.Handle(e, ctx)
}

func (f *FocusRing) Focusable() bool {
	return false
}

// Children returns the child view for traversal consistency.
func (f *FocusRing) Children() []rovel.View {
	return []rovel.View{f.child}
}

// isFocused checks if the given ID is focused within this focus ring's subtree.
func (f *FocusRing) isFocused(focusedID rovel.ID) bool {
	// Check if child is directly focused
	if focusedID == f.child.ID() {
		return true
	}

	// Check if focus is in child's subtree
	if composite, ok := f.child.(ui.Composite); ok {
		visited := make(map[rovel.ID]struct{})

		return f.hasFocusInSubtree(composite, focusedID, visited)
	}

	return false
}

// hasFocusInSubtree recursively checks if focusedID is in the subtree.
func (f *FocusRing) hasFocusInSubtree(c ui.Composite, focusedID rovel.ID, visited map[rovel.ID]struct{}) bool {
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
