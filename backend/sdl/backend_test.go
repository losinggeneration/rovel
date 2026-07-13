package sdl

import (
	"testing"

	"github.com/losinggeneration/rovel/backend"
	tevent "github.com/losinggeneration/rovel/event"
	"github.com/veandco/go-sdl2/sdl"
)

var (
	_ backend.Backend              = (*Backend)(nil)
	_ backend.CapabilityReporter   = (*Backend)(nil)
	_ backend.InputFeatureEnabler  = (*Backend)(nil)
	_ backend.CellFrameSink        = (*Backend)(nil)
	_ backend.ClipboardBackend     = (*Backend)(nil)
	_ backend.ClipboardAsyncReader = (*Backend)(nil)
)

func TestNew(t *testing.T) {
	b, err := New(DefaultOptions())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if b == nil {
		t.Fatalf("New returned nil backend")
	}
}

func TestBackendEnablePresentRestore_DummyVideo(t *testing.T) {
	t.Setenv("SDL_VIDEODRIVER", "dummy")

	bi, err := New(DefaultOptions())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	b := bi.(*Backend)
	if _, err := b.Enable(); err != nil {
		t.Fatalf("Enable: %v", err)
	}

	err = b.PresentCellFrame(backend.CellFrame{
		W: 2,
		H: 1,
		Cells: []backend.FrameCell{
			{R: 'O'},
			{R: 'K'},
		},
	})
	if err != nil {
		t.Fatalf("PresentCellFrame: %v", err)
	}

	if err := b.Restore(); err != nil {
		t.Fatalf("Restore: %v", err)
	}
}

func TestMapKeyShiftTab(t *testing.T) {
	key, ok := mapKey(sdl.K_TAB, sdl.KMOD_SHIFT)
	if !ok {
		t.Fatal("mapKey(tab, shift) = not ok, want ok")
	}

	if key != tevent.KeyShiftTab {
		t.Fatalf("mapKey(tab, shift) = %v, want %v", key, tevent.KeyShiftTab)
	}
}

func TestMapKeyArrows(t *testing.T) {
	tests := []struct {
		sym     sdl.Keycode
		wantKey tevent.Key
	}{
		{sdl.K_UP, tevent.KeyUp},
		{sdl.K_DOWN, tevent.KeyDown},
		{sdl.K_LEFT, tevent.KeyLeft},
		{sdl.K_RIGHT, tevent.KeyRight},
	}

	for _, tc := range tests {
		key, ok := mapKey(tc.sym, 0)
		if !ok {
			t.Errorf("mapKey(%v, 0) = not ok, want ok", tc.sym)

			continue
		}

		if key != tc.wantKey {
			t.Errorf("mapKey(%v, 0) = %v, want %v", tc.sym, key, tc.wantKey)
		}
	}
}

func TestMapKeyNavigation(t *testing.T) {
	tests := []struct {
		sym     sdl.Keycode
		wantKey tevent.Key
	}{
		{sdl.K_HOME, tevent.KeyHome},
		{sdl.K_END, tevent.KeyEnd},
		{sdl.K_INSERT, tevent.KeyInsert},
		{sdl.K_DELETE, tevent.KeyDelete},
		{sdl.K_PAGEUP, tevent.KeyPageUp},
		{sdl.K_PAGEDOWN, tevent.KeyPageDown},
	}

	for _, tc := range tests {
		key, ok := mapKey(tc.sym, 0)
		if !ok {
			t.Errorf("mapKey(%v, 0) = not ok, want ok", tc.sym)

			continue
		}

		if key != tc.wantKey {
			t.Errorf("mapKey(%v, 0) = %v, want %v", tc.sym, key, tc.wantKey)
		}
	}
}

func TestMapKeyFunctionKeys(t *testing.T) {
	tests := []struct {
		sym     sdl.Keycode
		wantKey tevent.Key
	}{
		{sdl.K_F1, tevent.KeyF1},
		{sdl.K_F2, tevent.KeyF2},
		{sdl.K_F3, tevent.KeyF3},
		{sdl.K_F4, tevent.KeyF4},
		{sdl.K_F5, tevent.KeyF5},
		{sdl.K_F6, tevent.KeyF6},
		{sdl.K_F7, tevent.KeyF7},
		{sdl.K_F8, tevent.KeyF8},
		{sdl.K_F9, tevent.KeyF9},
		{sdl.K_F10, tevent.KeyF10},
		{sdl.K_F11, tevent.KeyF11},
		{sdl.K_F12, tevent.KeyF12},
	}

	for _, tc := range tests {
		key, ok := mapKey(tc.sym, 0)
		if !ok {
			t.Errorf("mapKey(%v, 0) = not ok, want ok", tc.sym)

			continue
		}

		if key != tc.wantKey {
			t.Errorf("mapKey(%v, 0) = %v, want %v", tc.sym, key, tc.wantKey)
		}
	}
}

