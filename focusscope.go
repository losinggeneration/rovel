package rovel

// scopeState stores per-scope state (keyed by scope owner ID; 0 = root scope).
type scopeState struct {
	lastFocused ID
}

// collectFocusableInScope returns an ordered list of focusable node IDs
// within the given scope. Stops recursing when a child's focusScopeID differs
// (nested scope boundary).
func (a *App) collectFocusableInScope(scopeID ID) []ID {
	var (
		result  []ID
		walk    func(v View)
		visited = make(map[ID]struct{}, len(a.nodes))
	)

	walk = func(v View) {
		id := v.ID()

		entry, ok := a.nodes[id]
		if !ok {
			return
		}
		// If this node is in a different scope, don't descend.
		if entry.focusScopeID != scopeID {
			return
		}

		if id != 0 {
			if _, seen := visited[id]; seen {
				return
			}
			visited[id] = struct{}{}
		}

		if entry.focusable {
			result = append(result, id)
		}

		if c, ok := v.(viewChildren); ok {
			for _, child := range c.Children() {
				walk(child)
			}
		}
	}

	// Determine which tree root to start from.
	if scopeID == 0 {
		// Root scope: walk main tree.
		if a.root != nil {
			walk(a.root)
		}
	} else {
		// Non-root scope: find the scope owner and walk its children.
		if entry, ok := a.nodes[scopeID]; ok {
			if c, ok := entry.view.(viewChildren); ok {
				for _, child := range c.Children() {
					walk(child)
				}
			}
		}
	}

	return result
}

// focusNextInScope advances focus to the next focusable view within the scope
// of the currently focused view. Wraps around at boundaries.
func (a *App) focusNextInScope() {
	scope := a.focusScopeOf(a.focusedID)

	targets := a.collectFocusableInScope(scope)
	if len(targets) == 0 {
		return
	}

	// When focus isn't among the targets (e.g. nothing focused yet, or the
	// focused view is unfocusable), start from the first target.
	next := 0
	if idx := indexOf(targets, a.focusedID); idx >= 0 {
		next = (idx + 1) % len(targets)
	}

	a.setRequestFocus(targets[next])
}

// focusPrevInScope moves focus to the previous focusable view within scope.
func (a *App) focusPrevInScope() {
	scope := a.focusScopeOf(a.focusedID)

	targets := a.collectFocusableInScope(scope)
	if len(targets) == 0 {
		return
	}

	// When focus isn't among the targets, wrap to the last target.
	prev := len(targets) - 1
	if idx := indexOf(targets, a.focusedID); idx >= 0 {
		prev = (idx - 1 + len(targets)) % len(targets)
	}

	a.setRequestFocus(targets[prev])
}

// focusScopeOf returns the scope ID for a given node.
func (a *App) focusScopeOf(id ID) ID {
	if entry, ok := a.nodes[id]; ok {
		return entry.focusScopeID
	}

	return 0
}

// ensureValidFocusScoped is the scope-aware version of ensureValidFocus.
// Repair chain: (1) scopeMemory lastFocused if still valid, (2) first focusable
// in same scope, (3) walk to parent scope, (4) clear focus.
func (a *App) ensureValidFocusScoped() {
	if a.focusedID == 0 {
		return
	}

	// Currently focused view is still mounted and focusable => keep it.
	if entry, ok := a.nodes[a.focusedID]; ok && entry.focusable {
		return
	}

	scope := a.focusScopeOf(a.focusedID)

	// (1) Try scope memory.
	if ss, ok := a.scopeMemory[scope]; ok && ss.lastFocused != a.focusedID {
		if entry, eOk := a.nodes[ss.lastFocused]; eOk && entry.focusable {
			a.setRequestFocus(ss.lastFocused)

			return
		}
	}

	// (2) First focusable in same scope.
	targets := a.collectFocusableInScope(scope)
	if len(targets) > 0 {
		a.setRequestFocus(targets[0])

		return
	}

	// (3) Walk to parent scope.
	if scope != 0 {
		if scopeEntry, ok := a.nodes[scope]; ok {
			parentScope := scopeEntry.focusScopeID

			parentTargets := a.collectFocusableInScope(parentScope)
			if len(parentTargets) > 0 {
				a.setRequestFocus(parentTargets[0])

				return
			}
		}
	}

	// (4) Clear focus.
	a.setRequestFocus(0)
}

// indexOf returns the position of target in ids, or -1 if not present.
func indexOf(ids []ID, target ID) int {
	for i, id := range ids {
		if id == target {
			return i
		}
	}

	return -1
}
