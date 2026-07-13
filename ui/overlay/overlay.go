// Package overlay provides placement strategies and helpers for the
// overlay/layer subsystem defined in the tui package.
//
// The core overlay types ([rovel.Overlay], [rovel.OverlayManager]) live in
// package rovel to avoid import cycles. This package provides concrete
// [rovel.Placement] implementations and convenience constructors.
package overlay

import (
	"github.com/losinggeneration/rovel"
	"github.com/losinggeneration/rovel/geom"
	"github.com/losinggeneration/rovel/ui/layout"
)

// Centered places the overlay in the center of the screen at its preferred
// or minimum size.
type Centered struct{}

func (Centered) Resolve(root rovel.View, screen geom.Size) geom.Rect {
	sz := preferredOrMin(root)
	sz = clampSize(sz, screen)

	return geom.Rect{
		X: (screen.W - sz.W) / 2,
		Y: (screen.H - sz.H) / 2,
		W: sz.W,
		H: sz.H,
	}
}

// TopCentered places the overlay at the top of the screen, centered
// horizontally at its preferred or minimum size unless overridden.
type TopCentered struct {
	W int
	H int
}

func (p TopCentered) Resolve(root rovel.View, screen geom.Size) geom.Rect {
	sz := preferredOrMin(root)
	if p.W > 0 {
		sz.W = p.W
	}
	if p.H > 0 {
		sz.H = p.H
	}
	sz = clampSize(sz, screen)

	return geom.Rect{
		X: (screen.W - sz.W) / 2,
		Y: 0,
		W: sz.W,
		H: sz.H,
	}
}

// Anchored places the overlay relative to an anchor rect (e.g. a button).
// The overlay appears below the anchor if there is room, otherwise above.
type Anchored struct {
	Anchor geom.Rect
}

func (a Anchored) Resolve(root rovel.View, screen geom.Size) geom.Rect {
	sz := preferredOrMin(root)
	sz = clampSize(sz, screen)

	x := a.Anchor.X
	if x+sz.W > screen.W {
		x = screen.W - sz.W
	}

	x = max(x, 0)

	// Prefer below anchor
	y := a.Anchor.Y + a.Anchor.H
	if y+sz.H > screen.H {
		// Try above
		y = max(a.Anchor.Y-sz.H, 0)
	}

	return geom.Rect{X: x, Y: y, W: sz.W, H: sz.H}
}

// PointAnchored places the overlay relative to a point. It prefers to appear
// below and to the right of the anchor, flipping above/left when requested
// and needed to stay on-screen.
type PointAnchored struct {
	Anchor geom.Point
	Offset geom.Point
	W      int
	H      int
	Flip   bool
}

func (p PointAnchored) Resolve(root rovel.View, screen geom.Size) geom.Rect {
	sz := preferredOrMin(root)
	if p.W > 0 {
		sz.W = p.W
	}
	if p.H > 0 {
		sz.H = p.H
	}
	sz = clampSize(sz, screen)

	x := p.Anchor.X + p.Offset.X
	y := p.Anchor.Y + p.Offset.Y
	if p.Flip {
		if x+sz.W > screen.W {
			x = p.Anchor.X - sz.W - p.Offset.X
		}
		if y+sz.H > screen.H {
			y = p.Anchor.Y - sz.H - p.Offset.Y
		}
	}

	if x+sz.W > screen.W {
		x = screen.W - sz.W
	}
	if y+sz.H > screen.H {
		y = screen.H - sz.H
	}

	return geom.Rect{
		X: max(x, 0),
		Y: max(y, 0),
		W: sz.W,
		H: sz.H,
	}
}

// Fullscreen places the overlay covering the entire screen.
type Fullscreen struct{}

func (Fullscreen) Resolve(_ rovel.View, screen geom.Size) geom.Rect {
	return geom.Rect{X: 0, Y: 0, W: screen.W, H: screen.H}
}

// FocusFirst sets focus to the first focusable view in an overlay's subtree.
func FocusFirst(o *rovel.Overlay, requestFocus func(rovel.ID)) {
	if first := layout.FindFirstFocusable(o.Root()); first != nil {
		requestFocus(first.ID())
	}
}

func preferredOrMin(v rovel.View) geom.Size {
	type preferredSizer interface {
		PreferredSize() geom.Size
	}

	if ps, ok := v.(preferredSizer); ok {
		return ps.PreferredSize()
	}

	return v.MinSize()
}

func clampSize(sz, screen geom.Size) geom.Size {
	if sz.W > screen.W {
		sz.W = screen.W
	}

	if sz.H > screen.H {
		sz.H = screen.H
	}

	return sz
}
