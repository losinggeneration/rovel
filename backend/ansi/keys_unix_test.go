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
		var evs []event.KeyEvent
		evs = d.PushByte(evs, 0x1b) // ESC
		evs = d.PushByte(evs, '[')  // CSI start
		evs = d.PushByte(evs, tt.final)

		if len(evs) != 1 {
			t.Errorf("CSI %c: got %d events, want 1", tt.final, len(evs))
			continue
		}
		if evs[0].Key != tt.key {
			t.Errorf("CSI %c: got %#v, want %v", tt.final, evs[0], tt.key)
		}
	}
}

func TestKeyDecoder_CSIShiftTab(t *testing.T) {
	d := &KeyDecoder{}
	var evs []event.KeyEvent
	// ESC [ Z -> Shift+Tab
	evs = d.PushByte(evs, 0x1b) // ESC
	evs = d.PushByte(evs, '[')  // CSI start
	evs = d.PushByte(evs, 'Z')  // Shift+Tab

	if len(evs) != 1 {
		t.Fatalf("got %d events, want 1: %#v", len(evs), evs)
	}
	if evs[0].Key != event.KeyShiftTab {
		t.Fatalf("got %#v, want KeyShiftTab", evs[0])
	}
}

func TestKeyDecoder_CSIShiftTab_SplitAcrossPushByte(t *testing.T) {
	d := &KeyDecoder{}
	var evs []event.KeyEvent

	evs = d.PushByte(evs, 0x1b)
	if len(evs) != 0 {
		t.Fatalf("after ESC: got %d events, want 0", len(evs))
	}

	evs = d.PushByte(evs, '[')
	if len(evs) != 0 {
		t.Fatalf("after [: got %d events, want 0", len(evs))
	}

	evs = d.PushByte(evs, 'Z')
	if len(evs) != 1 {
		t.Fatalf("after Z: got %d events, want 1", len(evs))
	}
	if evs[0].Key != event.KeyShiftTab {
		t.Fatalf("got %#v, want KeyShiftTab", evs[0])
	}
}

func TestKeyDecoder_CSIShiftTab_RejectsParams(t *testing.T) {
	tests := []struct {
		name string
		seq  []byte
	}{
		{"CSI 1 Z", []byte{0x1b, '[', '1', 'Z'}},
		{"CSI ? Z", []byte{0x1b, '[', '?', 'Z'}},
		{"CSI 0 Z", []byte{0x1b, '[', '0', 'Z'}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &KeyDecoder{}
			var evs []event.KeyEvent

			for _, b := range tt.seq {
				evs = d.PushByte(evs, b)
			}

			for i, ev := range evs {
				if ev.Key == event.KeyShiftTab {
					t.Fatalf("event %d is KeyShiftTab, should be literal bytes", i)
				}
			}
		})
	}
}

func TestKeyDecoder_CSIShiftTab_FlushPendingDoesNotEmit(t *testing.T) {
	d := &KeyDecoder{}
	var evs []event.KeyEvent

	evs = d.PushByte(evs, 0x1b)
	evs = d.PushByte(evs, '[')

	evs = d.FlushPending(evs)
	if len(evs) != 0 {
		t.Fatalf("FlushPending: got %d events, want 0", len(evs))
	}

	evs = d.PushByte(evs, 'Z')
	if len(evs) != 1 {
		t.Fatalf("after Z: got %d events, want 1", len(evs))
	}
	if evs[0].Key != event.KeyShiftTab {
		t.Fatalf("got %#v, want KeyShiftTab", evs[0])
	}
}

