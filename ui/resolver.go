package ui

import (
	"github.com/losinggeneration/rovel"
	"github.com/losinggeneration/rovel/event"
)

// NewResolver creates a ResolveAction closure that detects KeyContext from
// the focused view and resolves keystrokes via the given keymap.
// The returned closure is suitable for AppOpts.ResolveAction.
func NewResolver(keymap Keymap) func(event.KeyEvent, rovel.View) (Action, bool) {
	return func(e event.KeyEvent, focused rovel.View) (Action, bool) {
		ctx := KeyCtxGlobal
		if tim, ok := focused.(TextInputMode); ok && tim.IsTextInputMode() {
			ctx = KeyCtxTextInput
		}

		ks := KeystrokeOf(e)

		act, ok := keymap.Resolve(ctx, ks)
		if !ok {
			return ActionNone, false
		}

		return act, true
	}
}

// DefaultAppResolver returns a ResolveAction closure using the DefaultKeymap.
func DefaultAppResolver() func(event.KeyEvent, rovel.View) (Action, bool) {
	return NewResolver(DefaultKeymap{})
}
