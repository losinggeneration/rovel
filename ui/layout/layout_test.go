package layout

import (
	"testing"

	"github.com/losinggeneration/tui"
	"github.com/losinggeneration/tui/event"
	"github.com/losinggeneration/tui/geom"
)

// simpleView is a minimal view implementation for testing
type simpleView struct {
	id        tui.ID
	rect      tui.Rect
	minSize   tui.Size
	focusable bool
}

func newSimpleView(w, h int, focusable bool) *simpleView {
	return &simpleView{
		id:        tui.NewID(),
		minSize:   tui.Size{W: w, H: h},
		focusable: focusable,
	}
}

func (v *simpleView) ID() tui.ID           { return v.id }
func (v *simpleView) Rect() tui.Rect       { return v.rect }
func (v *simpleView) Layout(r tui.Rect)    { v.rect = r }
func (v *simpleView) MinSize() tui.Size    { return v.minSize }
func (v *simpleView) Focusable() bool      { return v.focusable }
func (v *simpleView) Children() []tui.View { return nil }

func (v *simpleView) Paint(d tui.Drawer, ctx *tui.Ctx)      {}
func (v *simpleView) Handle(e tui.Event, ctx *tui.Ctx) bool { return false }

func TestPadding(t *testing.T) {
	child := newSimpleView(10, 5, false)
	padding := NewPadding(child)
	padding.SetInsets(1, 2, 3, 4)

	if padding.ID() == (tui.ID(0)) {
		t.Fatal("ID should not be zero")
	}

	childMin := child.MinSize()
	paddingMin := padding.MinSize()

	// Padding should add insets to min size
	if paddingMin.W != childMin.W+1+3 {
		t.Errorf("Expected padding MinSize.W to be %d, got %d", childMin.W+4, paddingMin.W)
	}

	if paddingMin.H != childMin.H+2+4 {
		t.Errorf("Expected padding MinSize.H to be %d, got %d", childMin.H+6, paddingMin.H)
	}

	if padding.Focusable() {
		t.Error("Padding should not be focusable")
	}
}

func TestBorder(t *testing.T) {
	child := newSimpleView(10, 5, false)
	border := NewBorder(child)

	if border.ID() == (tui.ID(0)) {
		t.Fatal("ID should not be zero")
	}

	childMin := child.MinSize()
	borderMin := border.MinSize()

	// Border should add 2 to each dimension
	if borderMin.W != childMin.W+2 {
		t.Errorf("Expected border MinSize.W to be %d, got %d", childMin.W+2, borderMin.W)
	}

	if borderMin.H != childMin.H+2 {
		t.Errorf("Expected border MinSize.H to be %d, got %d", childMin.H+2, borderMin.H)
	}

	border.SetTitle("Title")

	if border.title != "Title" {
		t.Error("Title not set")
	}

	if border.Focusable() {
		t.Error("Border should not be focusable")
	}
}

func TestVStack(t *testing.T) {
	stack := NewVStack()

	child1 := newSimpleView(10, 1, false)
	child2 := newSimpleView(15, 1, false)

	stack.Add(child1)
	stack.Add(child2)

	if stack.ID() == (tui.ID(0)) {
		t.Fatal("ID should not be zero")
	}

	minSize := stack.MinSize()
	// VStack: max width, sum of heights
	if minSize.W != 15 { // max(10, 15)
		t.Errorf("Expected MinSize.W to be 15, got %d", minSize.W)
	}

	if minSize.H != 2 { // 1 + 1
		t.Errorf("Expected MinSize.H to be 2, got %d", minSize.H)
	}

	if stack.Focusable() {
		t.Error("VStack should not be focusable")
	}
}

func TestHStack(t *testing.T) {
	stack := NewHStack()

	child1 := newSimpleView(5, 1, false)
	child2 := newSimpleView(10, 1, false)

	stack.Add(child1)
	stack.Add(child2)

	if stack.ID() == (tui.ID(0)) {
		t.Fatal("ID should not be zero")
	}

	minSize := stack.MinSize()
	// HStack: sum of widths, max height
	if minSize.W != 15 { // 5 + 10
		t.Errorf("Expected MinSize.W to be 15, got %d", minSize.W)
	}

	if minSize.H != 1 { // max(1, 1)
		t.Errorf("Expected MinSize.H to be 1, got %d", minSize.H)
	}

	if stack.Focusable() {
		t.Error("HStack should not be focusable")
	}
}

