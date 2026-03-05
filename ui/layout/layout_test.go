package layout

import (
	"testing"

	"github.com/losinggeneration/tui"
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

func (v *simpleView) Paint(p *tui.Painter, ctx *tui.Ctx)    {}
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