func TestKeyDecoder_CSI_ParamArrow_Normalized(t *testing.T) {
	d := &KeyDecoder{}
	var evs []event.KeyEvent
	// ESC [ 1 ; 2 A -> parameterized arrow should normalize to plain KeyUp
	evs = d.PushByte(evs, 0x1b)
	evs = d.PushByte(evs, '[')
	evs = d.PushByte(evs, '1')
	evs = d.PushByte(evs, ';')
	evs = d.PushByte(evs, '2')
	evs = d.PushByte(evs, 'A')

	if len(evs) != 1 {
		t.Fatalf("got %d events, want 1: %#v", len(evs), evs)
	}
	if evs[0].Key != event.KeyUp {
		t.Fatalf("got %#v, want KeyUp", evs[0])
	}
}

func TestKeyDecoder_SS3_Keys(t *testing.T) {
	tests := []struct {
		final byte
		key   event.Key
	}{
		{'A', event.KeyUp},
		{'B', event.KeyDown},
		{'C', event.KeyRight},
		{'D', event.KeyLeft},
		{'P', event.KeyF1},
		{'Q', event.KeyF2},
		{'R', event.KeyF3},
		{'S', event.KeyF4},
	}

	for _, tt := range tests {
		d := &KeyDecoder{}
		var evs []event.KeyEvent
		evs = d.PushByte(evs, 0x1b) // ESC
		evs = d.PushByte(evs, 'O')  // SS3 start
		evs = d.PushByte(evs, tt.final)

		if len(evs) != 1 {
			t.Errorf("SS3 %c: got %d events, want 1", tt.final, len(evs))
			continue
		}
		if evs[0].Key != tt.key {
			t.Errorf("SS3 %c: got %#v, want %v", tt.final, evs[0], tt.key)
		}
	}
}

func TestKeyDecoder_SS3_Unknown(t *testing.T) {
	d := &KeyDecoder{}
	var evs []event.KeyEvent
	evs = d.PushByte(evs, 0x1b) // ESC
	evs = d.PushByte(evs, 'O')  // SS3 start
	evs = d.PushByte(evs, 'x')  // Unknown final

	if len(evs) != 3 {
		t.Fatalf("got %d events, want 3: %#v", len(evs), evs)
	}
	if evs[0].Key != event.KeyEsc {
		t.Fatalf("event 0 = %#v, want KeyEsc", evs[0])
	}
	if evs[1].Key != event.KeyRune || evs[1].Rune != 'O' {
		t.Fatalf("event 1 = %#v, want 'O'", evs[1])
	}
	if evs[2].Key != event.KeyRune || evs[2].Rune != 'x' {
		t.Fatalf("event 2 = %#v, want 'x'", evs[2])
	}
}

func TestKeyDecoder_CSI_Unknown(t *testing.T) {
	// ESC [ ? 25 h -> unknown CSI, should preserve all bytes
	d := &KeyDecoder{}
	var evs []event.KeyEvent
	evs = d.PushByte(evs, 0x1b)
	evs = d.PushByte(evs, '[')
	evs = d.PushByte(evs, '?')
	evs = d.PushByte(evs, '2')
	evs = d.PushByte(evs, '5')
	evs = d.PushByte(evs, ' ')

	evs = d.PushByte(evs, 'h')

	// ESC, [, ?, 2, 5, ' ', h
	if len(evs) != 7 {
		t.Fatalf("got %d events, want 7: %#v", len(evs), evs)
	}
	if evs[0].Key != event.KeyEsc {
		t.Fatalf("event 0 = %#v, want KeyEsc", evs[0])
	}
	if evs[1].Key != event.KeyRune || evs[1].Rune != '[' {
		t.Fatalf("event 1 = %#v, want '['", evs[1])
	}
	if evs[2].Key != event.KeyRune || evs[2].Rune != '?' {
		t.Fatalf("event 2 = %#v, want '?'", evs[2])
	}
	if evs[3].Key != event.KeyRune || evs[3].Rune != '2' {
		t.Fatalf("event 3 = %#v, want '2'", evs[3])
	}
	if evs[4].Key != event.KeyRune || evs[4].Rune != '5' {
		t.Fatalf("event 4 = %#v, want '5'", evs[4])
	}
	if evs[5].Key != event.KeyRune || evs[5].Rune != ' ' {
		t.Fatalf("event 5 = %#v, want ' '", evs[5])
	}
	if evs[6].Key != event.KeyRune || evs[6].Rune != 'h' {
		t.Fatalf("event 6 = %#v, want 'h'", evs[6])
	}
}