func TestSplit(t *testing.T) {
	split := NewSplit(Horizontal)

	child1 := newSimpleView(4, 1, false)
	child2 := newSimpleView(5, 1, false)

	split.SetFirst(child1)
	split.SetSecond(child2)

	if split.ID() == (tui.ID(0)) {
		t.Fatal("ID should not be zero")
	}

	minSize := split.MinSize()
	// Horizontal split: sum of widths, max height
	if minSize.W != 9 { // 4 + 5
		t.Errorf("Expected MinSize.W to be 9, got %d", minSize.W)
	}

	if minSize.H != 1 {
		t.Errorf("Expected MinSize.H to be 1, got %d", minSize.H)
	}

	// Test ratio
	split.SetRatio(0.7)

	if split.ratio != 0.7 {
		t.Errorf("Expected ratio to be 0.7, got %f", split.ratio)
	}

	if split.Focusable() {
		t.Error("Split should not be focusable")
	}
}

func TestInsetRect(t *testing.T) {
	r := tui.Rect{X: 10, Y: 20, W: 100, H: 50}

	// Test basic inset
	inner := InsetRect(r, 5, 3, 7, 4)
	if inner.X != 15 || inner.Y != 23 || inner.W != 88 || inner.H != 43 {
		t.Errorf("Unexpected inset rect: %v", inner)
	}

	// Test zero inset
	zeroInset := InsetRect(r, 0, 0, 0, 0)
	if zeroInset != r {
		t.Errorf("Zero inset should equal original rect")
	}
}

// Test Focus Navigation

// trackInvalidationView is a test view that tracks invalidation calls
type trackInvalidationView struct {
	*simpleView

	invalidated bool
}

func newTrackInvalidationView(w, h int, focusable bool) *trackInvalidationView {
	return &trackInvalidationView{
		simpleView: newSimpleView(w, h, focusable),
	}
}

func (v *trackInvalidationView) Paint(d tui.Drawer, ctx *tui.Ctx) {
	if ctx.Invalidate != nil {
		ctx.Invalidate(v.rect)
		v.invalidated = true
	}
}

