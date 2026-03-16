package tui

import (
	"testing"

	"github.com/losinggeneration/tui/geom"
)

// ---------------------------------------------------------------------------
// Mock view types for node tests
// ---------------------------------------------------------------------------

// mockNode is a simple leaf view.
type mockNode struct {
	id        ID
	rect      geom.Rect
	focusable bool
}

func newMockNode(focusable bool) *mockNode {
	return &mockNode{id: NewID(), focusable: focusable}
}

func (m *mockNode) ID() ID                     { return m.id }
func (m *mockNode) MinSize() geom.Size         { return geom.Size{W: 1, H: 1} }
func (m *mockNode) Layout(r geom.Rect)         { m.rect = r }
func (m *mockNode) Rect() geom.Rect            { return m.rect }
func (m *mockNode) Paint(p *Painter, ctx *Ctx) {}
func (m *mockNode) Handle(e Event, ctx *Ctx) bool { return false }
func (m *mockNode) Focusable() bool               { return m.focusable }

// mockContainer is a view that exposes children.
type mockContainer struct {
	id       ID
	rect     geom.Rect
	children []View
}

func newMockContainer(children ...View) *mockContainer {
	return &mockContainer{id: NewID(), children: children}
}

func (c *mockContainer) ID() ID                     { return c.id }
func (c *mockContainer) MinSize() geom.Size         { return geom.Size{W: 1, H: 1} }
func (c *mockContainer) Layout(r geom.Rect)         { c.rect = r }
func (c *mockContainer) Rect() geom.Rect            { return c.rect }
func (c *mockContainer) Paint(p *Painter, ctx *Ctx) {}
func (c *mockContainer) Handle(e Event, ctx *Ctx) bool { return false }
func (c *mockContainer) Children() []View              { return c.children }

// mockFocusScope is a container that also implements FocusScope.
type mockFocusScope struct {
	mockContainer
}

func newMockFocusScope(children ...View) *mockFocusScope {
	return &mockFocusScope{mockContainer: *newMockContainer(children...)}
}

func (s *mockFocusScope) FocusScope() bool { return true }

// ---------------------------------------------------------------------------
// rebuildTree tests
// ---------------------------------------------------------------------------

func TestRebuildTree_NilRoot(t *testing.T) {
	app, _ := New(AppOpts{})
	// Should not panic with nil root.
	app.rebuildTree()
	if len(app.nodes) != 0 {
		t.Errorf("expected 0 nodes with nil root, got %d", len(app.nodes))
	}
}

func TestRebuildTree_SingleNode(t *testing.T) {
	app, _ := New(AppOpts{})
	v := newMockNode(true)
	app.root = v
	app.rebuildTree()

	if len(app.nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(app.nodes))
	}
	entry, ok := app.nodes[v.ID()]
	if !ok {
		t.Fatal("root node not found in nodes map")
	}
	if entry.parentID != 0 {
		t.Errorf("root parentID should be 0, got %v", entry.parentID)
	}
	if !entry.focusable {
		t.Error("expected root to be focusable")
	}
}

func TestRebuildTree_ParentPointers(t *testing.T) {
	app, _ := New(AppOpts{})
	child1 := newMockNode(false)
	child2 := newMockNode(true)
	root := newMockContainer(child1, child2)
	app.root = root
	app.rebuildTree()

	if len(app.nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(app.nodes))
	}

	rootEntry := app.nodes[root.ID()]
	if rootEntry.parentID != 0 {
		t.Errorf("root parentID should be 0, got %v", rootEntry.parentID)
	}

	c1Entry := app.nodes[child1.ID()]
	if c1Entry.parentID != root.ID() {
		t.Errorf("child1 parentID: got %v, want %v", c1Entry.parentID, root.ID())
	}

	c2Entry := app.nodes[child2.ID()]
	if c2Entry.parentID != root.ID() {
		t.Errorf("child2 parentID: got %v, want %v", c2Entry.parentID, root.ID())
	}
}