func TestKeyDecoder_CSI_Overflow(t *testing.T) {
	d := &KeyDecoder{}
	var evs []event.KeyEvent
	evs = d.PushByte(evs, 0x1b) // ESC
	evs = d.PushByte(evs, '[')  // CSI start

	// Fill buffer to overflow (8 bytes)
	for i := 0; i < 10; i++ {
		evs = d.PushByte(evs, '0'+byte(i%10))
	}

	// Buffer size is 8, so:
	// - First 8 bytes are buffered
	// - 9th byte causes overflow: emit ESC, [, 8 buffered bytes, 9th overflow byte = 11 events
	// - 10th byte is processed in ground state as regular digit = 1 event
	// Total: 12 events
	wantCount := 12
	if len(evs) != wantCount {
		t.Fatalf("got %d events, want %d: %#v", len(evs), wantCount, evs)
	}
	if evs[0].Key != event.KeyEsc {
		t.Fatalf("event 0 = %#v, want KeyEsc", evs[0])
	}
	if evs[1].Key != event.KeyRune || evs[1].Rune != '[' {
		t.Fatalf("event 1 = %#v, want '['", evs[1])
	}
}

func TestKeyDecoder_CSI_InvalidByte(t *testing.T) {
	d := &KeyDecoder{}
	var evs []event.KeyEvent
	evs = d.PushByte(evs, 0x1b) // ESC
	evs = d.PushByte(evs, '[')  // CSI start
	evs = d.PushByte(evs, '1')  // parameter
	evs = d.PushByte(evs, 0x1b) // Invalid: ESC in middle of CSI

	// Should emit ESC, [, 1, then replay ESC (which puts us in ESC state)
	if len(evs) != 3 {
		t.Fatalf("got %d events, want 3: %#v", len(evs), evs)
	}
	if evs[0].Key != event.KeyEsc {
		t.Fatalf("event 0 = %#v, want KeyEsc", evs[0])
	}
	if evs[1].Key != event.KeyRune || evs[1].Rune != '[' {
		t.Fatalf("event 1 = %#v, want '['", evs[1])
	}
	if evs[2].Key != event.KeyRune || evs[2].Rune != '1' {
		t.Fatalf("event 2 = %#v, want '1'", evs[2])
	}
	if d.State() != stateEsc {
		t.Errorf("state = %v, want stateEsc", d.State())
	}
}

func TestKeyDecoder_Finalize_PartialCSI(t *testing.T) {
	d := &KeyDecoder{}
	var evs []event.KeyEvent
	evs = d.PushByte(evs, 0x1b) // ESC
	evs = d.PushByte(evs, '[')  // CSI start
	evs = d.PushByte(evs, '1')
	evs = d.PushByte(evs, ';')

	evs = d.Finalize(evs)

	// ESC, [, 1, ;
	if len(evs) != 4 {
		t.Fatalf("got %d events, want 4: %#v", len(evs), evs)
	}
	if evs[0].Key != event.KeyEsc {
		t.Fatalf("event 0 = %#v, want KeyEsc", evs[0])
	}
	if evs[1].Key != event.KeyRune || evs[1].Rune != '[' {
		t.Fatalf("event 1 = %#v, want '['", evs[1])
	}
	if evs[2].Key != event.KeyRune || evs[2].Rune != '1' {
		t.Fatalf("event 2 = %#v, want '1'", evs[2])
	}
	if evs[3].Key != event.KeyRune || evs[3].Rune != ';' {
		t.Fatalf("event 3 = %#v, want ';'", evs[3])
	}
}

