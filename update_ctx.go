package rovel

import "github.com/losinggeneration/rovel/geom"

// UpdateCtx is passed to posted callbacks, providing app-loop-safe mutations.
type UpdateCtx struct {
	Invalidate       func(r geom.Rect)
	InvalidateAll    func()
	InvalidateLayout func()
	RequestFocus     func(id ID)
	Quit             func()

	// ShowOverlay pushes an overlay onto the stack.
	// The overlay is laid out immediately and the screen is invalidated.
	// If modal, focus is saved and moved to the first focusable view in the overlay subtree.
	ShowOverlay func(opts OverlayOpts) *Overlay

	// RaiseOverlay moves an existing overlay to the top of the stack
	// (below any modal overlay), preserving its ID and firing no dismiss
	// side effects.
	RaiseOverlay func(id ID) *Overlay

	// DismissOverlay removes the topmost overlay. Restores saved focus if modal.
	// Returns the dismissed overlay, or nil if no overlays exist.
	DismissOverlay func() *Overlay

	// DismissOverlayByID removes a specific overlay by ID.
	// Returns the dismissed overlay, or nil if not found.
	DismissOverlayByID func(id ID) *Overlay
}
