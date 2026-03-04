package tui

import (
	"github.com/losinggeneration/tui/event"
)

// Event types re-exported from event package for API convenience.

// Event is the interface for all events.
type Event = event.Event

// Key represents a key press.
type Key = event.Key

// Key constants re-exported for convenience.
const (
	KeyNone      = event.KeyNone
	KeyRune      = event.KeyRune
	KeyEnter     = event.KeyEnter
	KeyEsc       = event.KeyEsc
	KeyTab       = event.KeyTab
	KeyBackspace = event.KeyBackspace
	KeyUp        = event.KeyUp
	KeyDown      = event.KeyDown
	KeyLeft      = event.KeyLeft
	KeyRight     = event.KeyRight
)

// ModMask represents keyboard modifiers.
type ModMask = event.ModMask

// ModMask constants re-exported for convenience.
const (
	ModShift = event.ModShift
	ModAlt   = event.ModAlt
	ModCtrl  = event.ModCtrl
)

// KeyEvent represents a key press event.
type KeyEvent = event.KeyEvent

// ResizeEvent represents a terminal resize event.
type ResizeEvent = event.ResizeEvent
