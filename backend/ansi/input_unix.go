//go:build unix

package ansi

import (
	"encoding/base64"
	"strings"
	"unicode/utf8"

	"github.com/losinggeneration/tui/event"
)

// maxPasteBytes is the maximum paste buffer size (1 MB).
const maxPasteBytes = 1 << 20

// maxOSCBytes is the maximum OSC payload size (128 KiB, generous for base64 clipboard).
const maxOSCBytes = 128 * 1024

// InputDecoder decodes deterministic byte streams into Events (key, mouse, paste).
//
// INVARIANT: The decoder must never silently drop bytes.
// If a byte or buffered sequence cannot be interpreted in the current state,
// it must either:
//  1. Be emitted as one or more literal/semantic Events, or
//  2. Be replayed in stateGround.
type InputDecoder struct {
	state decodeState

	utf8Buf  [4]byte
	utf8N    int
	utf8Need int
	utf8Mod  event.ModMask

	csiBuf [32]byte // Sized for SGR mouse params like <64;200;100M (~14 bytes)
	csiN   int

	pasteBuf []byte
	oscBuf   []byte
}

type decodeState uint8

const (
	stateGround decodeState = iota
	stateEsc
	stateUTF8
	stateCSI
	stateSS3
	statePaste
	stateOSC
	stateOSCEsc // saw ESC inside OSC, waiting for '\' to confirm ST
)

// Reset resets the decoder to ground state.
func (d *InputDecoder) Reset() {
	*d = InputDecoder{}
}

// PushByte processes one byte and appends any generated events to dst.
func (d *InputDecoder) PushByte(
	dst []event.Event,
	b byte,
) []event.Event {
	switch d.state {
	case stateGround:
		return d.handleGround(dst, b)
	case stateEsc:
		return d.handleEsc(dst, b)
	case stateUTF8:
		return d.pushUTF8(dst, b)
	case stateCSI:
		return d.pushCSI(dst, b)
	case stateSS3:
		return d.handleSS3(dst, b)
	case statePaste:
		return d.pushPaste(dst, b)
	case stateOSC:
		return d.pushOSC(dst, b)
	case stateOSCEsc:
		return d.pushOSCEsc(dst, b)
	default:
		return dst
	}
}

// FlushPending emits events that are safe to resolve at an input boundary
// without assuming end-of-stream. This only flushes standalone ESC.
// Partial CSI/SS3/UTF-8/paste remain pending.
func (d *InputDecoder) FlushPending(
	dst []event.Event,
) []event.Event {
	if d.state == stateEsc {
		d.state = stateGround

		return append(dst, event.KeyEvent{Key: event.KeyEsc})
	}

	return dst
}

// Finalize flushes any incomplete state at end-of-stream.
func (d *InputDecoder) Finalize(dst []event.Event) []event.Event {
	switch d.state {
	case stateEsc:
		dst = append(dst, event.KeyEvent{Key: event.KeyEsc})

	case stateSS3:
		dst = append(dst,
			event.KeyEvent{Key: event.KeyEsc},
			event.KeyEvent{Key: event.KeyRune, Rune: 'O'},
		)

	case stateUTF8:
		dst = append(dst, event.KeyEvent{
			Key:  event.KeyRune,
			Rune: utf8.RuneError,
			Mod:  d.utf8Mod,
		})

	case stateCSI:
		dst = d.emitLiteralCSI(dst, nil)

	case statePaste:
		// Emit accumulated paste content even if end marker wasn't seen.
		if len(d.pasteBuf) > 0 {
			dst = append(dst, event.PasteEvent{Text: sanitizePasteUTF8(d.pasteBuf)})
			d.pasteBuf = nil
		}

	case stateOSC, stateOSCEsc:
		// Discard incomplete OSC at end-of-stream.
		d.oscBuf = d.oscBuf[:0]
	}

	d.Reset()

	return dst
}

