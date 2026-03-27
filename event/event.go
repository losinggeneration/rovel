// Package event provides shared event types for the tui library.
//
// This package is unstable before v0.1.0.
package event

import "github.com/losinggeneration/tui/geom"

// Event is the interface for all events.
type Event interface {
	isEvent()
}

// Key represents a key press.
type Key int

const (
	KeyNone Key = iota

	// KeyRune is a special key for printable characters.
	// The actual rune is in KeyEvent.Rune.
	KeyRune

	// Special keys
	KeyEnter
	KeyEsc
	KeyTab
	KeyBackspace
	KeyCtrlC // Ctrl+C for quit

	// Arrow keys
	KeyUp
	KeyDown
	KeyLeft
	KeyRight

	// Function keys F1–F4 (SS3 P/Q/R/S sequences)
	KeyF1
	KeyF2
	KeyF3
	KeyF4

	// Navigation keys (CSI tilde sequences and CSI/SS3 letter sequences)
	KeyHome
	KeyEnd
	KeyInsert
	KeyDelete
	KeyPageUp
	KeyPageDown

	// Extended function keys (CSI 15~/17~/18~/19~/20~/21~/23~/24~)
	KeyF5
	KeyF6
	KeyF7
	KeyF8
	KeyF9
	KeyF10
	KeyF11
	KeyF12

	// KeyShiftTab is defined here for completeness; but is not implemented yet
	KeyShiftTab
)

// ModMask represents keyboard modifiers.
type ModMask uint8

const (
	ModShift ModMask = 1 << iota
	ModAlt
	ModCtrl
)

// KeyEvent represents a key press event.
type KeyEvent struct {
	Key  Key
	Rune rune
	Mod  ModMask
}

func (KeyEvent) isEvent() {}

// ResizeEvent represents a terminal resize event.
type ResizeEvent struct {
	W int
	H int
}

func (ResizeEvent) isEvent() {}

// Size returns the size as a geom.Size.
func (e ResizeEvent) Size() geom.Size {
	return geom.Size{W: e.W, H: e.H}
}
