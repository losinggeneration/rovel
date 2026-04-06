package tui

import "github.com/losinggeneration/tui/geom"

// UpdateCtx is passed to posted callbacks, providing app-loop-safe mutations.
type UpdateCtx struct {
	Invalidate       func(r geom.Rect)
	InvalidateAll    func()
	InvalidateLayout func(id ID)
	RequestFocus     func(id ID)
	Quit             func()

	// ShowOverlay pushes an overlay onto the stack.
	// The overlay is laid out immediately and the screen is invalidated.
	// If modal, focus is saved and moved to the first focusable view in the overlay subtree.
	ShowOverlay func(opts OverlayOpts) *Overlay

	// DismissOverlay removes the topmost overlay. Restores saved focus if modal.
	// Returns the dismissed overlay, or nil if no overlays exist.
	DismissOverlay func() *Overlay

	// DismissOverlayByID removes a specific overlay by ID.
	// Returns the dismissed overlay, or nil if not found.
	DismissOverlayByID func(id ID) *Overlay
}