func dispatchCSI(final byte) event.Key {
	switch final {
	case 'A':
		return event.KeyUp
	case 'B':
		return event.KeyDown
	case 'C':
		return event.KeyRight
	case 'D':
		return event.KeyLeft
	case 'H':
		return event.KeyHome
	case 'F':
		return event.KeyEnd
	case 'Z':
		return event.KeyShiftTab
	default:
		return event.KeyNone
	}
}

func dispatchCSITilde(p0 int) event.Key {
	switch p0 {
	case 1:
		return event.KeyHome
	case 2:
		return event.KeyInsert
	case 3:
		return event.KeyDelete
	case 4:
		return event.KeyEnd
	case 5:
		return event.KeyPageUp
	case 6:
		return event.KeyPageDown
	case 15:
		return event.KeyF5
	case 17:
		return event.KeyF6
	case 18:
		return event.KeyF7
	case 19:
		return event.KeyF8
	case 20:
		return event.KeyF9
	case 21:
		return event.KeyF10
	case 23:
		return event.KeyF11
	case 24:
		return event.KeyF12
	default:
		return event.KeyNone
	}
}

// csiModToMask converts a CSI modifier parameter to a ModMask.
// CSI modifier encoding: value = 1 + bitmask (shift=1, alt=2, ctrl=4).
// n is the number of parsed params; if n < 2, there's no modifier param.
func csiModToMask(p1, n int) event.ModMask {
	if n < 2 || p1 <= 1 {
		return 0
	}

	bits := p1 - 1

	var mod event.ModMask
	if bits&1 != 0 {
		mod |= event.ModShift
	}

	if bits&2 != 0 {
		mod |= event.ModAlt
	}

	if bits&4 != 0 {
		mod |= event.ModCtrl
	}

	return mod
}

func dispatchSS3(final byte) event.Key {
	switch final {
	case 'A':
		return event.KeyUp
	case 'B':
		return event.KeyDown
	case 'C':
		return event.KeyRight
	case 'D':
		return event.KeyLeft
	case 'H':
		return event.KeyHome
	case 'F':
		return event.KeyEnd
	case 'P':
		return event.KeyF1
	case 'Q':
		return event.KeyF2
	case 'R':
		return event.KeyF3
	case 'S':
		return event.KeyF4
	default:
		return event.KeyNone
	}
}

// interpretSingleByte interprets a single raw byte as a KeyEvent.
// This is the canonical mapping for single-byte input, used by both
// normal processing (pushGround) and literal recovery (appendLiteralByte).
// It handles control characters, printable ASCII, and invalid bytes.
func interpretSingleByte(dst []event.Event, b byte) []event.Event {
	// Handle special control characters first
	switch b {
	case 0x09: // \t
		return append(dst, event.KeyEvent{Key: event.KeyTab})
	case 0x0d, 0x0a: // \r, \n
		return append(dst, event.KeyEvent{Key: event.KeyEnter})
	case 0x7f, 0x08: // DEL, BS
		return append(dst, event.KeyEvent{Key: event.KeyBackspace})
	case 0x1b: // ESC
		return append(dst, event.KeyEvent{Key: event.KeyEsc})
	case 0x03: // Ctrl+C
		return append(dst, event.KeyEvent{Key: event.KeyCtrlC})
	}

	// Ctrl+letter: 0x01-0x1A maps to Ctrl+A through Ctrl+Z
	// (excluding 0x03/Ctrl+C handled above, 0x08/BS, 0x09/Tab, 0x0A/LF, 0x0D/CR)
	if b >= 0x01 && b <= 0x1A {
		return append(dst, event.KeyEvent{
			Key:  event.KeyRune,
			Rune: rune('a' + b - 1),
			Mod:  event.ModCtrl,
		})
	}

	// Printable ASCII range (space through ~, excluding DEL)
	if b >= 0x20 && b < 0x7f {
		return append(dst, event.KeyEvent{
			Key:  event.KeyRune,
			Rune: rune(b),
		})
	}

	// Other control bytes (0x7f+ that weren't caught above)
	return append(dst, event.KeyEvent{
		Key:  event.KeyRune,
		Rune: utf8.RuneError,
	})
}

