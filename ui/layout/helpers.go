package layout

import (
	"github.com/losinggeneration/tui"
	"github.com/losinggeneration/tui/ui"
)

// InsetRect returns a rect inset by the given amounts.
func InsetRect(r tui.Rect, l, t, rr, b int) tui.Rect {
	return tui.Rect{
		X: r.X + l,
		Y: r.Y + t,
		W: r.W - l - rr,
		H: r.H - t - b,
	}
}

// DFSHelper is a helper for recursive DFS through composite views.
type DFSHelper struct {
	Composite ui.Composite
	Visited   map[tui.ID]struct{}
}

// Search recursively finds the first focusable view.
func (h *DFSHelper) Search() tui.View {
	for _, child := range h.Composite.Children() {
		id := child.ID()
		if _, ok := h.Visited[id]; ok {
			continue
		}
		h.Visited[id] = struct{}{}

		if f, ok := child.(ui.Focusable); ok && f.Focusable() {
			return child
		}

		if composite, ok := child.(ui.Composite); ok {
			nested := &DFSHelper{Composite: composite, Visited: h.Visited}
			if result := nested.Search(); result != nil {
				return result
			}
		}
	}
	return nil
}

// FindByID recursively finds a view with the given ID.
func (h *DFSHelper) FindByID(id tui.ID) tui.View {
	for _, child := range h.Composite.Children() {
		if child.ID() == id {
			return child
		}
	}

	for _, child := range h.Composite.Children() {
		cid := child.ID()
		if _, ok := h.Visited[cid]; ok {
			continue
		}
		h.Visited[cid] = struct{}{}

		if composite, ok := child.(ui.Composite); ok {
			nested := &DFSHelper{Composite: composite, Visited: h.Visited}
			if result := nested.FindByID(id); result != nil {
				return result
			}
		}
	}
	return nil
}

// Collect recursively collects focusable views.
func (h *DFSHelper) Collect(result *[]tui.View) {
	for _, child := range h.Composite.Children() {
		id := child.ID()
		if _, ok := h.Visited[id]; ok {
			continue
		}
		h.Visited[id] = struct{}{}

		if f, ok := child.(ui.Focusable); ok && f.Focusable() {
			*result = append(*result, child)
		}

		if composite, ok := child.(ui.Composite); ok {
			nested := &DFSHelper{Composite: composite, Visited: h.Visited}
			nested.Collect(result)
		}
	}
}
