package ui

import "github.com/losinggeneration/rovel/event"

// Keymap resolves keystrokes to actions in a given context.
type Keymap interface {
	Resolve(ctx KeyContext, k Keystroke) (Action, bool)
}

// DefaultKeymap provides standard key bindings.
type DefaultKeymap struct{}

func (DefaultKeymap) Resolve(ctx KeyContext, k Keystroke) (Action, bool) {
	// In text input mode, only bind a minimal set
	if ctx == KeyCtxTextInput {
		return defaultTextInputBinding(k)
	}

	return defaultGlobalBinding(k)
}

func defaultGlobalBinding(k Keystroke) (Action, bool) {
	switch k.Key {
	case event.KeyTab:
		if k.Mod == 0 {
			return ActionFocusNext, true
		}
	case event.KeyShiftTab:
		return ActionFocusPrev, true
	case event.KeyEsc:
		return ActionCancel, true
	case event.KeyEnter:
		return ActionActivate, true
	case event.KeyLeft:
		return ActionMoveLeft, true
	case event.KeyRight:
		return ActionMoveRight, true
	case event.KeyUp:
		return ActionMoveUp, true
	case event.KeyDown:
		return ActionMoveDown, true
	case event.KeyPageUp:
		return ActionPageUp, true
	case event.KeyPageDown:
		return ActionPageDown, true
	case event.KeyHome:
		return ActionHome, true
	case event.KeyEnd:
		return ActionEnd, true
	case event.KeyBackspace:
		return ActionDeleteBackward, true
	case event.KeyDelete:
		return ActionDeleteForward, true
	case event.KeyRune:
		if k.Rune == ' ' && k.Mod == 0 {
			return ActionActivate, true
		}
	default:
	}

	return ActionNone, false
}

func defaultTextInputBinding(k Keystroke) (Action, bool) {
	// Ctrl+key bindings
	if k.Mod == event.ModCtrl && k.Key == event.KeyRune {
		switch k.Rune {
		case 'a':
			return ActionSelectAll, true
		case 'c':
			return ActionCopy, true
		case 'x':
			return ActionCut, true
		}
	}

	// KeyCtrlC is emitted as a distinct key by some parsers; treat as copy
	// in text input context.
	if k.Key == event.KeyCtrlC {
		return ActionCopy, true
	}

	switch k.Key {
	case event.KeyTab:
		if k.Mod == 0 {
			return ActionFocusNext, true
		}
	case event.KeyShiftTab:
		return ActionFocusPrev, true
	case event.KeyEsc:
		return ActionCancel, true
	case event.KeyEnter:
		return ActionSubmit, true
	case event.KeyLeft:
		return ActionMoveLeft, true
	case event.KeyRight:
		return ActionMoveRight, true
	case event.KeyUp:
		return ActionMoveUp, true
	case event.KeyDown:
		return ActionMoveDown, true
	case event.KeyPageUp:
		return ActionPageUp, true
	case event.KeyPageDown:
		return ActionPageDown, true
	case event.KeyBackspace:
		return ActionDeleteBackward, true
	case event.KeyDelete:
		return ActionDeleteForward, true
	case event.KeyHome:
		return ActionHome, true
	case event.KeyEnd:
		return ActionEnd, true
	default:
	}

	return ActionNone, false
}

// CompositeKeymap chains multiple keymaps. First match wins.
type CompositeKeymap struct {
	Keymaps []Keymap
}

func (c CompositeKeymap) Resolve(ctx KeyContext, k Keystroke) (Action, bool) {
	for _, km := range c.Keymaps {
		if a, ok := km.Resolve(ctx, k); ok {
			return a, true
		}
	}

	return ActionNone, false
}
