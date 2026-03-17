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

// FindFirstFocusable does a DFS preorder traversal and returns the first
// focusable view found under root. If a view implements ui.FocusScope and
// FocusScope() is true, traversal does not descend into that view's children.
func FindFirstFocusable(root tui.View) tui.View {
	composite, ok := root.(ui.Composite)
	if !ok {
		return nil
	}

	visited := make(map[tui.ID]struct{})
	visited[root.ID()] = struct{}{}

	return findFirstFocusable(composite, visited)
}

func findFirstFocusable(c ui.Composite, visited map[tui.ID]struct{}) tui.View {
	for _, child := range c.Children() {
		id := child.ID()
		if _, ok := visited[id]; ok {
			continue
		}

		visited[id] = struct{}{}

		if f, ok := child.(ui.Focusable); ok && f.Focusable() {
			return child
		}

		if fs, ok := child.(ui.FocusScope); ok && fs.FocusScope() {
			continue
		}

		if nested, ok := child.(ui.Composite); ok {
			if v := findFirstFocusable(nested, visited); v != nil {
				return v
			}
		}
	}

	return nil
}

// FindByID does a DFS traversal and returns the first view found with the
// given id under root. This traversal does not treat ui.FocusScope as a
// boundary (it searches the full tree).
func FindByID(root tui.View, id tui.ID) tui.View {
	composite, ok := root.(ui.Composite)
	if !ok {
		return nil
	}

	visited := make(map[tui.ID]struct{})
	visited[root.ID()] = struct{}{}

	return findByID(composite, id, visited)
}

func findByID(c ui.Composite, id tui.ID, visited map[tui.ID]struct{}) tui.View {
	for _, child := range c.Children() {
		if child.ID() == id {
			return child
		}
	}

	for _, child := range c.Children() {
		cid := child.ID()
		if _, ok := visited[cid]; ok {
			continue
		}

		visited[cid] = struct{}{}

		if nested, ok := child.(ui.Composite); ok {
			if v := findByID(nested, id, visited); v != nil {
				return v
			}
		}
	}

	return nil
}

// CollectFocusable does a DFS preorder traversal and returns all focusable
// views under root. If a view implements ui.FocusScope and FocusScope() is
// true, traversal does not descend into that view's children.
func CollectFocusable(root tui.View) []tui.View {
	composite, ok := root.(ui.Composite)
	if !ok {
		return nil
	}

	visited := make(map[tui.ID]struct{})
	visited[root.ID()] = struct{}{}

	var out []tui.View
	collectFocusable(composite, visited, &out)

	return out
}

func collectFocusable(c ui.Composite, visited map[tui.ID]struct{}, out *[]tui.View) {
	for _, child := range c.Children() {
		id := child.ID()
		if _, ok := visited[id]; ok {
			continue
		}

		visited[id] = struct{}{}

		if f, ok := child.(ui.Focusable); ok && f.Focusable() {
			*out = append(*out, child)
		}

		if fs, ok := child.(ui.FocusScope); ok && fs.FocusScope() {
			continue
		}

		if nested, ok := child.(ui.Composite); ok {
			collectFocusable(nested, visited, out)
		}
	}
}
