//go:build unix

package ansi

import (
	"github.com/losinggeneration/tui/event"
)

// decodeState represents the state of the key decoder state machine.
type decodeState uint8

const (
	stateGround decodeState = iota
	stateEsc
	stateCSI
	stateUTF8
)

// KeyDecoder decodes ANSI escape sequences and UTF-8 input into KeyEvents.
type KeyDecoder struct {
	state    decodeState
	utf8Buf  [4]byte
	utf8N    int
	utf8Need int
	csiBuf   [8]byte
	csiN     int
}

// PushByte processes a single byte and returns a KeyEvent if one is complete.
// The second return value is true if a complete event was decoded.
func (d *KeyDecoder) PushByte(b byte) (event.KeyEvent, bool) {
	switch d.state {
	case stateGround:
		return d.ground(b)
	case stateEsc:
		return d.esc(b)
	case stateCSI:
		return d.csi(b)
	case stateUTF8:
		return d.utf8(b)
	default:
		d.Reset()
		return event.KeyEvent{}, false
	}
}

// ground is the default state - waiting for any input.
func (d *KeyDecoder) ground(b byte) (event.KeyEvent, bool) {
	switch b {
	case 0x1b: // ESC
		d.state = stateEsc
		d.csiN = 0
		return event.KeyEvent{}, false

	case '\t':
		return event.KeyEvent{Key: event.KeyTab}, true

	case '\n', '\r':
		return event.KeyEvent{Key: event.KeyEnter}, true

	case 0x7f: // DEL (backspace)
		return event.KeyEvent{Key: event.KeyBackspace}, true

	case 0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x0b, 0x0c, 0x0e, 0x0f,
		0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1a,
		0x1c, 0x1d, 0x1e, 0x1f:
		// Control characters - could map to Ctrl+Key combinations
		return event.KeyEvent{}, false

	default:
		// Check for UTF-8 lead byte
		if b >= 0x80 {
			d.state = stateUTF8
			d.utf8N = 0
			d.utf8Buf[0] = b

			// Determine expected UTF-8 sequence length
			if b&0xE0 == 0xC0 {
				d.utf8Need = 2
			} else if b&0xF0 == 0xE0 {
				d.utf8Need = 3
			} else if b&0xF8 == 0xF0 {
				d.utf8Need = 4
			} else {
				// Invalid UTF-8 lead byte, reset
				d.Reset()
				return event.KeyEvent{}, false
			}
			return event.KeyEvent{}, false
		}

		// Printable ASCII
		return event.KeyEvent{
			Key:  event.KeyRune,
			Rune: rune(b),
		}, true
	}
}

// esc handles input after ESC.
func (d *KeyDecoder) esc(b byte) (event.KeyEvent, bool) {
	if b == '[' {
		d.state = stateCSI
		d.csiN = 0
		return event.KeyEvent{}, false
	}

	// ESC followed by anything else could be:
	// 1. Alt+key combination (not implemented for MVP)
	// 2. Standalone ESC key

	// For MVP, treat as ESC key
	d.state = stateGround
	return event.KeyEvent{Key: event.KeyEsc}, true
}

// csi handles CSI (Control Sequence Introducer) sequences like ESC [ A
func (d *KeyDecoder) csi(b byte) (event.KeyEvent, bool) {
	// Store CSI parameter bytes
	if d.csiN < len(d.csiBuf) {
		d.csiBuf[d.csiN] = b
		d.csiN++
	}

	// Check for final byte (64-126)
	if b >= 64 && b <= 126 {
		d.state = stateGround
		return d.decodeCSI(b)
	}

	return event.KeyEvent{}, false
}

// decodeCSI decodes a complete CSI sequence.
func (d *KeyDecoder) decodeCSI(final byte) (event.KeyEvent, bool) {
	// Simple CSI sequences don't have parameters (or we ignore them for MVP)
	switch final {
	case 'A':
		return event.KeyEvent{Key: event.KeyUp}, true
	case 'B':
		return event.KeyEvent{Key: event.KeyDown}, true
	case 'C':
		return event.KeyEvent{Key: event.KeyRight}, true
	case 'D':
		return event.KeyEvent{Key: event.KeyLeft}, true
	default:
		// Unknown CSI sequence
		return event.KeyEvent{}, false
	}
}

// utf8 handles UTF-8 continuation bytes.
func (d *KeyDecoder) utf8(b byte) (event.KeyEvent, bool) {
	d.utf8N++
	d.utf8Buf[d.utf8N] = b

	if d.utf8N >= d.utf8Need-1 {
		// Complete UTF-8 sequence
		d.state = stateGround

		// Decode the rune
		var r rune
		var n int

		switch d.utf8Need {
		case 2:
			r = rune((uint32(d.utf8Buf[0])&0x1F)<<6 |
				(uint32(d.utf8Buf[1]) & 0x3F))
			n = 2
		case 3:
			r = rune((uint32(d.utf8Buf[0])&0x0F)<<12 |
				(uint32(d.utf8Buf[1])&0x3F)<<6 |
				(uint32(d.utf8Buf[2]) & 0x3F))
			n = 3
		case 4:
			r = rune((uint32(d.utf8Buf[0])&0x07)<<18 |
				(uint32(d.utf8Buf[1])&0x3F)<<12 |
				(uint32(d.utf8Buf[2])&0x3F)<<6 |
				(uint32(d.utf8Buf[3]) & 0x3F))
			n = 4
		default:
			d.Reset()
			return event.KeyEvent{}, false
		}

		// Validate the rune
		if r == utf8RuneError || n != d.utf8Need {
			d.Reset()
			return event.KeyEvent{}, false
		}

		return event.KeyEvent{
			Key:  event.KeyRune,
			Rune: r,
		}, true
	}

	return event.KeyEvent{}, false
}

// Reset resets the decoder to ground state.
func (d *KeyDecoder) Reset() {
	d.state = stateGround
	d.utf8N = 0
	d.utf8Need = 0
	d.csiN = 0
}

// utf8RuneError is the error rune returned by utf8.DecodeRune.
const utf8RuneError = '\uFFFD'
