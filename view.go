package rovel

import (
	"github.com/losinggeneration/rovel/backend"
	"github.com/losinggeneration/rovel/geom"
	"github.com/losinggeneration/rovel/style"
)

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

	// Paint renders the view using the provided drawer and context.
	// The view should only paint within the clip rect of the drawer.
	Paint(d Drawer, ctx *Ctx)

	// Handle processes an event and returns true if the event was handled.
	Handle(e Event, ctx *Ctx) bool
}

type CompositeView interface {
	View
	Children() []View
}

// PreferredSizer is an optional interface views may implement to expose a
// preferred size hint. The app uses it, when set, to derive a default render
// region in cbreak mode (see AppOpts.RenderSize).
type PreferredSizer interface {
	PreferredSize() geom.Size
}

// Ctx provides context methods for views during Paint and Handle.
type Ctx struct {
	// Theme is the app-wide theme.
	Theme Theme

	// Cap is the terminal color capability.
	Cap style.Capability

	// Invalidation callbacks (exported to match design doc API).
	Invalidate       func(r geom.Rect)
	InvalidateAll    func()
	InvalidateLayout func()

	// Focus callbacks (exported to match design doc API).
	RequestFocus func(id ID)
	FocusedID    ID

	// Quit requests the application to stop.
	Quit func()

	// Suspend suspends the application (restore terminal, stop the process,
	// repaint on resume). Nil if the backend does not support suspension.
	// Useful for binding a suspend key in raw mode, where Ctrl+Z arrives as a
	// key event rather than a signal.
	Suspend func() error

	// Mod holds the modifier keys active for the current key event.
	Mod ModMask

	// InputCaps reports which input features the backend supports.
	InputCaps backend.InputCapabilities

	// ClipboardWrite writes text to the system clipboard.
	// Nil if clipboard is not available.
	ClipboardWrite func(string)

	// ClipboardRead sends a clipboard read request. The response arrives
	// asynchronously as a ClipboardResponseEvent. Nil if not available.
	ClipboardRead func()

	// ShowOverlay pushes an overlay. Nil if overlay system not available.
	ShowOverlay func(opts OverlayOpts) *Overlay

	// DismissOverlay removes the topmost overlay. Nil if not available.
	DismissOverlay func() *Overlay

	// DismissOverlayByID removes a specific overlay by ID. Nil if not available.
	DismissOverlayByID func(id ID) *Overlay
}
