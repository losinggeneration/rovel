// Package event provides shared event types for the tui library.
//
// This package is unstable before v0.1.0.
package event

import "github.com/losinggeneration/rovel/geom"

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

	// KeyEnter is the enter key.
	KeyEnter
	KeyEsc
	KeyTab
	KeyBackspace
	KeyCtrlC // Ctrl+C for quit

	// KeyUp is the up arrow key.
	KeyUp
	KeyDown
	KeyLeft
	KeyRight

	// KeyF1 is the F1 function key (SS3 P sequence).
	KeyF1
	KeyF2
	KeyF3
	KeyF4

	// KeyHome is the Home key (CSI tilde and CSI/SS3 letter sequences).
	KeyHome
	KeyEnd
	KeyInsert
	KeyDelete
	KeyPageUp
	KeyPageDown

	// KeyF5 is the F5 function key (CSI 15~ sequence).
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

// Size returns the size as a geom.Size.
func (e ResizeEvent) Size() geom.Size {
	return geom.Size{W: e.W, H: e.H}
}

func (ResizeEvent) isEvent() {}