func TestRebuildTree_ChildOrder(t *testing.T) {
	app, _ := New(AppOpts{})
	children := make([]View, 4)
	for i := range children {
		children[i] = newMockNode(false)
	}
	root := newMockContainer(children...)
	app.root = root
	app.rebuildTree()

	rootEntry := app.nodes[root.ID()]
	if len(rootEntry.childIDs) != 4 {
		t.Fatalf("expected 4 child IDs, got %d", len(rootEntry.childIDs))
	}
	for i, child := range children {
		if rootEntry.childIDs[i] != child.ID() {
			t.Errorf("child order mismatch at %d: got %v, want %v", i, rootEntry.childIDs[i], child.ID())
		}
	}
}

func TestRebuildTree_FocusableFlag(t *testing.T) {
	app, _ := New(AppOpts{})
	focusable := newMockNode(true)
	notFocusable := newMockNode(false)
	root := newMockContainer(focusable, notFocusable)
	app.root = root
	app.rebuildTree()

	if !app.nodes[focusable.ID()].focusable {
		t.Error("focusable node should have focusable=true")
	}
	if app.nodes[notFocusable.ID()].focusable {
		t.Error("non-focusable node should have focusable=false")
	}
}

func TestRebuildTree_FocusScopeMembership(t *testing.T) {
	app, _ := New(AppOpts{})
	inner := newMockNode(true)
	scope := newMockFocusScope(inner)
	outer := newMockNode(false)
	root := newMockContainer(scope, outer)
	app.root = root
	app.rebuildTree()

	// The scope itself inherits the parent scope (0 for root children).
	scopeEntry := app.nodes[scope.ID()]
	if scopeEntry.focusScopeID != 0 {
		t.Errorf("scope node focusScopeID: got %v, want 0 (parent scope)", scopeEntry.focusScopeID)
	}

	// inner is a child of scope; its focusScopeID should be scope.ID()
	// because scope declares FocusScope() == true.
	innerEntry := app.nodes[inner.ID()]
	if innerEntry.focusScopeID != scope.ID() {
		t.Errorf("inner focusScopeID: got %v, want %v", innerEntry.focusScopeID, scope.ID())
	}

	// outer is a sibling of scope; its focusScopeID should remain 0.
	outerEntry := app.nodes[outer.ID()]
	if outerEntry.focusScopeID != 0 {
		t.Errorf("outer focusScopeID: got %v, want 0", outerEntry.focusScopeID)
	}
}

func TestRebuildTree_OverlayTagging(t *testing.T) {
	app, _ := New(AppOpts{})
	mainRoot := newMockNode(false)
	app.root = mainRoot

	overlayRoot := newMockNode(true)
	o := app.overlays.PushOverlay(OverlayOpts{
		Root:  overlayRoot,
		Modal: false,
		Place: &fixedPlacement{geom.Rect{X: 0, Y: 0, W: 10, H: 5}},
	}, 0)

	app.rebuildTree()

	// Main tree node should have overlayID == 0.
	mainEntry := app.nodes[mainRoot.ID()]
	if mainEntry.overlayID != 0 {
		t.Errorf("main node overlayID: got %v, want 0", mainEntry.overlayID)
	}

	// Overlay node should have overlayID == overlay's ID.
	overlayEntry := app.nodes[overlayRoot.ID()]
	if overlayEntry.overlayID != o.id {
		t.Errorf("overlay node overlayID: got %v, want %v", overlayEntry.overlayID, o.id)
	}
}

// fixedPlacement implements Placement for tests.
type fixedPlacement struct {
	rect geom.Rect
}

func (f *fixedPlacement) Resolve(_ View, _ geom.Size) geom.Rect { return f.rect }

// ---------------------------------------------------------------------------
// ancestorIDs tests
// ---------------------------------------------------------------------------

func TestAncestorIDs_Root(t *testing.T) {
	app, _ := New(AppOpts{})
	root := newMockNode(false)
	app.root = root
	app.rebuildTree()

	ids := app.ancestorIDs(root.ID())
	if len(ids) != 0 {
		t.Errorf("root should have no ancestors, got %v", ids)
	}
}

func TestAncestorIDs_Chain(t *testing.T) {
	app, _ := New(AppOpts{})
	leaf := newMockNode(false)
	mid := newMockContainer(leaf)
	root := newMockContainer(mid)
	app.root = root
	app.rebuildTree()

	ids := app.ancestorIDs(leaf.ID())
	if len(ids) != 2 {
		t.Fatalf("leaf should have 2 ancestors, got %d: %v", len(ids), ids)
	}
	if ids[0] != mid.ID() {
		t.Errorf("first ancestor should be mid, got %v", ids[0])
	}
	if ids[1] != root.ID() {
		t.Errorf("second ancestor should be root, got %v", ids[1])
	}
}