func TestMapKeySpecial(t *testing.T) {
	tests := []struct {
		sym     sdl.Keycode
		wantKey tevent.Key
	}{
		{sdl.K_RETURN, tevent.KeyEnter},
		{sdl.K_RETURN2, tevent.KeyEnter},
		{sdl.K_ESCAPE, tevent.KeyEsc},
		{sdl.K_TAB, tevent.KeyTab},
		{sdl.K_BACKSPACE, tevent.KeyBackspace},
	}

	for _, tc := range tests {
		key, ok := mapKey(tc.sym, 0)
		if !ok {
			t.Errorf("mapKey(%v, 0) = not ok, want ok", tc.sym)

			continue
		}

		if key != tc.wantKey {
			t.Errorf("mapKey(%v, 0) = %v, want %v", tc.sym, key, tc.wantKey)
		}
	}
}

func TestMapKeyNotFound(t *testing.T) {
	tests := []sdl.Keycode{
		sdl.K_LSHIFT,
		sdl.K_RSHIFT,
		sdl.K_LCTRL,
		sdl.K_RCTRL,
		sdl.K_LALT,
		sdl.K_RALT,
		sdl.K_CAPSLOCK,
		sdl.K_SEMICOLON,
		sdl.K_PERIOD,
		sdl.K_COMMA,
		sdl.K_MINUS,
		sdl.K_EQUALS,
		sdl.K_SLASH,
		sdl.K_BACKSLASH,
		sdl.K_BACKQUOTE,
		sdl.K_LEFTBRACKET,
		sdl.K_RIGHTBRACKET,
		sdl.K_QUOTE,
		sdl.K_EXCLAIM,
		sdl.K_AT,
		sdl.K_HASH,
		sdl.K_DOLLAR,
		sdl.K_AMPERSAND,
		sdl.K_ASTERISK,
		sdl.K_LEFTPAREN,
		sdl.K_RIGHTPAREN,
		sdl.K_PLUS,
		sdl.K_LESS,
		sdl.K_GREATER,
		sdl.K_QUESTION,
		sdl.K_COLON,
		sdl.K_UNDERSCORE,
	}

	for _, sym := range tests {
		_, ok := mapKey(sym, 0)
		if ok {
			t.Errorf("mapKey(%v, 0) = ok, want not ok", sym)
		}
	}
}

func TestMapMod(t *testing.T) {
	tests := []struct {
		mod  sdl.Keymod
		want tevent.ModMask
	}{
		{0, 0},
		{sdl.KMOD_SHIFT, tevent.ModShift},
		{sdl.KMOD_ALT, tevent.ModAlt},
		{sdl.KMOD_CTRL, tevent.ModCtrl},
		{sdl.KMOD_SHIFT | sdl.KMOD_ALT, tevent.ModShift | tevent.ModAlt},
		{sdl.KMOD_SHIFT | sdl.KMOD_CTRL, tevent.ModShift | tevent.ModCtrl},
		{sdl.KMOD_ALT | sdl.KMOD_CTRL, tevent.ModAlt | tevent.ModCtrl},
		{sdl.KMOD_SHIFT | sdl.KMOD_ALT | sdl.KMOD_CTRL, tevent.ModShift | tevent.ModAlt | tevent.ModCtrl},
	}

	for _, tc := range tests {
		got := mapMod(tc.mod)
		if got != tc.want {
			t.Errorf("mapMod(%v) = %v, want %v", tc.mod, got, tc.want)
		}
	}
}

func TestMapMouseButton(t *testing.T) {
	tests := []struct {
		btn  sdl.Button
		want tevent.MouseButton
	}{
		{sdl.ButtonLeft, tevent.MouseButtonLeft},
		{sdl.ButtonMiddle, tevent.MouseButtonMiddle},
		{sdl.ButtonRight, tevent.MouseButtonRight},
		{sdl.ButtonX1, tevent.MouseButtonNone},
		{sdl.ButtonX2, tevent.MouseButtonNone},
		{sdl.Button(0), tevent.MouseButtonNone},
		{sdl.Button(99), tevent.MouseButtonNone},
	}

	for _, tc := range tests {
		got := mapMouseButton(tc.btn)
		if got != tc.want {
			t.Errorf("mapMouseButton(%v) = %v, want %v", tc.btn, got, tc.want)
		}
	}
}

func TestWheelEventSteps(t *testing.T) {
	tests := []struct {
		ev         sdl.MouseWheelEvent
		wantSteps  int
		wantButton tevent.MouseButton
	}{
		{sdl.MouseWheelEvent{Y: 3}, 3, tevent.MouseButtonWheelUp},
		{sdl.MouseWheelEvent{Y: -2}, 2, tevent.MouseButtonWheelDown},
		{sdl.MouseWheelEvent{Y: -1}, 1, tevent.MouseButtonWheelDown},
		{sdl.MouseWheelEvent{Y: 1}, 1, tevent.MouseButtonWheelUp},
		{sdl.MouseWheelEvent{Y: 0, PreciseY: 0.25}, 1, tevent.MouseButtonWheelUp},
		{sdl.MouseWheelEvent{Y: 0, PreciseY: -0.25}, 1, tevent.MouseButtonWheelDown},
		{sdl.MouseWheelEvent{Y: 0, PreciseY: 0}, 0, tevent.MouseButtonNone},
	}

	for _, tc := range tests {
		steps, button := wheelEventSteps(tc.ev)
		if steps != tc.wantSteps || button != tc.wantButton {
			t.Errorf("wheelEventSteps(%+v) = (%d, %v), want (%d, %v)", tc.ev, steps, button, tc.wantSteps, tc.wantButton)
		}
	}
}