// appendLiteralByte appends a single byte as a literal KeyEvent.
// Delegates to interpretSingleByte for the canonical mapping.
func appendLiteralByte(dst []event.Event, b byte) []event.Event {
	return interpretSingleByte(dst, b)
}

func (d *InputDecoder) emitLiteralCSI(
	dst []event.Event,
	final *byte,
) []event.Event {
	dst = append(dst, event.KeyEvent{Key: event.KeyEsc})
	dst = append(dst, event.KeyEvent{Key: event.KeyRune, Rune: '['})
	// CSI non-final bytes are in 0x20..0x3F (all printable ASCII including
	// private markers like ?, >, =), so appendLiteralByte maps them correctly.
	for i := range d.csiN {
		dst = appendLiteralByte(dst, d.csiBuf[i])
	}

	if final != nil {
		dst = appendLiteralByte(dst, *final)
	}

	d.csiN = 0
	d.state = stateGround

	return dst
}

func (d *InputDecoder) handleGround(
	dst []event.Event,
	b byte,
) []event.Event {
	if b == 0x1b {
		d.state = stateEsc

		return dst
	}

	return d.pushGround(dst, b)
}

func (d *InputDecoder) handleEsc(
	dst []event.Event,
	b byte,
) []event.Event {
	switch b {
	case '[':
		d.state = stateCSI
		d.csiN = 0

		return dst

	case ']':
		d.state = stateOSC
		d.oscBuf = d.oscBuf[:0]

		return dst

	case 'O':
		d.state = stateSS3

		return dst

	case 0x1b:
		// ESC ESC -> emit one ESC, stay in Esc state.
		return append(dst, event.KeyEvent{Key: event.KeyEsc})
	}

	// ESC + UTF-8 start -> Alt+UTF-8
	if n := utf8StartLen(b); n > 1 {
		d.state = stateUTF8
		d.utf8Mod = event.ModAlt
		d.utf8Buf[0] = b
		d.utf8N = 1
		d.utf8Need = n

		return dst
	}

	// ESC + printable ASCII -> Alt+Rune
	if b >= 0x20 && b < 0x7f {
		d.state = stateGround

		return append(dst, event.KeyEvent{
			Key:  event.KeyRune,
			Rune: rune(b),
			Mod:  event.ModAlt,
		})
	}

	// ESC + other byte -> emit ESC, replay byte in Ground.
	d.state = stateGround

	dst = append(dst, event.KeyEvent{Key: event.KeyEsc})

	return d.PushByte(dst, b)
}

func isUTF8Cont(b byte) bool {
	return b >= 0x80 && b <= 0xBF
}

// isValidUTF8NextByte validates the next byte of an in-progress UTF-8 sequence.
// first is the lead byte and have is how many bytes are already buffered,
// including the lead byte. Special constraints apply only to the second byte
// of certain lead bytes to reject overlong and surrogate encodings.
func isValidUTF8NextByte(first byte, have int, b byte) bool {
	if have <= 0 {
		return false
	}
	// Special constraints only apply to the second byte.
	if have == 1 {
		switch first {
		case 0xE0:
			return b >= 0xA0 && b <= 0xBF
		case 0xED:
			return b >= 0x80 && b <= 0x9F
		case 0xF0:
			return b >= 0x90 && b <= 0xBF
		case 0xF4:
			return b >= 0x80 && b <= 0x8F
		default:
			return isUTF8Cont(b)
		}
	}

	return isUTF8Cont(b)
}