func TestVStackFocusNavigation(t *testing.T) {
	tests := []struct {
		name           string
		childCount     int
		initialFocus   tui.ID // zero ID means no initial focus
		key            event.Key
		expectFocusIdx int // -1 means no focus change
		expectHandled  bool
	}{
		{
			name:           "Tab with no focus focuses first child",
			childCount:     3,
			initialFocus:   tui.ID(0),
			key:            event.KeyTab,
			expectFocusIdx: 0,
			expectHandled:  true,
		},
		{
			name:           "Tab moves to next child",
			childCount:     3,
			initialFocus:   0, // Will be set to first child's ID
			key:            event.KeyTab,
			expectFocusIdx: 1,
			expectHandled:  true,
		},
		{
			name:           "Tab at last item bubbles to parent",
			childCount:     2,
			initialFocus:   0, // Will be set to last child's ID
			key:            event.KeyTab,
			expectFocusIdx: -1, // No focus change
			expectHandled:  false,
		},
		{
			name:           "Shift+Tab with no focus focuses last child",
			childCount:     3,
			initialFocus:   tui.ID(0),
			key:            event.KeyShiftTab,
			expectFocusIdx: 2,
			expectHandled:  true,
		},
		{
			name:           "Shift+Tab moves to previous child",
			childCount:     3,
			initialFocus:   0, // Will be set to middle child's ID
			key:            event.KeyShiftTab,
			expectFocusIdx: 1,
			expectHandled:  true,
		},
		{
			name:           "Shift+Tab at first item bubbles to parent",
			childCount:     2,
			initialFocus:   0, // Will be set to first child's ID
			key:            event.KeyShiftTab,
			expectFocusIdx: -1, // No focus change
			expectHandled:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stack := NewVStack()

			// Create focusable children
			var children []*trackInvalidationView

			for range tt.childCount {
				child := newTrackInvalidationView(10, 1, true)
				children = append(children, child)
				stack.Add(child)
			}

			// Layout the stack
			stack.Layout(geom.Rect{X: 0, Y: 0, W: 100, H: 100})

			// Set up context
			var focusedID tui.ID

			ctx := &tui.Ctx{
				RequestFocus: func(id tui.ID) { focusedID = id },
				Invalidate:   func(r geom.Rect) {},
			}

			// Set initial focus if specified
			if tt.initialFocus != 0 && tt.expectFocusIdx >= 0 {
				if tt.expectFocusIdx < len(children) {
					focusedID = children[tt.expectFocusIdx].ID()
					ctx.FocusedID = focusedID
				}
			} else {
				ctx.FocusedID = tt.initialFocus
			}

			// For "moves to next/previous" tests, set up specific initial focus
			if tt.name == "Tab moves to next child" {
				focusedID = children[0].ID()
				ctx.FocusedID = focusedID
			}

			if tt.name == "Shift+Tab moves to previous child" {
				focusedID = children[2].ID()
				ctx.FocusedID = focusedID
			}

			if tt.name == "Tab at last item bubbles to parent" {
				focusedID = children[1].ID()
				ctx.FocusedID = focusedID
			}

			if tt.name == "Shift+Tab at first item bubbles to parent" {
				focusedID = children[0].ID()
				ctx.FocusedID = focusedID
			}

			// Send the key event
			ke := event.KeyEvent{Key: tt.key}
			handled := stack.Handle(ke, ctx)

			// Check if the event was handled as expected
			if handled != tt.expectHandled {
				t.Errorf("Expected handled=%v, got %v", tt.expectHandled, handled)
			}

			// Check focus moved to expected child
			if tt.expectFocusIdx >= 0 && tt.expectFocusIdx < len(children) {
				expectedID := children[tt.expectFocusIdx].ID()
				if focusedID != expectedID {
					t.Errorf("Expected focus on child %d (ID=%v), got ID=%v",
						tt.expectFocusIdx, expectedID, focusedID)
				}
			} else if tt.expectFocusIdx == -1 {
				// Focus should not change when bubbling
				if focusedID != 0 && focusedID != ctx.FocusedID {
					t.Errorf("Focus should not change when bubbling, got %v", focusedID)
				}
			}
		})
	}
}

func TestHStackFocusNavigation(t *testing.T) {
	tests := []struct {
		name           string
		childCount     int
		key            event.Key
		expectFocusIdx int // -1 means no focus change
		expectHandled  bool
	}{
		{
			name:           "Tab with no focus focuses first child",
			childCount:     3,
			key:            event.KeyTab,
			expectFocusIdx: 0,
			expectHandled:  true,
		},
		{
			name:           "Shift+Tab with no focus focuses last child",
			childCount:     3,
			key:            event.KeyShiftTab,
			expectFocusIdx: 2,
			expectHandled:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stack := NewHStack()

			// Create focusable children
			var children []*trackInvalidationView

			for range tt.childCount {
				child := newTrackInvalidationView(10, 1, true)
				children = append(children, child)
				stack.Add(child)
			}

			// Layout the stack
			stack.Layout(geom.Rect{X: 0, Y: 0, W: 100, H: 100})

			// Set up context
			var focusedID tui.ID

			ctx := &tui.Ctx{
				RequestFocus: func(id tui.ID) { focusedID = id },
				Invalidate:   func(r geom.Rect) {},
			}

			// Send the key event
			ke := event.KeyEvent{Key: tt.key}
			handled := stack.Handle(ke, ctx)

			// Check if the event was handled as expected
			if handled != tt.expectHandled {
				t.Errorf("Expected handled=%v, got %v", tt.expectHandled, handled)
			}

			// Check focus moved to expected child
			if tt.expectFocusIdx >= 0 && tt.expectFocusIdx < len(children) {
				expectedID := children[tt.expectFocusIdx].ID()
				if focusedID != expectedID {
					t.Errorf("Expected focus on child %d (ID=%v), got ID=%v",
						tt.expectFocusIdx, expectedID, focusedID)
				}
			}
		})
	}
}

