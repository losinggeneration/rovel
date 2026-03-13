package layout

import (
	"github.com/losinggeneration/tui"
	"github.com/losinggeneration/tui/event"
	"github.com/losinggeneration/tui/ui"
)

// Orientation specifies the split direction.
type Orientation int

const (
	Horizontal Orientation = iota
	Vertical
)

// Split is a split-pane container with two children and a fixed ratio divider.
// For M4, the ratio is static at 50/50. Interactive resizing is deferred to later.
type Split struct {
	id          tui.ID
	first       tui.View
	second      tui.View
	rect        tui.Rect
	orientation Orientation
	ratio       float64 // fraction of space for first child (0.0 to 1.0)
}

// NewSplit creates a new split container with the given orientation.
func NewSplit(orientation Orientation) *Split {
	return &Split{
		id:          tui.NewID(),
		orientation: orientation,
		ratio:       0.5, // 50/50 split by default
	}
}

// SetFirst sets the first (top or left) child.
func (s *Split) SetFirst(v tui.View) {
	s.first = v
}

// SetSecond sets the second (bottom or right) child.
func (s *Split) SetSecond(v tui.View) {
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
func (s *Split) ID() tui.ID {
	return s.id
}

// Rect returns the split's current rect.
func (s *Split) Rect() tui.Rect {
	return s.rect
}

// Layout positions the split within the given rect.
func (s *Split) Layout(r tui.Rect) {
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
			firstRect := tui.Rect{
				X: r.X,
				Y: r.Y,
				W: dividerPos,
				H: r.H,
			}
			s.first.Layout(firstRect)
		}

		// Layout second child
		if s.second != nil {
			secondRect := tui.Rect{
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
			firstRect := tui.Rect{
				X: r.X,
				Y: r.Y,
				W: r.W,
				H: dividerPos,
			}
			s.first.Layout(firstRect)
		}

		// Layout second child
		if s.second != nil {
			secondRect := tui.Rect{
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
func (s *Split) MinSize() tui.Size {
	if s.first == nil && s.second == nil {
		return tui.Size{W: 1, H: 1}
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
		return tui.Size{W: totalW, H: maxH}
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
		return tui.Size{W: maxW, H: totalH}
	}
}

// Paint renders the split and its children.
func (s *Split) Paint(p *tui.Painter, ctx *tui.Ctx) {
	// Clip to container rect FIRST, then children
	p.WithClip(s.rect, func(p *tui.Painter) {
		if s.first != nil {
			p.WithClip(s.first.Rect(), func(p *tui.Painter) {
				s.first.Paint(p, ctx)
			})
		}
		if s.second != nil {
			p.WithClip(s.second.Rect(), func(p *tui.Painter) {
				s.second.Paint(p, ctx)
			})
		}
	})
}

// Handle processes events and focus navigation.
func (s *Split) Handle(e tui.Event, ctx *tui.Ctx) bool {
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
func (s *Split) Children() []tui.View {
	children := make([]tui.View, 0, 2)
	if s.first != nil {
		children = append(children, s.first)
	}
	if s.second != nil {
		children = append(children, s.second)
	}
	return children
}

// findFocusedDescendant finds the view with the given ID in the subtree.
func (s *Split) findFocusedDescendant(id tui.ID) tui.View {
	// Check first child
	if s.first != nil && s.first.ID() == id {
		return s.first
	}

	// Check second child
	if s.second != nil && s.second.ID() == id {
		return s.second
	}

	// Search in descendants of first child
	visited := make(map[tui.ID]struct{})
	if s.first != nil {
		visited[s.first.ID()] = struct{}{}
		if composite, ok := s.first.(ui.Composite); ok {
			helper := &DFSHelper{Composite: composite, Visited: visited}
			if result := helper.FindByID(id); result != nil {
				return result
			}
		}
	}

	// Search in descendants of second child
	if s.second != nil {
		visited := make(map[tui.ID]struct{})
		visited[s.second.ID()] = struct{}{}
		if composite, ok := s.second.(ui.Composite); ok {
			helper := &DFSHelper{Composite: composite, Visited: visited}
			if result := helper.FindByID(id); result != nil {
				return result
			}
		}
	}

	return nil
}

// findFirstFocusable returns the first focusable descendant (searches recursively).
func (s *Split) findFirstFocusable() tui.View {
	visited := make(map[tui.ID]struct{})

	// Check first child
	if s.first != nil {
		id := s.first.ID()
		if _, ok := visited[id]; !ok {
			visited[id] = struct{}{}

			if f, ok := s.first.(ui.Focusable); ok && f.Focusable() {
				return s.first
			}

			if composite, ok := s.first.(ui.Composite); ok {
				helper := &DFSHelper{Composite: composite, Visited: visited}
				if result := helper.Search(); result != nil {
					return result
				}
			}
		}
	}

	// Check second child
	if s.second != nil {
		id := s.second.ID()
		if _, ok := visited[id]; !ok {
			visited[id] = struct{}{}

			if f, ok := s.second.(ui.Focusable); ok && f.Focusable() {
				return s.second
			}

			if composite, ok := s.second.(ui.Composite); ok {
				helper := &DFSHelper{Composite: composite, Visited: visited}
				if result := helper.Search(); result != nil {
					return result
				}
			}
		}
	}

	return nil
}

// findNextFocusable returns the next focusable descendant after the currently focused ID,
// and whether the search wrapped around.
func (s *Split) findNextFocusable(currentFocusID tui.ID) (tui.View, bool) {
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
func (s *Split) collectFocusable() []tui.View {
	var result []tui.View
	visited := make(map[tui.ID]struct{})

	// Check first child
	if s.first != nil {
		id := s.first.ID()
		if _, ok := visited[id]; !ok {
			visited[id] = struct{}{}

			if f, ok := s.first.(ui.Focusable); ok && f.Focusable() {
				result = append(result, s.first)
			}

			if composite, ok := s.first.(ui.Composite); ok {
				if fs, ok := s.first.(ui.FocusScope); ok && fs.FocusScope() {
					goto second
				}
				helper := &DFSHelper{Composite: composite, Visited: visited}
				helper.Collect(&result)
			}
		}
	}

second:
	// Check second child
	if s.second != nil {
		id := s.second.ID()
		if _, ok := visited[id]; !ok {
			visited[id] = struct{}{}

			if f, ok := s.second.(ui.Focusable); ok && f.Focusable() {
				result = append(result, s.second)
			}

			if composite, ok := s.second.(ui.Composite); ok {
				if fs, ok := s.second.(ui.FocusScope); ok && fs.FocusScope() {
					return result
				}
				helper := &DFSHelper{Composite: composite, Visited: visited}
				helper.Collect(&result)
			}
		}
	}

	return result
}