// ---------------------------------------------------------------------------
// updateBounds / clip computation tests
// ---------------------------------------------------------------------------

func TestUpdateBounds_BasicRect(t *testing.T) {
	app, _ := New(AppOpts{})
	app.size = geom.Size{W: 80, H: 24}
	v := newMockNode(false)
	app.root = v
	app.rebuildTree()

	// Simulate layout placing the view at a known rect.
	v.rect = geom.Rect{X: 5, Y: 3, W: 10, H: 4}
	app.updateBounds()

	entry := app.nodes[v.ID()]
	if entry.rect != v.rect {
		t.Errorf("rect: got %v, want %v", entry.rect, v.rect)
	}
	// clipRect = intersection of rect with screen (0,0,80,24) = rect itself.
	if entry.clipRect != v.rect {
		t.Errorf("clipRect: got %v, want %v", entry.clipRect, v.rect)
	}
}

func TestUpdateBounds_ChildClippedByParent(t *testing.T) {
	app, _ := New(AppOpts{})
	app.size = geom.Size{W: 20, H: 10}

	child := newMockNode(false)
	container := newMockContainer(child)
	app.root = container
	app.rebuildTree()

	// Parent occupies a small rect. Child extends beyond it (simulating
	// a child positioned partially outside parent).
	container.rect = geom.Rect{X: 5, Y: 2, W: 8, H: 4}
	child.rect = geom.Rect{X: 5, Y: 2, W: 12, H: 4} // wider than parent

	app.updateBounds()

	parentEntry := app.nodes[container.ID()]
	childEntry := app.nodes[child.ID()]

	// Parent clipRect = rect ∩ screen = rect.
	if parentEntry.clipRect != container.rect {
		t.Errorf("parent clipRect: got %v, want %v", parentEntry.clipRect, container.rect)
	}

	// Child clipRect = child.rect ∩ parent.clipRect = (5,2,8,4).
	wantChildClip := geom.Rect{X: 5, Y: 2, W: 8, H: 4}
	if childEntry.clipRect != wantChildClip {
		t.Errorf("child clipRect: got %v, want %v", childEntry.clipRect, wantChildClip)
	}
}

// ---------------------------------------------------------------------------
// Dynamic children test (simulating Tabs switching content)
// ---------------------------------------------------------------------------

// dynamicContainer switches its child on demand.
type dynamicContainer struct {
	id      ID
	rect    geom.Rect
	current View
}

func newDynamicContainer(initial View) *dynamicContainer {
	return &dynamicContainer{id: NewID(), current: initial}
}

func (d *dynamicContainer) ID() ID                     { return d.id }
func (d *dynamicContainer) MinSize() geom.Size         { return geom.Size{W: 1, H: 1} }
func (d *dynamicContainer) Layout(r geom.Rect)         { d.rect = r }
func (d *dynamicContainer) Rect() geom.Rect            { return d.rect }
func (d *dynamicContainer) Paint(p *Painter, ctx *Ctx) {}
func (d *dynamicContainer) Handle(e Event, ctx *Ctx) bool { return false }
func (d *dynamicContainer) Children() []View              { return []View{d.current} }

func TestRebuildTree_DynamicChildren(t *testing.T) {
	app, _ := New(AppOpts{})
	tab1 := newMockNode(true)
	tab2 := newMockNode(false)

	container := newDynamicContainer(tab1)
	app.root = container

	// First build — tab1 is active.
	app.rebuildTree()
	if _, ok := app.nodes[tab1.ID()]; !ok {
		t.Fatal("tab1 should be in nodes on first build")
	}
	if _, ok := app.nodes[tab2.ID()]; ok {
		t.Error("tab2 should NOT be in nodes before switch")
	}

	// Switch to tab2 and rebuild.
	container.current = tab2
	app.rebuildTree()

	if _, ok := app.nodes[tab2.ID()]; !ok {
		t.Fatal("tab2 should be in nodes after switch")
	}
	if _, ok := app.nodes[tab1.ID()]; ok {
		t.Error("tab1 should NOT be in nodes after switch")
	}
}

