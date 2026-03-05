package layout

import (
	"github.com/losinggeneration/tui"
	"github.com/losinggeneration/tui/event"
	"github.com/losinggeneration/tui/ui"
)

// HStack is a horizontal stacking container that arranges children left-to-right.
type HStack struct {
	id       tui.ID
	children []tui.View
	rect     tui.Rect
}

// NewHStack creates a new horizontal stack container.
func NewHStack() *HStack {
	return &HStack{
		id:       tui.NewID(),
		children: make([]tui.View, 0),
	}
}

// Add adds a child to the stack.
func (s *HStack) Add(v tui.View) {
	s.children = append(s.children, v)
}

// ID returns the stack's unique ID.
func (s *HStack) ID() tui.ID {
	return s.id
}

// Rect returns the stack's current rect.
func (s *HStack) Rect() tui.Rect {
	return s.rect
}

// Layout positions the stack within the given rect.
func (s *HStack) Layout(r tui.Rect) {
	s.rect = r

	if len(s.children) == 0 {
		return
	}

	// Calculate total min size on main axis
	totalMinW := 0
	maxH := 0
	for _, child := range s.children {
		minSz := child.MinSize()
		totalMinW += minSz.W
		if minSz.H > maxH {
			maxH = minSz.H
		}
	}

	// Distribute space: each child gets MinSize on main axis,
	// remaining space goes to last child
	x := r.X
	availableW := r.W

	for i, child := range s.children {
		minSz := child.MinSize()

		childW := minSz.W
		if i == len(s.children)-1 {
			// Last child gets remaining space
			rem := availableW - totalMinW
			if rem > 0 {
				childW += rem
			}
		}
		// Clamp if we ran out of space
		if childW > availableW {
			childW = availableW
		}
		if childW < 0 {
			childW = 0
		}

		childH := r.H
		if childH < 0 {
			childH = 0
		}

		childRect := tui.Rect{
			X: x,
			Y: r.Y,
			W: childW,
			H: childH,
		}
		child.Layout(childRect)

		x += childW
		availableW -= childW
	}
}

// MinSize returns the minimum size needed for the stack.
func (s *HStack) MinSize() tui.Size {
	totalW := 0
	maxH := 0
	for _, child := range s.children {
		sz := child.MinSize()
		totalW += sz.W
		if sz.H > maxH {
			maxH = sz.H
		}
	}
	return tui.Size{W: totalW, H: maxH}
}

// Paint renders the stack and its children.
func (s *HStack) Paint(p *tui.Painter, ctx *tui.Ctx) {
	// Clip to container rect FIRST, then children
	p.WithClip(s.rect, func(p *tui.Painter) {
		for _, child := range s.children {
			p.WithClip(child.Rect(), func(p *tui.Painter) {
				child.Paint(p, ctx)
			})
		}
	})
}

// Handle processes events and focus navigation.
func (s *HStack) Handle(e tui.Event, ctx *tui.Ctx) bool {
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
		nextFocusable, wrapped := s.findNextFocusable(ctx.FocusedID)
		if nextFocusable != nil {
			ctx.RequestFocus(nextFocusable.ID())
			ctx.Invalidate(s.Rect())
			// If we wrapped, return false to let parent handle it
			return !wrapped
		}
	}

	return false
}

// Focusable returns false - containers don't receive focus.
func (s *HStack) Focusable() bool {
	return false
}

// Children returns the stack's children.
func (s *HStack) Children() []tui.View {
	return s.children
}

// findFocusedDescendant finds the view with the given ID in the subtree.
func (s *HStack) findFocusedDescendant(id tui.ID) tui.View {
	// Check direct children first
	for _, child := range s.children {
		if child.ID() == id {
			return child
		}
	}

	// Search in descendants
	visited := make(map[tui.ID]struct{})
	for _, child := range s.children {
		if child.ID() == id {
			return child
		}
		if _, ok := visited[child.ID()]; ok {
			continue
		}
		visited[child.ID()] = struct{}{}

		if composite, ok := child.(ui.Composite); ok {
			helper := &DFSHelper{Composite: composite, Visited: visited}
			if result := helper.FindByID(id); result != nil {
				return result
			}
		}
	}
	return nil
}

// findFirstFocusable returns the first focusable descendant (searches recursively).
func (s *HStack) findFirstFocusable() tui.View {
	visited := make(map[tui.ID]struct{})
	for _, child := range s.children {
		id := child.ID()
		if _, ok := visited[id]; ok {
			continue
		}
		visited[id] = struct{}{}

		if f, ok := child.(ui.Focusable); ok && f.Focusable() {
			return child
		}

		if composite, ok := child.(ui.Composite); ok {
			helper := &DFSHelper{Composite: composite, Visited: visited}
			if result := helper.Search(); result != nil {
				return result
			}
		}
	}
	return nil
}

// findNextFocusable returns the next focusable descendant after the given ID,
// and whether the search wrapped around.
func (s *HStack) findNextFocusable(currentFocusID tui.ID) (tui.View, bool) {
	// Collect all focusable views in order
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
func (s *HStack) collectFocusable() []tui.View {
	var result []tui.View
	visited := make(map[tui.ID]struct{})
	for _, child := range s.children {
		id := child.ID()
		if _, ok := visited[id]; ok {
			continue
		}
		visited[id] = struct{}{}

		if f, ok := child.(ui.Focusable); ok && f.Focusable() {
			result = append(result, child)
		}

		if composite, ok := child.(ui.Composite); ok {
			helper := &DFSHelper{Composite: composite, Visited: visited}
			helper.Collect(&result)
		}
	}
	return result
}
