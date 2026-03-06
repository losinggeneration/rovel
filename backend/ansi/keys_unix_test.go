//go:build unix

package ansi

import (
	"testing"
	"unicode/utf8"

	"github.com/losinggeneration/tui/event"
)

func TestKeyDecoder_ASCII(t *testing.T) {
	d := &KeyDecoder{}
	evs := d.PushByte(nil, 'a')

	if len(evs) != 1 {
		t.Fatalf("got %d events, want 1", len(evs))
	}
	if evs[0].Key != event.KeyRune || evs[0].Rune != 'a' {
		t.Fatalf("got %#v, want KeyRune('a')", evs[0])
	}
}

func TestKeyDecoder_Tab(t *testing.T) {
	d := &KeyDecoder{}
	evs := d.PushByte(nil, '\t')

	if len(evs) != 1 {
		t.Fatalf("got %d events, want 1", len(evs))
	}
	if evs[0].Key != event.KeyTab {
		t.Fatalf("got %#v, want KeyTab", evs[0])
	}
}

func TestKeyDecoder_Enter(t *testing.T) {
	d := &KeyDecoder{}

	evs := d.PushByte(nil, '\n')
	if len(evs) != 1 {
		t.Fatalf("got %d events, want 1", len(evs))
	}
	if evs[0].Key != event.KeyEnter {
		t.Fatalf("got %#v, want KeyEnter", evs[0])
	}

	d.Reset()
	evs = d.PushByte(nil, '\r')
	if len(evs) != 1 {
		t.Fatalf("got %d events, want 1", len(evs))
	}
	if evs[0].Key != event.KeyEnter {
		t.Fatalf("got %#v, want KeyEnter", evs[0])
	}
}

func TestKeyDecoder_Backspace(t *testing.T) {
	d := &KeyDecoder{}
	evs := d.PushByte(nil, 0x7f)

	if len(evs) != 1 {
		t.Fatalf("got %d events, want 1", len(evs))
	}
	if evs[0].Key != event.KeyBackspace {
		t.Fatalf("got %#v, want KeyBackspace", evs[0])
	}
}

func TestKeyDecoder_CtrlC(t *testing.T) {
	d := &KeyDecoder{}
	evs := d.PushByte(nil, 0x03)

	if len(evs) != 1 {
		t.Fatalf("got %d events, want 1", len(evs))
	}
	if evs[0].Key != event.KeyCtrlC {
		t.Fatalf("got %#v, want KeyCtrlC", evs[0])
	}
}

func TestKeyDecoder_CSIArrowUp(t *testing.T) {
	d := &KeyDecoder{}
	d.StartCSI()

	evs := d.PushByte(nil, 'A')
	if len(evs) != 1 {
		t.Fatalf("got %d events, want 1", len(evs))
	}
	if evs[0].Key != event.KeyUp {
		t.Fatalf("got %#v, want KeyUp", evs[0])
	}
}

func TestKeyDecoder_CSIArrowKeys(t *testing.T) {
	tests := []struct {
		final byte
		key   event.Key
	}{
		{'A', event.KeyUp},
		{'B', event.KeyDown},
		{'C', event.KeyRight},
		{'D', event.KeyLeft},
	}

	for _, tt := range tests {
		d := &KeyDecoder{}
		d.StartCSI()
		evs := d.PushByte(nil, tt.final)

		if len(evs) != 1 {
			t.Errorf("CSI %c: got %d events, want 1", tt.final, len(evs))
			continue
		}
		if evs[0].Key != tt.key {
			t.Errorf("CSI %c: got %#v, want %v", tt.final, evs[0], tt.key)
		}
	}
}

func TestKeyDecoder_UnsupportedSimpleCSIIsIgnored(t *testing.T) {
	d := &KeyDecoder{}
	d.StartCSI()

	evs := d.PushByte(nil, 'Z') // e.g. Shift+Tab final byte in ESC [ Z
	if len(evs) != 0 {
		t.Fatalf("got %#v, want no events", evs)
	}
}

func TestKeyDecoder_UnsupportedParameterizedCSIIsIgnored(t *testing.T) {
	d := &KeyDecoder{}
	d.StartCSI()

	var evs []event.KeyEvent
	evs = d.PushByte(evs, '1')
	evs = d.PushByte(evs, '~')

	if len(evs) != 0 {
		t.Fatalf("got %#v, want no events", evs)
	}
}

func TestKeyDecoder_FinalizeTrailingEsc(t *testing.T) {
	d := &KeyDecoder{}
	d.StartCSI()
	evs := d.Finalize(nil)

	if len(evs) != 2 {
		t.Fatalf("got %d events, want 2: %#v", len(evs), evs)
	}
	if evs[0].Key != event.KeyEsc {
		t.Fatalf("event 0 = %#v, want KeyEsc", evs[0])
	}
	if evs[1].Key != event.KeyRune || evs[1].Rune != '[' {
		t.Fatalf("event 1 = %#v, want '['", evs[1])
	}
}

func TestKeyDecoder_FinalizeTruncatedUTF8(t *testing.T) {
	d := &KeyDecoder{}
	evs := d.PushByte(nil, 0xE2) // start of 3-byte UTF-8 sequence
	if len(evs) != 0 {
		t.Fatalf("got %#v before finalize, want none", evs)
	}

	evs = d.Finalize(nil)
	if len(evs) != 1 {
		t.Fatalf("got %d events, want 1", len(evs))
	}
	if evs[0].Key != event.KeyRune || evs[0].Rune != utf8.RuneError {
		t.Fatalf("got %#v, want RuneError", evs[0])
	}
}

