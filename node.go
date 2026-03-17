package tui

import "github.com/losinggeneration/tui/geom"

// nodeEntry holds tree metadata for a single view node.
type nodeEntry struct {
	id   ID
	view View

	parentID ID   // 0 for root
	childIDs []ID // insertion-ordered

	// Layout geometry
	rect     geom.Rect // root-relative bounds (updated post-layout)
	clipRect geom.Rect // intersection of rect with ancestor clips

	// State
	visible   bool
	enabled   bool
	focusable bool // caches Focusable() check

	// Focus scope — nearest FocusScope ancestor (0 = root scope)
	focusScopeID ID

	// Overlay membership — non-zero = belongs to an overlay subtree
	overlayID ID
}

// focusScopeIface mirrors ui.FocusScope to avoid an import cycle.
type focusScopeIface interface {
	FocusScope() bool
}

// rebuildTree performs a DFS walk of the full view tree (main tree + overlays)
// and repopulates a.nodes with parent pointers, child order, focusable flags,
// focus scope membership, and overlay tags.
//
// Must be called after SetRoot, ShowOverlay, DismissOverlay, and when layoutDirty.
func (a *App) rebuildTree() {
	a.nodes = make(map[ID]*nodeEntry, len(a.nodes)+16)

	if a.root == nil {
		return
	}

	var walk func(v View, parentID ID, scopeID ID, overlayID ID)

	walk = func(v View, parentID ID, scopeID ID, overlayID ID) {
		id := v.ID()

		focusable := false
		if f, ok := v.(viewFocusable); ok {
			focusable = f.Focusable()
		}

		// Check if this view is itself a focus scope.
		newScopeID := scopeID
		if fs, ok := v.(focusScopeIface); ok && fs.FocusScope() {
			newScopeID = id
		}

		entry := &nodeEntry{
			id:           id,
			view:         v,
			parentID:     parentID,
			childIDs:     nil,
			visible:      true,
			enabled:      true,
			focusable:    focusable,
			focusScopeID: scopeID, // the *ancestor* scope (before this node)
			overlayID:    overlayID,
		}
		a.nodes[id] = entry

		// Wire child pointers.
		if parentID != 0 {
			if parent, ok := a.nodes[parentID]; ok {
				parent.childIDs = append(parent.childIDs, id)
			}
		}

		if c, ok := v.(viewChildren); ok {
			for _, child := range c.Children() {
				walk(child, id, newScopeID, overlayID)
			}
		}
	}

	walk(a.root, 0, 0, 0)

	// Walk overlay subtrees, tagging each node with the overlay's ID.
	// Modal overlays act as implicit focus scopes — their root ID becomes
	// the scope boundary so focus traversal wraps within the overlay.
	for _, o := range a.overlays.stack {
		scopeID := ID(0)
		if o.modal {
			scopeID = o.id
		}

		walk(o.root, 0, scopeID, o.id)
	}
}

// updateBounds performs a post-layout DFS that updates nodeEntry.rect from
// v.Rect() and computes clipRect as the intersection with the parent's clipRect.
// Replaces collectLayoutInfo for the geometry-tracking concern.
func (a *App) updateBounds() {
	if a.root == nil {
		return
	}

	screenRect := geom.Rect{X: 0, Y: 0, W: a.size.W, H: a.size.H}

	var walk func(v View, parentClip geom.Rect)

	walk = func(v View, parentClip geom.Rect) {
		id := v.ID()

		entry, ok := a.nodes[id]
		if !ok {
			return
		}

		r := v.Rect()
		entry.rect = r
		entry.clipRect = r.Intersect(parentClip)

		if c, ok := v.(viewChildren); ok {
			for _, child := range c.Children() {
				walk(child, entry.clipRect)
			}
		}
	}

	walk(a.root, screenRect)

	// Update bounds for overlay subtrees.
	for _, o := range a.overlays.stack {
		walk(o.root, screenRect)
	}
}

// ancestorIDs returns the IDs from id up to (but not including) the root, in
// child-to-root order. O(depth).
func (a *App) ancestorIDs(id ID) []ID {
	var result []ID

	current := id
	for {
		entry, ok := a.nodes[current]
		if !ok || entry.parentID == 0 {
			break
		}

		result = append(result, entry.parentID)
		current = entry.parentID
	}

	return result
}
