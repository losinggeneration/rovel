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

	// Arrow keys
	KeyUp
	KeyDown
	KeyLeft
	KeyRight
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