func TestKeyDecoder_Finalize_PartialSS3(t *testing.T) {
	d := &KeyDecoder{}
	var evs []event.KeyEvent
	evs = d.PushByte(evs, 0x1b) // ESC
	evs = d.PushByte(evs, 'O')  // SS3 start

	evs = d.Finalize(evs)

	// ESC, O
	if len(evs) != 2 {
		t.Fatalf("got %d events, want 2: %#v", len(evs), evs)
	}
	if evs[0].Key != event.KeyEsc {
		t.Fatalf("event 0 = %#v, want KeyEsc", evs[0])
	}
	if evs[1].Key != event.KeyRune || evs[1].Rune != 'O' {
		t.Fatalf("event 1 = %#v, want 'O'", evs[1])
	}
}

func TestKeyDecoder_Finalize_IncompleteAltUTF8(t *testing.T) {
	d := &KeyDecoder{}
	var evs []event.KeyEvent
	evs = d.PushByte(evs, 0x1b) // ESC
	evs = d.PushByte(evs, 0xC3) // Start Alt+UTF-8 (é is C3 A9)

	evs = d.Finalize(evs)

	if len(evs) != 1 {
		t.Fatalf("got %d events, want 1: %#v", len(evs), evs)
	}
	if evs[0].Key != event.KeyRune || evs[0].Rune != utf8.RuneError {
		t.Fatalf("got %#v, want RuneError with ModAlt", evs[0])
	}
	if evs[0].Mod != event.ModAlt {
		t.Fatalf("got Mod %v, want ModAlt", evs[0].Mod)
	}
}

func TestKeyDecoder_ESC_Esc(t *testing.T) {
	d := &KeyDecoder{}
	var evs []event.KeyEvent

	// First ESC -> stateEsc
	evs = d.PushByte(evs, 0x1b)
	if len(evs) != 0 {
		t.Fatalf("after first ESC: got %d events, want 0", len(evs))
	}
	if d.State() != stateEsc {
		t.Fatalf("after first ESC: state = %v, want stateEsc", d.State())
	}

	// Second ESC -> emit one ESC, stay in stateEsc
	evs = d.PushByte(evs, 0x1b)
	if len(evs) != 1 {
		t.Fatalf("after second ESC: got %d events, want 1", len(evs))
	}
	if evs[0].Key != event.KeyEsc {
		t.Fatalf("got %#v, want KeyEsc", evs[0])
	}
	if d.State() != stateEsc {
		t.Fatalf("after second ESC: state = %v, want stateEsc", d.State())
	}

	// Finalize should emit the pending ESC (we're still in stateEsc)
	evs = d.Finalize(evs)
	// Total 2 ESCs: one from second ESC byte, one from Finalize
	if len(evs) != 2 {
		t.Fatalf("after Finalize: got %d events, want 2", len(evs))
	}
	if evs[0].Key != event.KeyEsc || evs[1].Key != event.KeyEsc {
		t.Fatalf("got %#v, want two KeyEsc", evs)
	}
}

func TestKeyDecoder_ESC_Esc_AltRune(t *testing.T) {
	d := &KeyDecoder{}
	var evs []event.KeyEvent

	// ESC ESC a
	evs = d.PushByte(evs, 0x1b)
	evs = d.PushByte(evs, 0x1b)
	evs = d.PushByte(evs, 'a')

	if len(evs) != 2 {
		t.Fatalf("got %d events, want 2: %#v", len(evs), evs)
	}
	if evs[0].Key != event.KeyEsc {
		t.Fatalf("event 0 = %#v, want KeyEsc", evs[0])
	}
	if evs[1].Key != event.KeyRune || evs[1].Rune != 'a' || evs[1].Mod != event.ModAlt {
		t.Fatalf("event 1 = %#v, want Alt+'a'", evs[1])
	}
}