func TestVStackFocusNavigationNested(t *testing.T) {
	// Test nested containers to verify focus traversal works through hierarchy
	outerStack := NewVStack()

	// Create two inner HStacks, each with two focusable children
	inner1 := NewHStack()
	child1 := newSimpleView(10, 1, true)
	child2 := newSimpleView(10, 1, true)

	inner1.Add(child1)
	inner1.Add(child2)

	inner2 := NewHStack()
	child3 := newSimpleView(10, 1, true)
	child4 := newSimpleView(10, 1, true)

	inner2.Add(child3)
	inner2.Add(child4)

	outerStack.Add(inner1)
	outerStack.Add(inner2)

	// Layout
	outerStack.Layout(geom.Rect{X: 0, Y: 0, W: 100, H: 100})

	// Set up context
	var focusedID tui.ID

	ctx := &tui.Ctx{
		RequestFocus: func(id tui.ID) { focusedID = id },
		Invalidate:   func(r geom.Rect) {},
	}

	// Tab should focus first child (child1)
	ke := event.KeyEvent{Key: event.KeyTab}

	handled := outerStack.Handle(ke, ctx)
	if !handled {
		t.Error("Tab should be handled when no focus exists")
	}

	if focusedID != child1.ID() {
		t.Errorf("Expected focus on child1, got %v", focusedID)
	}

	// Another Tab should move to child2
	ctx.FocusedID = focusedID

	handled = outerStack.Handle(ke, ctx)
	if !handled {
		t.Error("Tab should be handled moving to next sibling")
	}

	if focusedID != child2.ID() {
		t.Errorf("Expected focus on child2, got %v", focusedID)
	}
}

