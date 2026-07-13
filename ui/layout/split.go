package layout

import (
	"github.com/losinggeneration/rovel"
	"github.com/losinggeneration/rovel/event"
)

// Orientation specifies the split direction.
type Orientation int

const (
	Horizontal Orientation = iota
	Vertical
)

// Split is a split-pane container with two children and a fixed ratio divider.
// The ratio is static at 50/50. Interactive resizing is deferred to later.
type Split struct {
	id          rovel.ID
	first       rovel.View
	second      rovel.View
	rect        rovel.Rect
	orientation Orientation
	ratio       float64 // fraction of space for first child (0.0 to 1.0)
}

// NewSplit creates a new split container with the given orientation.
func NewSplit(orientation Orientation) *Split {
	return &Split{
		id:          rovel.NewID(),
		orientation: orientation,
		ratio:       0.5, // 50/50 split by default
	}
}

// SetFirst sets the first (top or left) child.
func (s *Split) SetFirst(v rovel.View) {
	s.first = v
}

// SetSecond sets the second (bottom or right) child.
func (s *Split) SetSecond(v rovel.View) {
	s.second = v
}

// SetRatio sets the split ratio for the first child.
// The second child gets the remaining space (1.0 - ratio).
func (s *Split) SetRatio(r float64) {
	if r < 0 {
		r = 0
	}

	if r > 1 {
		r = 1
	}

	s.ratio = r
}

// ID returns the split's unique ID.
func (s *Split) ID() rovel.ID {
	return s.id
}

// Rect returns the split's current rect.
func (s *Split) Rect() rovel.Rect {
	return s.rect
}

// Layout positions the split within the given rect.
func (s *Split) Layout(r rovel.Rect) {
	s.rect = r

	if s.first == nil && s.second == nil {
		return
	}

	if s.orientation == Horizontal {
		// Horizontal split: left and right panes
		totalW := r.W

		// Calculate divider position
		dividerPos := int(float64(totalW) * s.ratio)

		// Ensure at least 1 cell for each side
		if dividerPos < 1 {
			dividerPos = 1
		}

		if dividerPos > totalW-1 {
			dividerPos = totalW - 1
		}

		// Layout first child
		if s.first != nil {
			firstRect := rovel.Rect{
				X: r.X,
				Y: r.Y,
				W: dividerPos,
				H: r.H,
			}
			s.first.Layout(firstRect)
		}

		// Layout second child
		if s.second != nil {
			secondRect := rovel.Rect{
				X: r.X + dividerPos,
				Y: r.Y,
				W: totalW - dividerPos,
				H: r.H,
			}
			s.second.Layout(secondRect)
		}
	} else {
		// Vertical split: top and bottom panes
		totalH := r.H

		// Calculate divider position
		dividerPos := int(float64(totalH) * s.ratio)

		// Ensure at least 1 cell for each side
		if dividerPos < 1 {
			dividerPos = 1
		}

		if dividerPos > totalH-1 {
			dividerPos = totalH - 1
		}

		// Layout first child
		if s.first != nil {
			firstRect := rovel.Rect{
				X: r.X,
				Y: r.Y,
				W: r.W,
				H: dividerPos,
			}
			s.first.Layout(firstRect)
		}

		// Layout second child
		if s.second != nil {
			secondRect := rovel.Rect{
				X: r.X,
				Y: r.Y + dividerPos,
				W: r.W,
				H: totalH - dividerPos,
			}
			s.second.Layout(secondRect)
		}
	}
}

// MinSize returns the minimum size needed for the split.
func (s *Split) MinSize() rovel.Size {
	if s.first == nil && s.second == nil {
		return rovel.Size{W: 1, H: 1}
	}

	if s.orientation == Horizontal {
		// Horizontal: sum widths, max heights
		totalW := 0
		maxH := 0

		if s.first != nil {
			sz := s.first.MinSize()

			totalW += sz.W

			if sz.H > maxH {
				maxH = sz.H
			}
		}

		if s.second != nil {
			sz := s.second.MinSize()

			totalW += sz.W

			if sz.H > maxH {
				maxH = sz.H
			}
		}

		return rovel.Size{W: totalW, H: maxH}
	} else {
		// Vertical: max widths, sum heights
		maxW := 0
		totalH := 0

		if s.first != nil {
			sz := s.first.MinSize()
			if sz.W > maxW {
				maxW = sz.W
			}

			totalH += sz.H
		}

		if s.second != nil {
			sz := s.second.MinSize()
			if sz.W > maxW {
				maxW = sz.W
			}

			totalH += sz.H
		}

		return rovel.Size{W: maxW, H: totalH}
	}
}

