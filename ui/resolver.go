package ui

import (
	"github.com/losinggeneration/tui"
	"github.com/losinggeneration/tui/event"
)

// NewResolver creates a ResolveAction closure that detects KeyContext from
// the focused view and resolves keystrokes via the given keymap.
// The returned closure is suitable for AppOpts.ResolveAction.
func NewResolver(keymap Keymap) func(event.KeyEvent, tui.View) (int, bool) {
	return func(e event.KeyEvent, focused tui.View) (int, bool) {
		ctx := KeyCtxGlobal
		if tim, ok := focused.(TextInputMode); ok && tim.IsTextInputMode() {
			ctx = KeyCtxTextInput
		}

		ks := KeystrokeOf(e)

		action, ok := keymap.Resolve(ctx, ks)
		if !ok {
			return 0, false
		}

		return int(action), true
	}
}

// DefaultAppResolver returns a ResolveAction closure using the DefaultKeymap.
func DefaultAppResolver() func(event.KeyEvent, tui.View) (int, bool) {
	return NewResolver(DefaultKeymap{})
}