func (d *InputDecoder) pushUTF8(
	dst []event.Event,
	b byte,
) []event.Event {
	// First byte is already validated by the caller that entered stateUTF8.
	// For continuation bytes, validate incrementally so malformed UTF-8 does not
	// consume unrelated following bytes.
	if d.utf8N > 0 && !isValidUTF8NextByte(d.utf8Buf[0], d.utf8N, b) {
		mod := d.utf8Mod

		d.state = stateGround
		d.utf8N = 0
		d.utf8Need = 0
		d.utf8Mod = 0

		dst = append(dst, event.KeyEvent{
			Key:  event.KeyRune,
			Rune: utf8.RuneError,
			Mod:  mod,
		})

		return d.PushByte(dst, b)
	}

	d.utf8Buf[d.utf8N] = b
	d.utf8N++

	if d.utf8N < d.utf8Need {
		return dst
	}

	r, _ := utf8.DecodeRune(d.utf8Buf[:d.utf8Need])

	mod := d.utf8Mod
	d.state = stateGround
	d.utf8N = 0
	d.utf8Need = 0
	d.utf8Mod = 0

	if r == utf8.RuneError {
		return append(dst, event.KeyEvent{
			Key:  event.KeyRune,
			Rune: utf8.RuneError,
			Mod:  mod,
		})
	}

	return append(dst, event.KeyEvent{
		Key:  event.KeyRune,
		Rune: r,
		Mod:  mod,
	})
}

// parseCSIParams2 parses at most two semicolon-separated numeric CSI params.
// It rejects private markers, intermediates, empty groups, and more than two
// params.
//
// Returns p0, p1: parsed params; n: count (0, 1, or 2); ok: whether parsing
// succeeded.
func parseCSIParams2(buf []byte) (p0, p1, n int, ok bool) {
	if len(buf) == 0 {
		return 0, 0, 0, true
	}

	cur := -1
	commit := func(v int) bool {
		switch n {
		case 0:
			p0 = v
		case 1:
			p1 = v
		default:
			return false
		}

		n++

		return true
	}

	for _, b := range buf {
		switch {
		case b >= '0' && b <= '9':
			if cur < 0 {
				cur = 0
			}

			cur = cur*10 + int(b-'0')
		case b == ';':
			if cur < 0 {
				return 0, 0, 0, false
			}

			if !commit(cur) {
				return 0, 0, 0, false
			}

			cur = -1
		default:
			return 0, 0, 0, false
		}
	}

	if cur < 0 {
		return 0, 0, 0, false
	}

	if !commit(cur) {
		return 0, 0, 0, false
	}

	return p0, p1, n, true
}

// parseCSIParams3 parses at most three semicolon-separated numeric CSI params.
// Used for SGR mouse sequences: Pb;Px;Py.
func parseCSIParams3(buf []byte) (p0, p1, p2, n int, ok bool) {
	if len(buf) == 0 {
		return 0, 0, 0, 0, true
	}

	cur := -1
	commit := func(v int) bool {
		switch n {
		case 0:
			p0 = v
		case 1:
			p1 = v
		case 2:
			p2 = v
		default:
			return false
		}

		n++

		return true
	}

	for _, b := range buf {
		switch {
		case b >= '0' && b <= '9':
			if cur < 0 {
				cur = 0
			}

			cur = cur*10 + int(b-'0')
		case b == ';':
			if cur < 0 {
				return 0, 0, 0, 0, false
			}

			if !commit(cur) {
				return 0, 0, 0, 0, false
			}

			cur = -1
		default:
			return 0, 0, 0, 0, false
		}
	}

	if cur < 0 {
		return 0, 0, 0, 0, false
	}

	if !commit(cur) {
		return 0, 0, 0, 0, false
	}

	return p0, p1, p2, n, true
}

// acceptsCSIKey reports whether the CSI final byte and parsed param shape are
// keyboard-shaped enough to normalize semantically rather than preserve literally.
func acceptsCSIKey(final byte, p0, _ /* mod */, n int) bool {
	switch final {
	case 'A', 'B', 'C', 'D':
		// Accept: CSI A, CSI 1 A, CSI 1;<mod> A
		switch n {
		case 0:
			return true
		case 1:
			return p0 == 1
		case 2:
			return p0 == 1
		default:
			return false
		}
	case 'H', 'F', 'Z':
		// Accept: CSI H, CSI F, CSI Z (no params)
		return n == 0
	case '~':
		// Accept known tilde params with exactly one param
		if n != 1 {
			return false
		}

		switch p0 {
		case 1, 2, 3, 4, 5, 6, 15, 17, 18, 19, 20, 21, 23, 24:
			return true
		}

		return false
	default:
		return false
	}
}

