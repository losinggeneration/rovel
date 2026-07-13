package ui

import "github.com/losinggeneration/rovel/event"

// Keystroke represents a normalized key press for keymap lookup.
type Keystroke struct {
	Key  event.Key
	Rune rune
	Mod  event.ModMask
}

// KeystrokeOf converts a KeyEvent to a Keystroke for keymap lookup.
func KeystrokeOf(e event.KeyEvent) Keystroke {
	return Keystroke{
		Key:  e.Key,
		Rune: e.Rune,
		Mod:  e.Mod,
	}
}
