package widgets

import (
	"github.com/losinggeneration/tui"
)

// Clickable wraps any View to add mouse click handling.
// All View methods delegate to the embedded View. On left-click,
// it requests focus on the wrapped view and calls OnClick.
type Clickable struct {
	tui.View
	OnClick func(ctx *tui.Ctx)
}

func (c *Clickable) Handle(e tui.Event, ctx *tui.Ctx) bool {
	if me, ok := e.(tui.MouseEvent); ok {
		if me.Button == tui.MouseButtonLeft && me.Action == tui.MousePress {
			if ctx != nil && ctx.RequestFocus != nil {
				ctx.RequestFocus(c.ID())
			}

			if c.OnClick != nil {
				c.OnClick(ctx)
			}

			if ctx != nil {
				ctx.Invalidate(c.Rect())
			}

			return true
		}

		return false
	}

	return c.View.Handle(e, ctx)
}

// MouseOpaque marks Clickable as opaque to hit-testing so it always receives
// clicks even when wrapping a composite view.
func (c *Clickable) MouseOpaque() {}

// Children delegates to the wrapped view if it implements the Children interface.
// This preserves hit-test traversal for composite views.
func (c *Clickable) Children() []tui.View {
	if vc, ok := c.View.(interface{ Children() []tui.View }); ok {
		return vc.Children()
	}

	return nil
}