func (d *InputDecoder) pushCSI(
	dst []event.Event,
	b byte,
) []event.Event {
	// Buffer overflow guard: preserve all bytes literally.
	if d.csiN >= len(d.csiBuf) {
		return d.emitLiteralCSI(dst, &b)
	}

	// Non-final CSI bytes: 0x20..0x3F
	// This includes intermediates, parameters, and private-marker bytes.
	if b >= 0x20 && b <= 0x3F {
		d.csiBuf[d.csiN] = b
		d.csiN++

		return dst
	}

	// Final byte: 0x40..0x7E
	if b >= 0x40 && b <= 0x7E {
		// Check for SGR mouse: CSI < Pb;Px;Py M/m
		if (b == 'M' || b == 'm') && d.csiN > 0 && d.csiBuf[0] == '<' {
			if me, ok := d.parseSGRMouse(b); ok {
				d.state = stateGround
				d.csiN = 0

				return append(dst, me)
			}
		}

		// Check for bracketed paste: CSI 200~ (start) or CSI 201~ (end)
		if b == '~' {
			p0, _, n, ok := parseCSIParams2(d.csiBuf[:d.csiN])
			if ok && n == 1 && p0 == 200 {
				d.state = statePaste
				d.csiN = 0
				d.pasteBuf = d.pasteBuf[:0]

				return dst
			}
			// CSI 201~ outside paste state: ignore (shouldn't happen normally)
		}

		p0, p1, n, ok := parseCSIParams2(d.csiBuf[:d.csiN])
		if ok && acceptsCSIKey(b, p0, p1, n) {
			mod := csiModToMask(p1, n)

			if b == '~' {
				if key := dispatchCSITilde(p0); key != event.KeyNone {
					d.state = stateGround
					d.csiN = 0

					return append(dst, event.KeyEvent{Key: key, Mod: mod})
				}
			} else if key := dispatchCSI(b); key != event.KeyNone {
				d.state = stateGround
				d.csiN = 0

				return append(dst, event.KeyEvent{Key: key, Mod: mod})
			}
		}

		// Unknown or over-broad CSI final: preserve all bytes literally.
		return d.emitLiteralCSI(dst, &b)
	}

	// Invalid byte for CSI: emit buffered CSI literally, then replay offending byte.
	dst = d.emitLiteralCSI(dst, nil)

	return d.PushByte(dst, b)
}