func TestKeyDecoder_ESC_AltRune(t *testing.T) {
	d := &KeyDecoder{}
	var evs []event.KeyEvent
	evs = d.PushByte(evs, 0x1b)
	evs = d.PushByte(evs, 'a')

	if len(evs) != 1 {
		t.Fatalf("got %d events, want 1: %#v", len(evs), evs)
	}
	if evs[0].Key != event.KeyRune || evs[0].Rune != 'a' {
		t.Fatalf("got %#v, want KeyRune('a')", evs[0])
	}
	if evs[0].Mod != event.ModAlt {
		t.Fatalf("got Mod %v, want ModAlt", evs[0].Mod)
	}
}

func TestKeyDecoder_ESC_AltUTF8(t *testing.T) {
	d := &KeyDecoder{}
	var evs []event.KeyEvent
	// ESC C3 A9 -> Alt+é
	evs = d.PushByte(evs, 0x1b)
	evs = d.PushByte(evs, 0xC3)
	evs = d.PushByte(evs, 0xA9)

	if len(evs) != 1 {
		t.Fatalf("got %d events, want 1: %#v", len(evs), evs)
	}
	if evs[0].Key != event.KeyRune || evs[0].Rune != 'é' {
		t.Fatalf("got %#v, want KeyRune('é')", evs[0])
	}
	if evs[0].Mod != event.ModAlt {
		t.Fatalf("got Mod %v, want ModAlt", evs[0].Mod)
	}
}

func TestKeyDecoder_UTF8_InvalidContinuation(t *testing.T) {
	d := &KeyDecoder{}
	var evs []event.KeyEvent
	// C3 28 -> invalid UTF-8 (0x28 is not a valid continuation byte)
	evs = d.PushByte(evs, 0xC3)
	evs = d.PushByte(evs, 0x28) // '('

	if len(evs) != 2 {
		t.Fatalf("got %d events, want 2: %#v", len(evs), evs)
	}
	if evs[0].Key != event.KeyRune || evs[0].Rune != utf8.RuneError {
		t.Fatalf("event 0 = %#v, want RuneError", evs[0])
	}
	if evs[1].Key != event.KeyRune || evs[1].Rune != '(' {
		t.Fatalf("event 1 = %#v, want '('", evs[1])
	}
}

func TestKeyDecoder_ESC_InvalidUTF8(t *testing.T) {
	d := &KeyDecoder{}
	var evs []event.KeyEvent
	// ESC C3 28 -> invalid Alt+UTF-8
	evs = d.PushByte(evs, 0x1b)
	evs = d.PushByte(evs, 0xC3)
	evs = d.PushByte(evs, 0x28) // '('

	if len(evs) != 2 {
		t.Fatalf("got %d events, want 2: %#v", len(evs), evs)
	}
	if evs[0].Key != event.KeyRune || evs[0].Rune != utf8.RuneError {
		t.Fatalf("event 0 = %#v, want RuneError with ModAlt", evs[0])
	}
	if evs[0].Mod != event.ModAlt {
		t.Fatalf("event 0 Mod = %v, want ModAlt", evs[0].Mod)
	}
	if evs[1].Key != event.KeyRune || evs[1].Rune != '(' {
		t.Fatalf("event 1 = %#v, want '('", evs[1])
	}
	if evs[1].Mod != 0 {
		t.Fatalf("event 1 Mod = %v, want 0", evs[1].Mod)
	}
}

func TestKeyDecoder_ESC_ControlChar(t *testing.T) {
	d := &KeyDecoder{}
	var evs []event.KeyEvent
	// ESC 0x03 -> ESC followed by Ctrl+C
	evs = d.PushByte(evs, 0x1b)
	evs = d.PushByte(evs, 0x03)

	if len(evs) != 2 {
		t.Fatalf("got %d events, want 2: %#v", len(evs), evs)
	}
	if evs[0].Key != event.KeyEsc {
		t.Fatalf("event 0 = %#v, want KeyEsc", evs[0])
	}
	if evs[1].Key != event.KeyCtrlC {
		t.Fatalf("event 1 = %#v, want KeyCtrlC", evs[1])
	}
}

