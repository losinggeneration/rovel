package tui

import "github.com/losinggeneration/tui/geom"

// View is the interface that all UI components must implement.
type View interface {
	// ID returns the unique identifier for this view.
	ID() ID

	// MinSize returns the minimum size needed for this view.
	MinSize() geom.Size

	// Layout positions the view within the given rect.
	Layout(r geom.Rect)

	// Rect returns the current rect of the view.
	Rect() geom.Rect

	// Paint renders the view using the provided painter and context.
	// The view should only paint within the clip rect of the painter.
	Paint(p *Painter, ctx *Ctx)

	// Handle processes an event and returns true if the event was handled.
	Handle(e Event, ctx *Ctx) bool
}

// Ctx provides context methods for views during Paint and Handle.
type Ctx struct {
	// Invalidation callbacks
	invalidate       func(r geom.Rect)
	invalidateAll    func()
	invalidateLayout func(id ID)

	// Focus callbacks
	requestFocus func(id ID)
	focusedID    ID
}

// Invalidate marks a rect as needing repaint.
func (c *Ctx) Invalidate(r geom.Rect) {
	if c.invalidate != nil {
		c.invalidate(r)
	}
}

// InvalidateAll marks the entire screen as needing repaint.
func (c *Ctx) InvalidateAll() {
	if c.invalidateAll != nil {
		c.invalidateAll()
	}
}

// InvalidateLayout marks that a layout pass is needed for a view.
func (c *Ctx) InvalidateLayout(id ID) {
	if c.invalidateLayout != nil {
		c.invalidateLayout(id)
	}
}

// RequestFocus requests focus for the given view ID.
func (c *Ctx) RequestFocus(id ID) {
	if c.requestFocus != nil {
		c.requestFocus(id)
	}
}

// Focused returns true if the given ID is currently focused.
func (c *Ctx) Focused(id ID) bool {
	return c.focusedID == id
}
