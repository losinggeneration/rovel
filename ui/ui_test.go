package ui

import (
	"testing"

	"github.com/losinggeneration/tui/event"
)

func TestKeystrokeOf(t *testing.T) {
	e := event.KeyEvent{Key: event.KeyRune, Rune: 'x', Mod: event.ModCtrl}
	ks := KeystrokeOf(e)
	if ks.Key != event.KeyRune || ks.Rune != 'x' || ks.Mod != event.ModCtrl {
		t.Errorf("KeystrokeOf mismatch: got %+v", ks)
	}
}

func TestDefaultKeymap_GlobalBindings(t *testing.T) {
	km := DefaultKeymap{}

	tests := []struct {
		name   string
		ks     Keystroke
		want   Action
		wantOK bool
	}{
		{"Tab", Keystroke{Key: event.KeyTab}, ActionFocusNext, true},
		{"ShiftTab", Keystroke{Key: event.KeyShiftTab}, ActionFocusPrev, true},
		{"Esc", Keystroke{Key: event.KeyEsc}, ActionCancel, true},
		{"Enter", Keystroke{Key: event.KeyEnter}, ActionActivate, true},
		{"Left", Keystroke{Key: event.KeyLeft}, ActionMoveLeft, true},
		{"Right", Keystroke{Key: event.KeyRight}, ActionMoveRight, true},
		{"Up", Keystroke{Key: event.KeyUp}, ActionMoveUp, true},
		{"Down", Keystroke{Key: event.KeyDown}, ActionMoveDown, true},
		{"PageUp", Keystroke{Key: event.KeyPageUp}, ActionPageUp, true},
		{"PageDown", Keystroke{Key: event.KeyPageDown}, ActionPageDown, true},
		{"Home", Keystroke{Key: event.KeyHome}, ActionHome, true},
		{"End", Keystroke{Key: event.KeyEnd}, ActionEnd, true},
		{"Backspace", Keystroke{Key: event.KeyBackspace}, ActionDeleteBackward, true},
		{"Delete", Keystroke{Key: event.KeyDelete}, ActionDeleteForward, true},
		{"Space", Keystroke{Key: event.KeyRune, Rune: ' '}, ActionActivate, true},
		{"UnboundRune", Keystroke{Key: event.KeyRune, Rune: 'z'}, ActionNone, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := km.Resolve(KeyCtxGlobal, tt.ks)
			if ok != tt.wantOK || got != tt.want {
				t.Errorf("Resolve(Global, %+v) = (%d, %v), want (%d, %v)", tt.ks, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestDefaultKeymap_TextInputBindings(t *testing.T) {
	km := DefaultKeymap{}

	tests := []struct {
		name   string
		ks     Keystroke
		want   Action
		wantOK bool
	}{
		{"Ctrl+A=SelectAll", Keystroke{Key: event.KeyRune, Rune: 'a', Mod: event.ModCtrl}, ActionSelectAll, true},
		{"Ctrl+C=Copy", Keystroke{Key: event.KeyRune, Rune: 'c', Mod: event.ModCtrl}, ActionCopy, true},
		{"Ctrl+X=Cut", Keystroke{Key: event.KeyRune, Rune: 'x', Mod: event.ModCtrl}, ActionCut, true},
		{"KeyCtrlC=Copy", Keystroke{Key: event.KeyCtrlC}, ActionCopy, true},
		{"Enter=Submit", Keystroke{Key: event.KeyEnter}, ActionSubmit, true},
		{"Tab=FocusNext", Keystroke{Key: event.KeyTab}, ActionFocusNext, true},
		{"Esc=Cancel", Keystroke{Key: event.KeyEsc}, ActionCancel, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := km.Resolve(KeyCtxTextInput, tt.ks)
			if ok != tt.wantOK || got != tt.want {
				t.Errorf("Resolve(TextInput, %+v) = (%d, %v), want (%d, %v)", tt.ks, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestCompositeKeymap_FirstMatchWins(t *testing.T) {
	custom := funcKeymap(func(ctx KeyContext, k Keystroke) (Action, bool) {
		if k.Key == event.KeyEnter {
			return ActionSubmit, true
		}
		return ActionNone, false
	})

	composite := CompositeKeymap{
		Keymaps: []Keymap{custom, DefaultKeymap{}},
	}

	// Custom keymap overrides Enter
	got, ok := composite.Resolve(KeyCtxGlobal, Keystroke{Key: event.KeyEnter})
	if !ok || got != ActionSubmit {
		t.Errorf("expected ActionSubmit, got %d", got)
	}

	// Fallback to default for other keys
	got, ok = composite.Resolve(KeyCtxGlobal, Keystroke{Key: event.KeyEsc})
	if !ok || got != ActionCancel {
		t.Errorf("expected ActionCancel from fallback, got %d", got)
	}
}

func TestCompositeKeymap_Empty(t *testing.T) {
	composite := CompositeKeymap{}
	_, ok := composite.Resolve(KeyCtxGlobal, Keystroke{Key: event.KeyEnter})
	if ok {
		t.Error("empty composite should not resolve anything")
	}
}

func TestAction_Constants(t *testing.T) {
	// Verify action constants are distinct.
	actions := []Action{
		ActionNone, ActionFocusNext, ActionFocusPrev, ActionFocusFirst, ActionFocusLast,
		ActionActivate, ActionCancel, ActionSubmit,
		ActionMoveLeft, ActionMoveRight, ActionMoveUp, ActionMoveDown,
		ActionPageUp, ActionPageDown, ActionHome, ActionEnd,
		ActionDeleteBackward, ActionDeleteForward,
		ActionPaste, ActionSelectAll, ActionCopy, ActionCut,
	}
	seen := make(map[Action]bool)
	for _, a := range actions {
		if seen[a] {
			t.Errorf("duplicate Action value: %d", a)
		}
		seen[a] = true
	}
}

func TestKeyContext_Constants(t *testing.T) {
	// Verify contexts are distinct.
	if KeyCtxGlobal == KeyCtxTextInput || KeyCtxGlobal == KeyCtxOverlay || KeyCtxTextInput == KeyCtxOverlay {
		t.Error("KeyContext constants should be distinct")
	}
}

// funcKeymap adapts a function to the Keymap interface for testing.
type funcKeymap func(KeyContext, Keystroke) (Action, bool)

func (f funcKeymap) Resolve(ctx KeyContext, k Keystroke) (Action, bool) {
	return f(ctx, k)
}