func TestFocusTraversalBoundaryBehavior(t *testing.T) {
	t.Run("VStack Tab at last item bubbles to parent", func(t *testing.T) {
		stack := NewVStack()
		child1 := newSimpleView(10, 1, true)
		child2 := newSimpleView(10, 1, true)

		stack.Add(child1)
		stack.Add(child2)
		stack.Layout(geom.Rect{X: 0, Y: 0, W: 100, H: 100})

		var focusedID tui.ID

		ctx := &tui.Ctx{
			FocusedID:    child2.ID(),
			RequestFocus: func(id tui.ID) { focusedID = id },
			Invalidate:   func(r geom.Rect) {},
		}

		ke := event.KeyEvent{Key: event.KeyTab}
		handled := stack.Handle(ke, ctx)

		if handled != false {
			t.Error("Tab at last item should return false to bubble")
		}

		if focusedID == child1.ID() {
			t.Error("Tab at last item should NOT wrap locally")
		}
	})

	t.Run("VStack Shift+Tab at first item bubbles to parent", func(t *testing.T) {
		stack := NewVStack()
		child1 := newSimpleView(10, 1, true)
		child2 := newSimpleView(10, 1, true)

		stack.Add(child1)
		stack.Add(child2)
		stack.Layout(geom.Rect{X: 0, Y: 0, W: 100, H: 100})

		var focusedID tui.ID

		ctx := &tui.Ctx{
			FocusedID:    child1.ID(),
			RequestFocus: func(id tui.ID) { focusedID = id },
			Invalidate:   func(r geom.Rect) {},
		}

		ke := event.KeyEvent{Key: event.KeyShiftTab}
		handled := stack.Handle(ke, ctx)

		if handled != false {
			t.Error("Shift+Tab at first item should return false to bubble")
		}

		if focusedID == child2.ID() {
			t.Error("Shift+Tab at first item should NOT wrap locally")
		}
	})

	t.Run("HStack Tab at last item bubbles to parent", func(t *testing.T) {
		stack := NewHStack()
		child1 := newSimpleView(10, 1, true)
		child2 := newSimpleView(10, 1, true)

		stack.Add(child1)
		stack.Add(child2)
		stack.Layout(geom.Rect{X: 0, Y: 0, W: 100, H: 100})

		ctx := &tui.Ctx{
			FocusedID:    child2.ID(),
			RequestFocus: func(id tui.ID) {},
			Invalidate:   func(r geom.Rect) {},
		}

		ke := event.KeyEvent{Key: event.KeyTab}
		handled := stack.Handle(ke, ctx)

		if handled != false {
			t.Error("Tab at last item should bubble")
		}
	})

	t.Run("HStack Shift+Tab at first item bubbles to parent", func(t *testing.T) {
		stack := NewHStack()
		child1 := newSimpleView(10, 1, true)
		child2 := newSimpleView(10, 1, true)

		stack.Add(child1)
		stack.Add(child2)
		stack.Layout(geom.Rect{X: 0, Y: 0, W: 100, H: 100})

		ctx := &tui.Ctx{
			FocusedID:    child1.ID(),
			RequestFocus: func(id tui.ID) {},
			Invalidate:   func(r geom.Rect) {},
		}

		ke := event.KeyEvent{Key: event.KeyShiftTab}
		handled := stack.Handle(ke, ctx)

		if handled != false {
			t.Error("Shift+Tab at first item should bubble")
		}
	})

	t.Run("Single focusable container allows bubbling both directions", func(t *testing.T) {
		stack := NewVStack()
		child := newSimpleView(10, 1, true)
		stack.Add(child)
		stack.Layout(geom.Rect{X: 0, Y: 0, W: 100, H: 100})

		ctx := &tui.Ctx{
			FocusedID:    child.ID(),
			RequestFocus: func(id tui.ID) {},
			Invalidate:   func(r geom.Rect) {},
		}

		ke := event.KeyEvent{Key: event.KeyTab}

		handled := stack.Handle(ke, ctx)
		if handled != false {
			t.Error("Tab in single-focusable container should bubble")
		}

		ctx.FocusedID = child.ID()
		ke = event.KeyEvent{Key: event.KeyShiftTab}

		handled = stack.Handle(ke, ctx)
		if handled != false {
			t.Error("Shift+Tab in single-focusable container should bubble")
		}
	})

	t.Run("Nested containers: Tab from first inner bubbles, outer moves to second", func(t *testing.T) {
		outer := NewVStack()

		inner1 := NewHStack()
		child1 := newSimpleView(10, 1, true)
		inner1.Add(child1)

		inner2 := NewHStack()
		child2 := newSimpleView(10, 1, true)
		inner2.Add(child2)

		outer.Add(inner1)
		outer.Add(inner2)
		outer.Layout(geom.Rect{X: 0, Y: 0, W: 100, H: 100})

		var focusedID tui.ID

		ctx := &tui.Ctx{
			FocusedID:    child1.ID(),
			RequestFocus: func(id tui.ID) { focusedID = id },
			Invalidate:   func(r geom.Rect) {},
		}

		ke := event.KeyEvent{Key: event.KeyTab}
		handled := outer.Handle(ke, ctx)

		if handled != true {
			t.Error("Outer should handle Tab traversal")
		}

		if focusedID != child2.ID() {
			t.Errorf("Expected focus on child2, got %v", focusedID)
		}
	})

	t.Run("Nested containers: Shift+Tab from second inner bubbles, outer moves to first", func(t *testing.T) {
		outer := NewVStack()

		inner1 := NewHStack()
		child1 := newSimpleView(10, 1, true)
		inner1.Add(child1)

		inner2 := NewHStack()
		child2 := newSimpleView(10, 1, true)
		inner2.Add(child2)

		outer.Add(inner1)
		outer.Add(inner2)
		outer.Layout(geom.Rect{X: 0, Y: 0, W: 100, H: 100})

		var focusedID tui.ID

		ctx := &tui.Ctx{
			FocusedID:    child2.ID(),
			RequestFocus: func(id tui.ID) { focusedID = id },
			Invalidate:   func(r geom.Rect) {},
		}

		ke := event.KeyEvent{Key: event.KeyShiftTab}
		handled := outer.Handle(ke, ctx)

		if handled != true {
			t.Error("Outer should handle Shift+Tab traversal")
		}

		if focusedID != child1.ID() {
			t.Errorf("Expected focus on child1, got %v", focusedID)
		}
	})
}
