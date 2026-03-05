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
	// Theme is the app-wide theme (includes Base + Focus styles).
	Theme Theme

	// Invalidation callbacks (exported to match design doc API).
	Invalidate       func(r geom.Rect)
	InvalidateAll    func()
	InvalidateLayout func(id ID)

	// Focus callbacks (exported to match design doc API).
	RequestFocus func(id ID)
	FocusedID    ID
}
