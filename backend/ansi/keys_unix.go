//go:build unix

package ansi

import (
	"unicode/utf8"

	"github.com/losinggeneration/tui/event"
)

type decodeState uint8

const (
	stateGround decodeState = iota
	stateUTF8
	stateCSI
)

// KeyDecoder decodes deterministic byte streams into KeyEvents.
//
// Important:
// - ESC disambiguation is handled by the backend, not here.
// - This decoder only enters CSI mode after the backend has already seen ESC [.
type KeyDecoder struct {
	state    decodeState
	utf8Buf  [4]byte
	utf8N    int
	utf8Need int
	csiBuf   [8]byte
	csiN     int
}

// Reset resets the decoder to ground state.
func (d *KeyDecoder) Reset() {
	*d = KeyDecoder{}
}

// State returns the current decoder state.
func (d *KeyDecoder) State() decodeState {
	return d.state
}

// Abort finalizes the current decoder state mid-stream and resets to ground.
// Returns any recovery events for the incomplete state.
func (d *KeyDecoder) Abort(dst []event.KeyEvent) []event.KeyEvent {
	switch d.state {
	case stateUTF8:
		// Truncated UTF-8 mid-stream: emit replacement rune.
		dst = append(dst, event.KeyEvent{
			Key:  event.KeyRune,
			Rune: utf8.RuneError,
		})
	case stateCSI:
		// Incomplete CSI sequence mid-stream: preserve as literal.
		dst = append(dst,
			event.KeyEvent{Key: event.KeyEsc},
			event.KeyEvent{Key: event.KeyRune, Rune: '['},
		)
		for i := 0; i < d.csiN; i++ {
			dst = append(dst, event.KeyEvent{
				Key:  event.KeyRune,
				Rune: rune(d.csiBuf[i]),
			})
		}
	}
	d.Reset()
	return dst
}

// StartCSI begins decoding a CSI sequence after ESC [ has already been seen.
func (d *KeyDecoder) StartCSI() {
	d.state = stateCSI
	d.csiN = 0
}

// PushByte processes one byte and appends any generated events to dst.
func (d *KeyDecoder) PushByte(
	dst []event.KeyEvent,
	b byte,
) []event.KeyEvent {
	switch d.state {
	case stateGround:
		return d.pushGround(dst, b)
	case stateUTF8:
		return d.pushUTF8(dst, b)
	case stateCSI:
		return d.pushCSI(dst, b)
	default:
		return dst
	}
}

// Finalize flushes any incomplete state at end-of-stream.
func (d *KeyDecoder) Finalize(dst []event.KeyEvent) []event.KeyEvent {
	switch d.state {
	case stateUTF8:
		dst = append(dst, event.KeyEvent{
			Key:  event.KeyRune,
			Rune: utf8.RuneError,
		})

	case stateCSI:
		// Preserve incomplete ESC [ ... literally.
		dst = append(dst,
			event.KeyEvent{Key: event.KeyEsc},
			event.KeyEvent{Key: event.KeyRune, Rune: '['},
		)
		for i := 0; i < d.csiN; i++ {
			dst = append(dst, event.KeyEvent{
				Key:  event.KeyRune,
				Rune: rune(d.csiBuf[i]),
			})
		}
	}

	d.Reset()
	return dst
}

func (d *KeyDecoder) pushGround(
	dst []event.KeyEvent,
	b byte,
) []event.KeyEvent {
	switch b {
	case '\t':
		return append(dst, event.KeyEvent{Key: event.KeyTab})

	case '\r', '\n':
		return append(dst, event.KeyEvent{Key: event.KeyEnter})

	case 0x7f, 0x08:
		return append(dst, event.KeyEvent{Key: event.KeyBackspace})

	case 0x03:
		return append(dst, event.KeyEvent{Key: event.KeyCtrlC})
	}

	if b >= 0x20 && b <= 0x7e {
		return append(dst, event.KeyEvent{
			Key:  event.KeyRune,
			Rune: rune(b),
		})
	}

	switch {
	case b < 0x20:
		// Ignore other control bytes in MVP.
		return dst

	case b < 0x80:
		// DEL and other odd single-byte control-ish cases already handled above.
		return dst
	}

	// UTF-8 start byte.
	if n := utf8StartLen(b); n > 1 {
		d.state = stateUTF8
		d.utf8Buf[0] = b
		d.utf8N = 1
		d.utf8Need = n
		return dst
	}

	// Invalid UTF-8 start: emit replacement rune.
	return append(dst, event.KeyEvent{
		Key:  event.KeyRune,
		Rune: utf8.RuneError,
	})
}

func (d *KeyDecoder) pushUTF8(
	dst []event.KeyEvent,
	b byte,
) []event.KeyEvent {
	d.utf8Buf[d.utf8N] = b
	d.utf8N++

	if d.utf8N < d.utf8Need {
		return dst
	}

	r, _ := utf8.DecodeRune(d.utf8Buf[:d.utf8Need])

	d.state = stateGround
	d.utf8N = 0
	d.utf8Need = 0

	if r == utf8.RuneError {
		return append(dst, event.KeyEvent{
			Key:  event.KeyRune,
			Rune: utf8.RuneError,
		})
	}

	return append(dst, event.KeyEvent{
		Key:  event.KeyRune,
		Rune: r,
	})
}

func (d *KeyDecoder) pushCSI(
	dst []event.KeyEvent,
	b byte,
) []event.KeyEvent {
	// Parameter bytes.
	if (b >= '0' && b <= '9') || b == ';' {
		if d.csiN < len(d.csiBuf) {
			d.csiBuf[d.csiN] = b
			d.csiN++
		}
		return dst
	}

	d.state = stateGround
	defer func() { d.csiN = 0 }()

	switch b {
	case 'A':
		return append(dst, event.KeyEvent{Key: event.KeyUp})
	case 'B':
		return append(dst, event.KeyEvent{Key: event.KeyDown})
	case 'C':
		return append(dst, event.KeyEvent{Key: event.KeyRight})
	case 'D':
		return append(dst, event.KeyEvent{Key: event.KeyLeft})
	case '~':
		// Unsupported CSI ~ forms in MVP; ignore the sequence.
		return dst
	default:
		// Unsupported CSI sequence in MVP; ignore it.
		// Important: do not synthesize KeyEsc here, or keys like ESC [ Z
		// (Shift+Tab on many terminals) will look like a real Escape press.
		return dst
	}
}

func utf8StartLen(b byte) int {
	switch {
	case b&0xe0 == 0xc0:
		return 2
	case b&0xf0 == 0xe0:
		return 3
	case b&0xf8 == 0xf0:
		return 4
	default:
		return 0
	}
}
