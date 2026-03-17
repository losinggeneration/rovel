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
//
// F1–F4 are encoded as SS3 P/Q/R/S sequences.
// F5–F12 are encoded as CSI tilde sequences (e.g. ESC [ 15 ~).
// KeyShiftTab constant exists but decoder wiring is not yet implemented.
const (
	KeyNone      = event.KeyNone
	KeyRune      = event.KeyRune
	KeyEnter     = event.KeyEnter
	KeyEsc       = event.KeyEsc
	KeyTab       = event.KeyTab
	KeyBackspace = event.KeyBackspace
	KeyCtrlC     = event.KeyCtrlC
	KeyUp        = event.KeyUp
	KeyDown      = event.KeyDown
	KeyLeft      = event.KeyLeft
	KeyRight     = event.KeyRight
	KeyF1        = event.KeyF1
	KeyF2        = event.KeyF2
	KeyF3        = event.KeyF3
	KeyF4        = event.KeyF4
	KeyHome      = event.KeyHome
	KeyEnd       = event.KeyEnd
	KeyInsert    = event.KeyInsert
	KeyDelete    = event.KeyDelete
	KeyPageUp    = event.KeyPageUp
	KeyPageDown  = event.KeyPageDown
	KeyF5        = event.KeyF5
	KeyF6        = event.KeyF6
	KeyF7        = event.KeyF7
	KeyF8        = event.KeyF8
	KeyF9        = event.KeyF9
	KeyF10       = event.KeyF10
	KeyF11       = event.KeyF11
	KeyF12       = event.KeyF12
	KeyShiftTab  = event.KeyShiftTab
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

// MouseEvent represents a mouse event.
type MouseEvent = event.MouseEvent

// MouseButton represents a mouse button.
type MouseButton = event.MouseButton

// MouseButton constants re-exported for convenience.
const (
	MouseButtonNone      = event.MouseButtonNone
	MouseButtonLeft      = event.MouseButtonLeft
	MouseButtonMiddle    = event.MouseButtonMiddle
	MouseButtonRight     = event.MouseButtonRight
	MouseButtonWheelUp   = event.MouseButtonWheelUp
	MouseButtonWheelDown = event.MouseButtonWheelDown
)

// MouseAction represents the type of mouse action.
type MouseAction = event.MouseAction

// MouseAction constants re-exported for convenience.
const (
	MousePress   = event.MousePress
	MouseRelease = event.MouseRelease
	MouseMove    = event.MouseMove
	MouseDrag    = event.MouseDrag
)

// PasteEvent represents a bracketed paste event.
type PasteEvent = event.PasteEvent

// ClipboardResponseEvent is emitted when the terminal responds to a clipboard read query.
type ClipboardResponseEvent = event.ClipboardResponseEvent
