//go:build unix

package ansi

import (
	"testing"
	"unicode/utf8"

	"github.com/losinggeneration/tui/event"
)

func FuzzInputDecoder(f *testing.F) {
	seeds := [][]byte{
		{0x1b},
		{0x1b, 0x1b},
		{0x1b, '[', 'A'},
		{0x1b, 'O', 'A'},
		{'a'},
		{0xc3, 0xa9},
		{0x1b, 'a'},
		{0x1b, 0xc3, 0xa9},
		{0x1b, '[', '1', ';', '2', 'A'},
		{0x1b, '[', '?', '2', '5', 'h'},
		{0x1b, '[', '9', '9', '9', '9', '9', '9', '9', '9', '9'},
		{0x1b, 'O', 0xff},
		{0xc3, 0x28},
		{0x1b, 0xc3, 0x28},
		{0x1b, '[', '['},               // Double bracket
		{0x1b, 0x1b, 0x1b},             // Triple ESC
		{0x1b, '[', '[', 'A'},          // ESC [ [ A
		{0x1b, 'O', '[', 'A'},          // ESC O [ A
		{0xf0, 0x9f, 0x8c, 0x8d},       // 4-byte UTF-8 emoji
		{0x1b, 0xf0, 0x9f, 0x8c, 0x8d}, // Alt+emoji
		{0x03},                         // Ctrl+C
		{0x1b, 0x03},                   // ESC Ctrl+C

		// Clipboard response (OSC 52)
		{0x1b, ']', '5', '2', ';', 'c', ';', 'a', 'G', 'V', 's', 'b', 'G', '8', '=', 0x1b, '\\'},
		{0x1b, ']', '5', '2', ';', 'c', ';', 0x07},
		{0x1b, ']', '5', '2', ';', 'c', ';', 'Y', 'W', 'J', 'j', 0x1b, '\\'},

		// SGR mouse sequences
		{0x1b, '[', '<', '0', ';', '1', ';', '1', 'M'},                // SGR press
		{0x1b, '[', '<', '0', ';', '1', ';', '1', 'm'},                // SGR release
		{0x1b, '[', '<', '3', '5', ';', '1', '0', ';', '2', '0', 'M'}, // SGR motion
		{0x1b, '[', '<', '6', '4', ';', '5', ';', '5', 'M'},           // SGR wheel up
		{0x1b, '[', '<', '6', '5', ';', '5', ';', '5', 'M'},           // SGR wheel down

		// Paste bracket sequences
		{0x1b, '[', '2', '0', '0', '~'}, // Paste start
		{0x1b, '[', '2', '0', '0', '~', 'h', 'e', 'l', 'l', 'o', 0x1b, '[', '2', '0', '1', '~'}, // Full paste
		{0x1b, '[', '2', '0', '1', '~'}, // Paste end
	}

	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, input []byte) {
		d := &InputDecoder{}
		var evs []event.Event

		for _, b := range input {
			evs = d.PushByte(evs, b)
		}
		evs = d.Finalize(evs)

		// Invariant: Finalize() resets decoder to stateGround
		if d.state != stateGround {
			t.Fatalf("decoder not reset after finalize: state=%d", d.state)
		}

		// Invariant: KeyRune values are valid runes or utf8.RuneError
		for _, ev := range evs {
			if ke, ok := ev.(event.KeyEvent); ok {
				if ke.Key == event.KeyRune {
					if ke.Rune != utf8.RuneError && !utf8.ValidRune(ke.Rune) {
						t.Fatalf("invalid rune: %U", ke.Rune)
					}
				}
			}
		}

		// Invariant: event growth remains linearly bounded
		// Each byte can generate at most 3 events (ESC, [, buffered byte)
		// Plus 4 for end-of-stream recovery
		maxEvents := len(input)*3 + 4
		if len(evs) > maxEvents {
			t.Fatalf("event explosion: %d events from %d bytes", len(evs), len(input))
		}
	})
}
