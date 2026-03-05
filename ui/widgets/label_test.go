package widgets

import (
	"testing"

	"github.com/losinggeneration/tui"
)

func TestLabel(t *testing.T) {
	label := NewLabel("test")

	if label.ID() == (tui.ID(0)) {
		t.Fatal("ID should not be zero")
	}

	minSize := label.MinSize()
	if minSize.W != 4 { // "test" is 4 characters
		t.Errorf("Expected MinSize.W to be 4, got %d", minSize.W)
	}
	if minSize.H != 1 {
		t.Errorf("Expected MinSize.H to be 1, got %d", minSize.H)
	}

	rect := tui.Rect{X: 0, Y: 0, W: 10, H: 5}
	label.Layout(rect)
	if label.Rect() != rect {
		t.Errorf("Rect not set correctly")
	}

	if label.Focusable() {
		t.Error("Label should not be focusable")
	}
}

func TestLabelMultiline(t *testing.T) {
	label := NewLabel("line1\nline2")

	minSize := label.MinSize()
	if minSize.H != 2 {
		t.Errorf("Expected MinSize.H to be 2 for multiline label, got %d", minSize.H)
	}
}

func TestButton(t *testing.T) {
	button := NewButton("OK")

	if button.ID() == (tui.ID(0)) {
		t.Fatal("ID should not be zero")
	}

	minSize := button.MinSize()
	// Button needs space for "[ OK ]" which is 6 characters
	if minSize.W < 6 {
		t.Errorf("Expected MinSize.W to be at least 6, got %d", minSize.W)
	}

	if !button.Focusable() {
		t.Error("Button should be focusable")
	}

	// Test callback - now takes ctx parameter
	pressed := false
	button.SetOnPress(func(ctx *tui.Ctx) {
		pressed = true
	})
	button.onPress(nil)
	if !pressed {
		t.Error("Callback was not called")
	}
}

func TestTextInput(t *testing.T) {
	input := NewTextInput()

	if input.ID() == (tui.ID(0)) {
		t.Fatal("ID should not be zero")
	}

	minSize := input.MinSize()
	if minSize.W < 10 || minSize.H != 1 {
		t.Errorf("Unexpected MinSize: %v", minSize)
	}

	if !input.Focusable() {
		t.Error("TextInput should be focusable")
	}

	// Test text setting - now takes ctx parameter
	input.SetText(nil, "hello")
	if input.Text() != "hello" {
		t.Errorf("Expected text to be 'hello', got '%s'", input.Text())
	}

	// Test cursor position
	if input.cursor != 5 {
		t.Errorf("Expected cursor at position 5, got %d", input.cursor)
	}
}

func TestFocusRing(t *testing.T) {
	child := NewLabel("test")
	ring := NewFocusRing(child)

	if ring.ID() == (tui.ID(0)) {
		t.Fatal("ID should not be zero")
	}

	childMin := child.MinSize()
	ringMin := ring.MinSize()

	// Focus ring should add 2 to each dimension for border
	if ringMin.W != childMin.W+2 {
		t.Errorf("Expected ring MinSize.W to be %d, got %d", childMin.W+2, ringMin.W)
	}
	if ringMin.H != childMin.H+2 {
		t.Errorf("Expected ring MinSize.H to be %d, got %d", childMin.H+2, ringMin.H)
	}

	if ring.Focusable() {
		t.Error("FocusRing should not be focusable")
	}
}
