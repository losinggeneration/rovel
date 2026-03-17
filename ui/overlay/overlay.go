// Package overlay provides placement strategies and helpers for the
// overlay/layer subsystem defined in the tui package.
//
// The core overlay types ([tui.Overlay], [tui.OverlayManager]) live in
// package tui to avoid import cycles. This package provides concrete
// [tui.Placement] implementations and convenience constructors.
package overlay

import (
	"github.com/losinggeneration/tui"
	"github.com/losinggeneration/tui/geom"
	"github.com/losinggeneration/tui/ui/layout"
)

// Centered places the overlay in the center of the screen at its preferred
// or minimum size.
type Centered struct{}

func (Centered) Resolve(root tui.View, screen geom.Size) geom.Rect {
	sz := preferredOrMin(root)
	sz = clampSize(sz, screen)

	return geom.Rect{
		X: (screen.W - sz.W) / 2,
		Y: (screen.H - sz.H) / 2,
		W: sz.W,
		H: sz.H,
	}
}

// Anchored places the overlay relative to an anchor rect (e.g. a button).
// The overlay appears below the anchor if there is room, otherwise above.
type Anchored struct {
	Anchor geom.Rect
}

func (a Anchored) Resolve(root tui.View, screen geom.Size) geom.Rect {
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

// Fullscreen places the overlay covering the entire screen.
type Fullscreen struct{}

func (Fullscreen) Resolve(_ tui.View, screen geom.Size) geom.Rect {
	return geom.Rect{X: 0, Y: 0, W: screen.W, H: screen.H}
}

// FocusFirst sets focus to the first focusable view in an overlay's subtree.
func FocusFirst(o *tui.Overlay, requestFocus func(tui.ID)) {
	if first := layout.FindFirstFocusable(o.Root()); first != nil {
		requestFocus(first.ID())
	}
}

func preferredOrMin(v tui.View) geom.Size {
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
