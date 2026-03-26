package sdl

import (
	"testing"

	tevent "github.com/losinggeneration/tui/event"
	gsdl "github.com/veandco/go-sdl2/sdl"
)

func TestMapKeyShiftTab(t *testing.T) {
	key, ok := mapKey(gsdl.K_TAB, gsdl.KMOD_SHIFT)
	if !ok {
		t.Fatal("mapKey(tab, shift) = not ok, want ok")
	}

	if key != tevent.KeyShiftTab {
		t.Fatalf("mapKey(tab, shift) = %v, want %v", key, tevent.KeyShiftTab)
	}
}

func TestWheelEventSteps(t *testing.T) {
	steps, button := wheelEventSteps(gsdl.MouseWheelEvent{Y: 3})
	if steps != 3 || button != tevent.MouseButtonWheelUp {
		t.Fatalf("wheelEventSteps(+3) = (%d, %v), want (3, %v)", steps, button, tevent.MouseButtonWheelUp)
	}

	steps, button = wheelEventSteps(gsdl.MouseWheelEvent{Y: -2})
	if steps != 2 || button != tevent.MouseButtonWheelDown {
		t.Fatalf("wheelEventSteps(-2) = (%d, %v), want (2, %v)", steps, button, tevent.MouseButtonWheelDown)
	}

	steps, button = wheelEventSteps(gsdl.MouseWheelEvent{PreciseY: 0.25})
	if steps != 1 || button != tevent.MouseButtonWheelUp {
		t.Fatalf("wheelEventSteps(precise 0.25) = (%d, %v), want (1, %v)", steps, button, tevent.MouseButtonWheelUp)
	}
}