// parseSGRMouse parses an SGR mouse sequence from the CSI buffer.
// The CSI buffer should contain '<' followed by Pb;Px;Py params.
// final is 'M' (press/move) or 'm' (release).
func (d *InputDecoder) parseSGRMouse(final byte) (event.MouseEvent, bool) {
	// Skip the leading '<'
	if d.csiN < 2 || d.csiBuf[0] != '<' {
		return event.MouseEvent{}, false
	}

	pb, px, py, n, ok := parseCSIParams3(d.csiBuf[1:d.csiN])
	if !ok || n != 3 {
		return event.MouseEvent{}, false
	}

	// Coordinates are 1-based in SGR, convert to 0-based
	x := px - 1
	y := py - 1

	if x < 0 {
		x = 0
	}

	if y < 0 {
		y = 0
	}

	// Extract modifiers from button value
	var mod event.ModMask
	if pb&4 != 0 {
		mod |= event.ModShift
	}

	if pb&8 != 0 {
		mod |= event.ModAlt
	}

	if pb&16 != 0 {
		mod |= event.ModCtrl
	}

	// Determine button and action
	buttonBits := pb & 0xC3 // bits 0-1 and 6-7

	var (
		button event.MouseButton
		action event.MouseAction
	)

	switch {
	case pb&64 != 0:
		// Wheel events
		switch buttonBits & 3 {
		case 0:
			button = event.MouseButtonWheelUp
		case 1:
			button = event.MouseButtonWheelDown
		default:
			button = event.MouseButtonNone
		}

		action = event.MousePress

	case pb&32 != 0:
		// Motion events
		action = event.MouseMove

		switch buttonBits & 3 {
		case 0:
			button = event.MouseButtonLeft
		case 1:
			button = event.MouseButtonMiddle
		case 2:
			button = event.MouseButtonRight
		default:
			button = event.MouseButtonNone
		}

	default:
		// Regular button events
		switch buttonBits & 3 {
		case 0:
			button = event.MouseButtonLeft
		case 1:
			button = event.MouseButtonMiddle
		case 2:
			button = event.MouseButtonRight
		case 3:
			button = event.MouseButtonNone // Release in X10 mode
		}

		if final == 'm' {
			action = event.MouseRelease
		} else {
			action = event.MousePress
		}
	}

	return event.MouseEvent{
		X:      x,
		Y:      y,
		Button: button,
		Action: action,
		Mod:    mod,
	}, true
}

// pushOSC accumulates bytes in OSC state.
// Terminators: BEL (0x07) or ESC \ (ST).
func (d *InputDecoder) pushOSC(dst []event.Event, b byte) []event.Event {
	switch b {
	case 0x07: // BEL — immediate terminator
		dst = d.finishOSC(dst)

		return dst
	case 0x1b: // Possible start of ESC \ (ST)
		d.state = stateOSCEsc

		return dst
	}

	d.oscBuf = append(d.oscBuf, b)
	if len(d.oscBuf) > maxOSCBytes {
		// Overflow: discard and return to ground.
		d.oscBuf = d.oscBuf[:0]
		d.state = stateGround
	}

	return dst
}

// pushOSCEsc handles the byte after ESC inside an OSC sequence.
// If it's '\', the OSC is terminated (ST = ESC \).
// Otherwise, the ESC was not a terminator — append it and the current byte.
func (d *InputDecoder) pushOSCEsc(dst []event.Event, b byte) []event.Event {
	if b == '\\' {
		// ESC \ = ST — terminate OSC
		dst = d.finishOSC(dst)

		return dst
	}
	// Not ST — the ESC was part of the payload (unusual but possible).
	d.oscBuf = append(d.oscBuf, 0x1b, b)

	d.state = stateOSC
	if len(d.oscBuf) > maxOSCBytes {
		d.oscBuf = d.oscBuf[:0]
		d.state = stateGround
	}

	return dst
}

// finishOSC parses a completed OSC payload and emits events.
func (d *InputDecoder) finishOSC(dst []event.Event) []event.Event {
	payload := string(d.oscBuf)
	d.oscBuf = d.oscBuf[:0]
	d.state = stateGround

	// OSC 52 clipboard response: "52;c;<base64-data>"
	if after, ok := strings.CutPrefix(payload, "52;"); ok {
		// Strip the selection parameter (typically "c" or "s" or "p")
		if idx := strings.IndexByte(after, ';'); idx >= 0 {
			encoded := after[idx+1:]

			decoded, err := base64.StdEncoding.DecodeString(encoded)
			if err == nil {
				dst = append(dst, event.ClipboardResponseEvent{Text: string(decoded)})
			}
			// Invalid base64: silently discard
		}
	}
	// Other OSC sequences: silently discard (we don't use them yet)

	return dst
}