func TestKeyDecoder_UTF8(t *testing.T) {
	tests := []struct {
		seq  []byte
		r    rune
		name string
	}{
		{[]byte{0xC2, 0xA9}, '©', "2-byte UTF-8"},
		{[]byte{0xE2, 0x82, 0xAC}, '€', "3-byte UTF-8"},
		{[]byte{0xF0, 0x9F, 0x8C, 0x8D}, '🌍', "4-byte UTF-8"},
		{[]byte{0xE4, 0xBD, 0xA0}, '你', "Chinese character"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &KeyDecoder{}
			var evs []event.KeyEvent

			for i, b := range tt.seq {
				evs = d.PushByte(evs, b)
				if i == len(tt.seq)-1 {
					if len(evs) != 1 {
						t.Errorf("%s: got %d events on final byte, want 1", tt.name, len(evs))
					}
				} else {
					if len(evs) != 0 {
						t.Errorf("%s: got %d events on byte %d, want 0", tt.name, len(evs), i)
					}
				}
			}

			if len(evs) != 1 {
				t.Fatalf("%s: got %d events total, want 1", tt.name, len(evs))
			}
			if evs[0].Key != event.KeyRune {
				t.Errorf("%s: key = %v, want KeyRune", tt.name, evs[0].Key)
			}
			if evs[0].Rune != tt.r {
				t.Errorf("%s: rune = %c, want %c", tt.name, evs[0].Rune, tt.r)
			}
		})
	}
}

func TestKeyDecoder_Reset(t *testing.T) {
	d := &KeyDecoder{}

	// Start a CSI sequence
	d.StartCSI()

	// Reset should return to ground state
	d.Reset()

	// Should now handle normal input
	evs := d.PushByte(nil, 'a')
	if len(evs) != 1 {
		t.Errorf("After reset, got %d events, want 1", len(evs))
	}
	if evs[0].Key != event.KeyRune || evs[0].Rune != 'a' {
		t.Errorf("After reset, got unexpected event: key=%v rune=%c", evs[0].Key, evs[0].Rune)
	}
}

func TestKeyDecoder_State(t *testing.T) {
	d := &KeyDecoder{}

	// Initial state should be ground
	if d.State() != stateGround {
		t.Errorf("Initial state = %v, want stateGround", d.State())
	}

	// Start CSI should change state
	d.StartCSI()
	if d.State() != stateCSI {
		t.Errorf("After StartCSI, state = %v, want stateCSI", d.State())
	}

	// Reset should return to ground
	d.Reset()
	if d.State() != stateGround {
		t.Errorf("After reset, state = %v, want stateGround", d.State())
	}

	// Starting UTF-8 sequence should change state
	d.PushByte(nil, 0xE2) // Start of 3-byte UTF-8
	if d.State() != stateUTF8 {
		t.Errorf("After UTF-8 start, state = %v, want stateUTF8", d.State())
	}
}

func TestKeyDecoder_AbortMidCSI(t *testing.T) {
	d := &KeyDecoder{}

	// Start a CSI sequence with parameters
	d.StartCSI()
	var evs []event.KeyEvent
	evs = d.PushByte(evs, '1')
	evs = d.PushByte(evs, ';')
	evs = d.PushByte(evs, '2')

	if d.State() != stateCSI {
		t.Fatalf("State = %v, want stateCSI", d.State())
	}

	// Abort should emit ESC + [ + parameters
	evs = d.Abort(evs)

	if d.State() != stateGround {
		t.Errorf("After abort, state = %v, want stateGround", d.State())
	}

	// Should have: ESC, [, '1', ';', '2'
	if len(evs) != 5 {
		t.Fatalf("After abort, got %d events, want 5: %#v", len(evs), evs)
	}
	if evs[0].Key != event.KeyEsc {
		t.Errorf("event 0 = %#v, want KeyEsc", evs[0])
	}
	if evs[1].Key != event.KeyRune || evs[1].Rune != '[' {
		t.Errorf("event 1 = %#v, want '['", evs[1])
	}
	if evs[2].Key != event.KeyRune || evs[2].Rune != '1' {
		t.Errorf("event 2 = %#v, want '1'", evs[2])
	}
	if evs[3].Key != event.KeyRune || evs[3].Rune != ';' {
		t.Errorf("event 3 = %#v, want ';'", evs[3])
	}
	if evs[4].Key != event.KeyRune || evs[4].Rune != '2' {
		t.Errorf("event 4 = %#v, want '2'", evs[4])
	}
}

func TestKeyDecoder_AbortMidUTF8(t *testing.T) {
	d := &KeyDecoder{}

	// Start a UTF-8 sequence but don't complete it
	var evs []event.KeyEvent
	evs = d.PushByte(evs, 0xE2) // Start of 3-byte UTF-8

	if d.State() != stateUTF8 {
		t.Fatalf("State = %v, want stateUTF8", d.State())
	}

	// Abort should emit replacement rune
	evs = d.Abort(evs)

	if d.State() != stateGround {
		t.Errorf("After abort, state = %v, want stateGround", d.State())
	}

	if len(evs) != 1 {
		t.Fatalf("After abort, got %d events, want 1: %#v", len(evs), evs)
	}
	if evs[0].Key != event.KeyRune || evs[0].Rune != utf8.RuneError {
		t.Errorf("got %#v, want RuneError", evs[0])
	}
}

func TestKeyDecoder_AbortInGroundState(t *testing.T) {
	d := &KeyDecoder{}

	// Aborting in ground state should do nothing
	evs := d.Abort(nil)

	if d.State() != stateGround {
		t.Errorf("After abort, state = %v, want stateGround", d.State())
	}

	if len(evs) != 0 {
		t.Fatalf("After abort in ground state, got %d events, want 0: %#v", len(evs), evs)
	}
}
