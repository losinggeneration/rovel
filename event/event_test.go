package event

import (
	"testing"

	"github.com/losinggeneration/rovel/geom"
)

// Compile-level checks that event types satisfy the Event interface.
var (
	_ Event = KeyEvent{}
	_ Event = ResizeEvent{}
	_ Event = MouseEvent{}
	_ Event = PasteEvent{}
	_ Event = ClipboardResponseEvent{}
)

func TestResizeEvent_Size(t *testing.T) {
	e := ResizeEvent{W: 120, H: 40}
	got := e.Size()

	want := geom.Size{W: 120, H: 40}
	if got != want {
		t.Errorf("Size() = %v, want %v", got, want)
	}
}

func TestKey_ConstantsAreDistinct(t *testing.T) {
	keys := []Key{
		KeyNone, KeyRune, KeyEnter, KeyEsc, KeyTab, KeyBackspace, KeyCtrlC,
		KeyUp, KeyDown, KeyLeft, KeyRight,
		KeyF1, KeyF2, KeyF3, KeyF4,
		KeyHome, KeyEnd, KeyInsert, KeyDelete, KeyPageUp, KeyPageDown,
		KeyF5, KeyF6, KeyF7, KeyF8, KeyF9, KeyF10, KeyF11, KeyF12,
		KeyShiftTab,
	}

	seen := make(map[Key]bool)
	for _, k := range keys {
		if seen[k] {
			t.Errorf("duplicate key constant value: %d", k)
		}

		seen[k] = true
	}
}

func TestModMask_BitsAreDistinct(t *testing.T) {
	if ModShift&ModAlt != 0 {
		t.Error("ModShift and ModAlt overlap")
	}

	if ModShift&ModCtrl != 0 {
		t.Error("ModShift and ModCtrl overlap")
	}

	if ModAlt&ModCtrl != 0 {
		t.Error("ModAlt and ModCtrl overlap")
	}
}

func TestMouseButton_ConstantsAreDistinct(t *testing.T) {
	buttons := []MouseButton{
		MouseButtonNone, MouseButtonLeft, MouseButtonMiddle, MouseButtonRight,
		MouseButtonWheelUp, MouseButtonWheelDown,
	}

	seen := make(map[MouseButton]bool)
	for _, b := range buttons {
		if seen[b] {
			t.Errorf("duplicate MouseButton value: %d", b)
		}

		seen[b] = true
	}
}

func TestMouseAction_ConstantsAreDistinct(t *testing.T) {
	actions := []MouseAction{MousePress, MouseRelease, MouseMove, MouseDrag}

	seen := make(map[MouseAction]bool)
	for _, a := range actions {
		if seen[a] {
			t.Errorf("duplicate MouseAction value: %d", a)
		}

		seen[a] = true
	}
}