// pushPaste accumulates bytes in paste state until the end marker ESC[201~ is seen.
func (d *InputDecoder) pushPaste(dst []event.Event, b byte) []event.Event {
	// Watch for ESC which could start the end marker sequence
	if b == 0x1b {
		// Check if we can peek ahead in the paste buffer for [201~
		// We can't peek ahead since we process byte-by-byte.
		// Instead, append the ESC and check for the end marker pattern
		// in the accumulated buffer.
		d.pasteBuf = append(d.pasteBuf, b)
		if len(d.pasteBuf) > maxPasteBytes {
			// Cap exceeded: emit what we have and reset
			dst = append(dst, event.PasteEvent{Text: sanitizePasteUTF8(d.pasteBuf)})
			d.pasteBuf = nil
			d.state = stateGround
		}

		return dst
	}

	d.pasteBuf = append(d.pasteBuf, b)

	// Check for end marker: ESC [ 2 0 1 ~
	// That's 6 bytes: 0x1b, '[', '2', '0', '1', '~'
	if b == '~' && len(d.pasteBuf) >= 6 {
		n := len(d.pasteBuf)
		if d.pasteBuf[n-6] == 0x1b &&
			d.pasteBuf[n-5] == '[' &&
			d.pasteBuf[n-4] == '2' &&
			d.pasteBuf[n-3] == '0' &&
			d.pasteBuf[n-2] == '1' &&
			d.pasteBuf[n-1] == '~' {
			// Found end marker. Emit paste event without the marker.
			content := d.pasteBuf[:n-6]
			dst = append(dst, event.PasteEvent{Text: sanitizePasteUTF8(content)})
			d.pasteBuf = nil
			d.state = stateGround

			return dst
		}
	}

	if len(d.pasteBuf) > maxPasteBytes {
		dst = append(dst, event.PasteEvent{Text: sanitizePasteUTF8(d.pasteBuf)})
		d.pasteBuf = nil
		d.state = stateGround
	}

	return dst
}

// sanitizePasteUTF8 replaces invalid UTF-8 sequences with U+FFFD.
func sanitizePasteUTF8(buf []byte) string {
	if utf8.Valid(buf) {
		return string(buf)
	}

	// Build valid UTF-8 string, replacing invalid bytes with replacement char
	var out []byte

	for i := 0; i < len(buf); {
		r, size := utf8.DecodeRune(buf[i:])
		if r == utf8.RuneError && size <= 1 {
			out = append(out, []byte(string(utf8.RuneError))...)
			i++
		} else {
			out = append(out, buf[i:i+size]...)
			i += size
		}
	}

	return string(out)
}

func (d *InputDecoder) handleSS3(
	dst []event.Event,
	b byte,
) []event.Event {
	if key := dispatchSS3(b); key != event.KeyNone {
		d.state = stateGround

		return append(dst, event.KeyEvent{Key: key})
	}

	// Unknown SS3 -> emit ESC, literal 'O', then replay byte.
	d.state = stateGround

	dst = append(dst, event.KeyEvent{Key: event.KeyEsc})
	dst = append(dst, event.KeyEvent{Key: event.KeyRune, Rune: 'O'})

	return d.PushByte(dst, b)
}

func (d *InputDecoder) pushGround(
	dst []event.Event,
	b byte,
) []event.Event {
	// Handle UTF-8 start bytes (state transition)
	if n := utf8StartLen(b); n > 1 {
		d.state = stateUTF8
		d.utf8Buf[0] = b
		d.utf8N = 1
		d.utf8Need = n
		d.utf8Mod = 0

		return dst
	}

	// For all other single bytes, use the canonical interpretation
	return interpretSingleByte(dst, b)
}

func utf8StartLen(b byte) int {
	// Strict UTF-8 lead-byte validation:
	//   C2..DF => 2-byte
	//   E0..EF => 3-byte
	//   F0..F4 => 4-byte
	// Reject C0/C1 (overlong) and F5..FF (out of range).
	switch {
	case b >= 0xC2 && b <= 0xDF:
		return 2
	case b >= 0xE0 && b <= 0xEF:
		return 3
	case b >= 0xF0 && b <= 0xF4:
		return 4
	default:
		return 0
	}
}
