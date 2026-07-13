package widgets

import (
	"github.com/losinggeneration/rovel"
	"github.com/losinggeneration/rovel/ui"
)

// Clickable wraps any View to add mouse click handling.
// All View methods delegate to the embedded View. On left-click,
// it requests focus on the wrapped view and calls OnClick.
type Clickable struct {
	rovel.View

	OnClick func(ctx *rovel.Ctx)
}

type actionHandler interface {
	HandleAction(act ui.Action, ctx *rovel.Ctx) bool
}

func (c *Clickable) Handle(e rovel.Event, ctx *rovel.Ctx) bool {
	if me, ok := e.(rovel.MouseEvent); ok {
		if me.Button == rovel.MouseButtonLeft && me.Action == rovel.MousePress {
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

func (c *Clickable) Focusable() bool {
	if f, ok := c.View.(interface{ Focusable() bool }); ok {
		return f.Focusable()
	}

	return false
}

func (c *Clickable) HandleAction(act ui.Action, ctx *rovel.Ctx) bool {
	if ah, ok := c.View.(actionHandler); ok {
		return ah.HandleAction(act, ctx)
	}

	return false
}

// Children delegates to the wrapped view if it implements the Children interface.
// This preserves hit-test traversal for composite views.
func (c *Clickable) Children() []rovel.View {
	if vc, ok := c.View.(interface{ Children() []rovel.View }); ok {
		return vc.Children()
	}

	return nil
}