// Paint renders the split and its children.
func (s *Split) Paint(d rovel.Drawer, ctx *rovel.Ctx) {
	d.WithClip(s.rect, func(d rovel.Drawer) {
		if s.first != nil {
			d.WithClip(s.first.Rect(), func(d rovel.Drawer) {
				s.first.Paint(d, ctx)
			})
		}

		if s.second != nil {
			d.WithClip(s.second.Rect(), func(d rovel.Drawer) {
				s.second.Paint(d, ctx)
			})
		}
	})
}

// Handle processes events and focus navigation.
func (s *Split) Handle(e rovel.Event, ctx *rovel.Ctx) bool {
	ke, ok := e.(event.KeyEvent)
	if !ok {
		return false
	}

	// Find the focused descendant (not just direct child)
	// and route the event to it first
	focusedDescendant := s.findFocusedDescendant(ctx.FocusedID)
	if focusedDescendant != nil && focusedDescendant.Handle(ke, ctx) {
		return true
	}

	// Handle Tab navigation
	if ke.Key == event.KeyTab {
		// If no focus yet, focus first focusable descendant (MANDATORY)
		if focusedDescendant == nil {
			firstFocusable := s.findFirstFocusable()
			if firstFocusable != nil {
				ctx.RequestFocus(firstFocusable.ID())
				ctx.Invalidate(s.Rect())

				return true
			}
		}

		// Move to next focusable descendant
		// Return false if wrapping would occur, to let parent handle navigation
		nextChild, wrapped := s.findNextFocusable(ctx.FocusedID)
		if nextChild != nil {
			ctx.RequestFocus(nextChild.ID())
			ctx.Invalidate(s.Rect())
			// If we wrapped, return false to let parent handle it
			return !wrapped
		}
	}

	return false
}

// Focusable returns false - containers don't receive focus.
func (s *Split) Focusable() bool {
	return false
}

// Children returns the split's children.
func (s *Split) Children() []rovel.View {
	children := make([]rovel.View, 0, 2)
	if s.first != nil {
		children = append(children, s.first)
	}

	if s.second != nil {
		children = append(children, s.second)
	}

	return children
}

// findFocusedDescendant finds the view with the given ID in the subtree.
func (s *Split) findFocusedDescendant(id rovel.ID) rovel.View {
	return FindByID(s, id)
}

// findFirstFocusable returns the first focusable descendant (searches recursively).
func (s *Split) findFirstFocusable() rovel.View {
	return FindFirstFocusable(s)
}

// findNextFocusable returns the next focusable descendant after the currently focused ID,
// and whether the search wrapped around.
func (s *Split) findNextFocusable(currentFocusID rovel.ID) (rovel.View, bool) {
	// Collect all focusable descendants
	focusableViews := s.collectFocusable()

	if len(focusableViews) == 0 {
		return nil, false
	}

	// Find the index of the currently focused view by ID
	currentIdx := -1

	for i, v := range focusableViews {
		if v.ID() == currentFocusID {
			currentIdx = i

			break
		}
	}

	// If focused view not in list, start from beginning
	if currentIdx < 0 {
		return focusableViews[0], false
	}

	// Return next focusable, check if wrapping
	nextIdx := (currentIdx + 1) % len(focusableViews)
	wrapped := nextIdx < currentIdx // wrapped if next index is less than current

	return focusableViews[nextIdx], wrapped
}

// collectFocusable collects all focusable descendants in order.
func (s *Split) collectFocusable() []rovel.View {
	return CollectFocusable(s)
}
