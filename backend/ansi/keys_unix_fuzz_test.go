//go:build unix

package ansi

import (
	"testing"
	"unicode/utf8"

	"github.com/losinggeneration/tui/event"
)

func FuzzKeyDecoder(f *testing.F) {
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
	}

	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, input []byte) {
		d := &KeyDecoder{}
		var evs []event.KeyEvent

		for _, b := range input {
			evs = d.PushByte(evs, b)
		}
		evs = d.Finalize(evs)

		// Invariant: Finalize() resets decoder to stateGround
		if d.State() != stateGround {
			t.Fatalf("decoder not reset after finalize: state=%d", d.State())
		}

		// Invariant: KeyRune values are valid runes or utf8.RuneError
		for _, ev := range evs {
			if ev.Key == event.KeyRune {
				if ev.Rune != utf8.RuneError && !utf8.ValidRune(ev.Rune) {
					t.Fatalf("invalid rune: %U", ev.Rune)
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