func TestKeyDecoder_FlushPending(t *testing.T) {
	d := &KeyDecoder{}
	var evs []event.KeyEvent

	// ESC -> goes to stateEsc, no event yet
	evs = d.PushByte(evs, 0x1b)
	if len(evs) != 0 {
		t.Fatalf("after ESC: got %d events, want 0", len(evs))
	}

	// FlushPending should emit the ESC
	evs = d.FlushPending(evs)
	if len(evs) != 1 {
		t.Fatalf("after FlushPending: got %d events, want 1", len(evs))
	}
	if evs[0].Key != event.KeyEsc {
		t.Fatalf("got %#v, want KeyEsc", evs[0])
	}

	// Decoder should be back in ground state
	if d.State() != stateGround {
		t.Fatalf("state = %v, want stateGround", d.State())
	}
}

func TestKeyDecoder_FlushPending_PartialCSI(t *testing.T) {
	d := &KeyDecoder{}
	var evs []event.KeyEvent

	evs = d.PushByte(evs, 0x1b)
	evs = d.PushByte(evs, '[')

	// FlushPending should NOT flush partial CSI
	evs = d.FlushPending(evs)
	if len(evs) != 0 {
		t.Fatalf("FlushPending flushed partial CSI: got %d events, want 0", len(evs))
	}
	if d.State() != stateCSI {
		t.Fatalf("state = %v, want stateCSI", d.State())
	}
}

func TestKeyDecoder_FlushPending_PartialSS3(t *testing.T) {
	d := &KeyDecoder{}
	var evs []event.KeyEvent

	evs = d.PushByte(evs, 0x1b)
	evs = d.PushByte(evs, 'O')

	// FlushPending should NOT flush partial SS3
	evs = d.FlushPending(evs)
	if len(evs) != 0 {
		t.Fatalf("FlushPending flushed partial SS3: got %d events, want 0", len(evs))
	}
	if d.State() != stateSS3 {
		t.Fatalf("state = %v, want stateSS3", d.State())
	}
}

func TestKeyDecoder_FlushPending_PartialUTF8(t *testing.T) {
	d := &KeyDecoder{}
	var evs []event.KeyEvent

	evs = d.PushByte(evs, 0xC3) // Start of 2-byte UTF-8

	// FlushPending should NOT flush partial UTF-8
	evs = d.FlushPending(evs)
	if len(evs) != 0 {
		t.Fatalf("FlushPending flushed partial UTF-8: got %d events, want 0", len(evs))
	}
	if d.State() != stateUTF8 {
		t.Fatalf("state = %v, want stateUTF8", d.State())
	}
}

func TestKeyDecoder_FinalizeTrailingEsc(t *testing.T) {
	d := &KeyDecoder{}
	var evs []event.KeyEvent
	evs = d.PushByte(evs, 0x1b)
	evs = d.Finalize(evs)

	if len(evs) != 1 {
		t.Fatalf("got %d events, want 1: %#v", len(evs), evs)
	}
	if evs[0].Key != event.KeyEsc {
		t.Fatalf("got %#v, want KeyEsc", evs[0])
	}
}

func TestKeyDecoder_FinalizeTruncatedUTF8(t *testing.T) {
	d := &KeyDecoder{}
	var evs []event.KeyEvent
	evs = d.PushByte(evs, 0xE2) // start of 3-byte UTF-8 sequence
	if len(evs) != 0 {
		t.Fatalf("got %#v before finalize, want none", evs)
	}

	evs = d.Finalize(evs)
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
	var evs []event.KeyEvent
	evs = d.PushByte(evs, 0x1b)
	_ = d.PushByte(evs, '[')

	// Reset should return to ground state
	d.Reset()

	// Should now handle normal input
	evs = d.PushByte(nil, 'a')
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

	// ESC should change to Esc state
	d.PushByte(nil, 0x1b)
	if d.State() != stateEsc {
		t.Errorf("After ESC, state = %v, want stateEsc", d.State())
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
