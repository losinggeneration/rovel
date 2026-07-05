package tui

import (
	"testing"

	"github.com/losinggeneration/tui/geom"
)

// ---------------------------------------------------------------------------
// FocusScope integration tests
// ---------------------------------------------------------------------------

func TestFocusNextInScope_WrapsWithinScope(t *testing.T) {
	app, _ := New(AppOpts{})
	app.size = geom.Size{W: 80, H: 24}

	// Two scopes: scope1 has a1, a2; root scope has b1.
	a1 := newMockNode(true)
	a2 := newMockNode(true)
	scope1 := newMockFocusScope(a1, a2)
	b1 := newMockNode(true)
	root := newMockContainer(scope1, b1)

	app.root = root
	app.rebuildTree()

	// Focus a2 (last in scope1). Next should wrap to a1, not b1.
	app.focusedID = a2.ID()
	app.focusNextInScope()

	if app.focusedID != a1.ID() {
		t.Errorf("focusNext from a2: got %v, want a1 (%v)", app.focusedID, a1.ID())
	}
}

func TestFocusPrevInScope_WrapsWithinScope(t *testing.T) {
	app, _ := New(AppOpts{})
	app.size = geom.Size{W: 80, H: 24}

	a1 := newMockNode(true)
	a2 := newMockNode(true)
	scope1 := newMockFocusScope(a1, a2)
	b1 := newMockNode(true)
	root := newMockContainer(scope1, b1)

	app.root = root
	app.rebuildTree()

	// Focus a1 (first in scope1). Prev should wrap to a2.
	app.focusedID = a1.ID()
	app.focusPrevInScope()

	if app.focusedID != a2.ID() {
		t.Errorf("focusPrev from a1: got %v, want a2 (%v)", app.focusedID, a2.ID())
	}
}

func TestFocusRepair_FallsThroughScopeChain(t *testing.T) {
	app, _ := New(AppOpts{})
	app.size = geom.Size{W: 80, H: 24}

	a1 := newMockNode(true)
	scope1 := newMockFocusScope(a1)
	b1 := newMockNode(true)
	root := newMockContainer(scope1, b1)

	app.root = root
	app.rebuildTree()

	// Focus a1, then make it unfocusable to trigger repair.
	app.focusedID = a1.ID()
	a1.focusable = false

	app.rebuildTree()

	app.ensureValidFocus()

	// With a1 unfocusable, scope1 has no focusable children.
	// Repair should fall back to parent scope (root) and find b1.
	if app.focusedID != b1.ID() {
		t.Errorf("focus repair: got %v, want b1 (%v)", app.focusedID, b1.ID())
	}
}

func TestScopeMemory_RestoresLastFocused(t *testing.T) {
	app, _ := New(AppOpts{})
	app.size = geom.Size{W: 80, H: 24}

	a1 := newMockNode(true)
	a2 := newMockNode(true)
	scope1 := newMockFocusScope(a1, a2)
	b1 := newMockNode(true)
	root := newMockContainer(scope1, b1)

	app.root = root
	app.rebuildTree()

	// Focus a2 (records scope memory for scope1).
	app.setRequestFocus(a2.ID())

	// Move focus to b1 (different scope).
	app.setRequestFocus(b1.ID())

	// Verify scope memory was recorded.
	scopeID := scope1.ID()
	if ss, ok := app.scopeMemory[scopeID]; !ok || ss.lastFocused != a2.ID() {
		t.Errorf("scope memory: want a2, got %v", app.scopeMemory[scopeID])
	}
}

func TestModalOverlay_CreatesImplicitScope(t *testing.T) {
	app, _ := New(AppOpts{})
	app.size = geom.Size{W: 80, H: 24}

	mainRoot := newMockNode(true)
	app.root = mainRoot

	overlayChild := newMockNode(true)
	overlayRoot := newMockContainer(overlayChild)

	o := app.overlays.PushOverlay(OverlayOpts{
		Root:  overlayRoot,
		Modal: true,
		Place: &fixedPlacement{geom.Rect{X: 0, Y: 0, W: 10, H: 5}},
	}, 0)

	app.rebuildTree()

	// The overlay child should have focusScopeID set to the overlay's ID
	// (implicit scope from modal overlay).
	entry := app.nodes[overlayChild.ID()]
	if entry.focusScopeID != o.id {
		t.Errorf("modal overlay child scope: got %v, want %v", entry.focusScopeID, o.id)
	}

	// Main tree node should have scope 0.
	mainEntry := app.nodes[mainRoot.ID()]
	if mainEntry.focusScopeID != 0 {
		t.Errorf("main tree scope: got %v, want 0", mainEntry.focusScopeID)
	}
}

func TestNestedScopes(t *testing.T) {
	app, _ := New(AppOpts{})
	app.size = geom.Size{W: 80, H: 24}

	// Inner scope nested inside outer scope.
	inner1 := newMockNode(true)
	inner2 := newMockNode(true)
	innerScope := newMockFocusScope(inner1, inner2)

	outer1 := newMockNode(true)
	outerScope := newMockFocusScope(outer1, innerScope)

	root := newMockContainer(outerScope)
	app.root = root
	app.rebuildTree()

	// inner1 and inner2 should be in innerScope's scope.
	if app.nodes[inner1.ID()].focusScopeID != innerScope.ID() {
		t.Errorf("inner1 scope: got %v, want %v", app.nodes[inner1.ID()].focusScopeID, innerScope.ID())
	}

	// outer1 should be in outerScope's scope.
	if app.nodes[outer1.ID()].focusScopeID != outerScope.ID() {
		t.Errorf("outer1 scope: got %v, want %v", app.nodes[outer1.ID()].focusScopeID, outerScope.ID())
	}

	// Tab from inner1 should stay within inner scope.
	app.focusedID = inner1.ID()
	app.focusNextInScope()

	if app.focusedID != inner2.ID() {
		t.Errorf("tab from inner1: got %v, want inner2 (%v)", app.focusedID, inner2.ID())
	}

	// Tab again should wrap to inner1.
	app.focusNextInScope()

	if app.focusedID != inner1.ID() {
		t.Errorf("tab from inner2: got %v, want inner1 (%v)", app.focusedID, inner1.ID())
	}
}

// When nothing is focused (focusedID not among the scope's targets), Tab must
// land on the first focusable rather than skipping it, and Shift+Tab must land
// on the last. Regression test for indexOf returning 0 ("found at index 0")
// for the not-found case.
func TestFocusNextInScope_FromNoFocus_SelectsFirst(t *testing.T) {
	app, _ := New(AppOpts{})
	app.size = geom.Size{W: 80, H: 24}

	a1 := newMockNode(true)
	a2 := newMockNode(true)
	a3 := newMockNode(true)
	app.root = newMockContainer(a1, a2, a3)
	app.rebuildTree()

	app.focusedID = 0
	app.focusNextInScope()

	if app.focusedID != a1.ID() {
		t.Errorf("focusNext from no focus: got %v, want a1 (%v)", app.focusedID, a1.ID())
	}
}

func TestFocusPrevInScope_FromNoFocus_SelectsLast(t *testing.T) {
	app, _ := New(AppOpts{})
	app.size = geom.Size{W: 80, H: 24}

	a1 := newMockNode(true)
	a2 := newMockNode(true)
	a3 := newMockNode(true)
	app.root = newMockContainer(a1, a2, a3)
	app.rebuildTree()

	app.focusedID = 0
	app.focusPrevInScope()

	if app.focusedID != a3.ID() {
		t.Errorf("focusPrev from no focus: got %v, want a3 (%v)", app.focusedID, a3.ID())
	}
}
