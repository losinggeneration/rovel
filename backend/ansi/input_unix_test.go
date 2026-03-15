//go:build unix

package ansi

import (
	"testing"
	"unicode/utf8"

	"github.com/losinggeneration/tui/event"
)

// ke extracts a KeyEvent from an Event, failing the test if it's not one.
func ke(t *testing.T, e event.Event, idx int) event.KeyEvent {
	t.Helper()
	k, ok := e.(event.KeyEvent)
	if !ok {
		t.Fatalf("event %d is %T, want KeyEvent", idx, e)
	}
	return k
}

func TestInputDecoder_ASCII(t *testing.T) {
	d := &InputDecoder{}
	evs := d.PushByte(nil, 'a')

	if len(evs) != 1 {
		t.Fatalf("got %d events, want 1", len(evs))
	}
	k := ke(t, evs[0], 0)
	if k.Key != event.KeyRune || k.Rune != 'a' {
		t.Fatalf("got %#v, want KeyRune('a')", k)
	}
}

func TestInputDecoder_Tab(t *testing.T) {
	d := &InputDecoder{}
	evs := d.PushByte(nil, '\t')

	if len(evs) != 1 {
		t.Fatalf("got %d events, want 1", len(evs))
	}
	k := ke(t, evs[0], 0)
	if k.Key != event.KeyTab {
		t.Fatalf("got %#v, want KeyTab", k)
	}
}

func TestInputDecoder_Enter(t *testing.T) {
	d := &InputDecoder{}

	evs := d.PushByte(nil, '\n')
	if len(evs) != 1 {
		t.Fatalf("got %d events, want 1", len(evs))
	}
	k := ke(t, evs[0], 0)
	if k.Key != event.KeyEnter {
		t.Fatalf("got %#v, want KeyEnter", k)
	}

	d.Reset()
	evs = d.PushByte(nil, '\r')
	if len(evs) != 1 {
		t.Fatalf("got %d events, want 1", len(evs))
	}
	k = ke(t, evs[0], 0)
	if k.Key != event.KeyEnter {
		t.Fatalf("got %#v, want KeyEnter", k)
	}
}

func TestInputDecoder_Backspace(t *testing.T) {
	d := &InputDecoder{}
	evs := d.PushByte(nil, 0x7f)

	if len(evs) != 1 {
		t.Fatalf("got %d events, want 1", len(evs))
	}
	k := ke(t, evs[0], 0)
	if k.Key != event.KeyBackspace {
		t.Fatalf("got %#v, want KeyBackspace", k)
	}
}

func TestInputDecoder_CtrlC(t *testing.T) {
	d := &InputDecoder{}
	evs := d.PushByte(nil, 0x03)

	if len(evs) != 1 {
		t.Fatalf("got %d events, want 1", len(evs))
	}
	k := ke(t, evs[0], 0)
	if k.Key != event.KeyCtrlC {
		t.Fatalf("got %#v, want KeyCtrlC", k)
	}
}

func TestInputDecoder_CSIArrowKeys(t *testing.T) {
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
		d := &InputDecoder{}
		var evs []event.Event
		evs = d.PushByte(evs, 0x1b) // ESC
		evs = d.PushByte(evs, '[')  // CSI start
		evs = d.PushByte(evs, tt.final)

		if len(evs) != 1 {
			t.Errorf("CSI %c: got %d events, want 1", tt.final, len(evs))
			continue
		}
		k := ke(t, evs[0], 0)
		if k.Key != tt.key {
			t.Errorf("CSI %c: got %#v, want %v", tt.final, k, tt.key)
		}
	}
}

func TestInputDecoder_CSIShiftTab(t *testing.T) {
	d := &InputDecoder{}
	var evs []event.Event
	// ESC [ Z -> Shift+Tab
	evs = d.PushByte(evs, 0x1b) // ESC
	evs = d.PushByte(evs, '[')  // CSI start
	evs = d.PushByte(evs, 'Z')  // Shift+Tab

	if len(evs) != 1 {
		t.Fatalf("got %d events, want 1: %#v", len(evs), evs)
	}
	k := ke(t, evs[0], 0)
	if k.Key != event.KeyShiftTab {
		t.Fatalf("got %#v, want KeyShiftTab", k)
	}
}

func TestInputDecoder_CSIShiftTab_SplitAcrossPushByte(t *testing.T) {
	d := &InputDecoder{}
	var evs []event.Event

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
	k := ke(t, evs[0], 0)
	if k.Key != event.KeyShiftTab {
		t.Fatalf("got %#v, want KeyShiftTab", k)
	}
}

func TestInputDecoder_CSIShiftTab_RejectsParams(t *testing.T) {
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
			d := &InputDecoder{}
			var evs []event.Event

			for _, b := range tt.seq {
				evs = d.PushByte(evs, b)
			}

			for i, ev := range evs {
				if k, ok := ev.(event.KeyEvent); ok && k.Key == event.KeyShiftTab {
					t.Fatalf("event %d is KeyShiftTab, should be literal bytes", i)
				}
			}
		})
	}
}

func TestInputDecoder_CSIShiftTab_FlushPendingDoesNotEmit(t *testing.T) {
	d := &InputDecoder{}
	var evs []event.Event

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
	k := ke(t, evs[0], 0)
	if k.Key != event.KeyShiftTab {
		t.Fatalf("got %#v, want KeyShiftTab", k)
	}
}

func TestInputDecoder_CSI_ParamArrow_Normalized(t *testing.T) {
	d := &InputDecoder{}
	var evs []event.Event
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
	k := ke(t, evs[0], 0)
	if k.Key != event.KeyUp {
		t.Fatalf("got %#v, want KeyUp", k)
	}
}

func TestInputDecoder_SS3_Keys(t *testing.T) {
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
		d := &InputDecoder{}
		var evs []event.Event
		evs = d.PushByte(evs, 0x1b) // ESC
		evs = d.PushByte(evs, 'O')  // SS3 start
		evs = d.PushByte(evs, tt.final)

		if len(evs) != 1 {
			t.Errorf("SS3 %c: got %d events, want 1", tt.final, len(evs))
			continue
		}
		k := ke(t, evs[0], 0)
		if k.Key != tt.key {
			t.Errorf("SS3 %c: got %#v, want %v", tt.final, k, tt.key)
		}
	}
}

func TestInputDecoder_SS3_Unknown(t *testing.T) {
	d := &InputDecoder{}
	var evs []event.Event
	evs = d.PushByte(evs, 0x1b) // ESC
	evs = d.PushByte(evs, 'O')  // SS3 start
	evs = d.PushByte(evs, 'x')  // Unknown final

	if len(evs) != 3 {
		t.Fatalf("got %d events, want 3: %#v", len(evs), evs)
	}
	k0 := ke(t, evs[0], 0)
	k1 := ke(t, evs[1], 1)
	k2 := ke(t, evs[2], 2)
	if k0.Key != event.KeyEsc {
		t.Fatalf("event 0 = %#v, want KeyEsc", k0)
	}
	if k1.Key != event.KeyRune || k1.Rune != 'O' {
		t.Fatalf("event 1 = %#v, want 'O'", k1)
	}
	if k2.Key != event.KeyRune || k2.Rune != 'x' {
		t.Fatalf("event 2 = %#v, want 'x'", k2)
	}
}

func TestInputDecoder_CSI_Unknown(t *testing.T) {
	// ESC [ ? 25 h -> unknown CSI, should preserve all bytes
	d := &InputDecoder{}
	var evs []event.Event
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
	k0 := ke(t, evs[0], 0)
	if k0.Key != event.KeyEsc {
		t.Fatalf("event 0 = %#v, want KeyEsc", k0)
	}
	k1 := ke(t, evs[1], 1)
	if k1.Key != event.KeyRune || k1.Rune != '[' {
		t.Fatalf("event 1 = %#v, want '['", k1)
	}
	k2 := ke(t, evs[2], 2)
	if k2.Key != event.KeyRune || k2.Rune != '?' {
		t.Fatalf("event 2 = %#v, want '?'", k2)
	}
	k3 := ke(t, evs[3], 3)
	if k3.Key != event.KeyRune || k3.Rune != '2' {
		t.Fatalf("event 3 = %#v, want '2'", k3)
	}
	k4 := ke(t, evs[4], 4)
	if k4.Key != event.KeyRune || k4.Rune != '5' {
		t.Fatalf("event 4 = %#v, want '5'", k4)
	}
	k5 := ke(t, evs[5], 5)
	if k5.Key != event.KeyRune || k5.Rune != ' ' {
		t.Fatalf("event 5 = %#v, want ' '", k5)
	}
	k6 := ke(t, evs[6], 6)
	if k6.Key != event.KeyRune || k6.Rune != 'h' {
		t.Fatalf("event 6 = %#v, want 'h'", k6)
	}
}

func TestInputDecoder_CSI_Overflow(t *testing.T) {
	d := &InputDecoder{}
	var evs []event.Event
	evs = d.PushByte(evs, 0x1b) // ESC
	evs = d.PushByte(evs, '[')  // CSI start

	// Fill buffer to overflow (32 bytes now)
	for i := 0; i < 34; i++ {
		evs = d.PushByte(evs, '0'+byte(i%10))
	}

	// Buffer size is 32, so:
	// - First 32 bytes are buffered
	// - 33rd byte causes overflow: emit ESC, [, 32 buffered bytes, 33rd overflow byte = 35 events
	// - 34th byte is processed in ground state as regular digit = 1 event
	// Total: 36 events
	wantCount := 36
	if len(evs) != wantCount {
		t.Fatalf("got %d events, want %d: %#v", len(evs), wantCount, evs)
	}
	k0 := ke(t, evs[0], 0)
	if k0.Key != event.KeyEsc {
		t.Fatalf("event 0 = %#v, want KeyEsc", k0)
	}
	k1 := ke(t, evs[1], 1)
	if k1.Key != event.KeyRune || k1.Rune != '[' {
		t.Fatalf("event 1 = %#v, want '['", k1)
	}
}

func TestInputDecoder_CSI_InvalidByte(t *testing.T) {
	d := &InputDecoder{}
	var evs []event.Event
	evs = d.PushByte(evs, 0x1b) // ESC
	evs = d.PushByte(evs, '[')  // CSI start
	evs = d.PushByte(evs, '1')  // parameter
	evs = d.PushByte(evs, 0x1b) // Invalid: ESC in middle of CSI

	// Should emit ESC, [, 1, then replay ESC (which puts us in ESC state)
	if len(evs) != 3 {
		t.Fatalf("got %d events, want 3: %#v", len(evs), evs)
	}
	k0 := ke(t, evs[0], 0)
	if k0.Key != event.KeyEsc {
		t.Fatalf("event 0 = %#v, want KeyEsc", k0)
	}
	k1 := ke(t, evs[1], 1)
	if k1.Key != event.KeyRune || k1.Rune != '[' {
		t.Fatalf("event 1 = %#v, want '['", k1)
	}
	k2 := ke(t, evs[2], 2)
	if k2.Key != event.KeyRune || k2.Rune != '1' {
		t.Fatalf("event 2 = %#v, want '1'", k2)
	}
	if d.state != stateEsc {
		t.Errorf("state = %v, want stateEsc", d.state)
	}
}

func TestInputDecoder_Finalize_PartialCSI(t *testing.T) {
	d := &InputDecoder{}
	var evs []event.Event
	evs = d.PushByte(evs, 0x1b) // ESC
	evs = d.PushByte(evs, '[')  // CSI start
	evs = d.PushByte(evs, '1')
	evs = d.PushByte(evs, ';')

	evs = d.Finalize(evs)

	// ESC, [, 1, ;
	if len(evs) != 4 {
		t.Fatalf("got %d events, want 4: %#v", len(evs), evs)
	}
	k0 := ke(t, evs[0], 0)
	if k0.Key != event.KeyEsc {
		t.Fatalf("event 0 = %#v, want KeyEsc", k0)
	}
	k1 := ke(t, evs[1], 1)
	if k1.Key != event.KeyRune || k1.Rune != '[' {
		t.Fatalf("event 1 = %#v, want '['", k1)
	}
	k2 := ke(t, evs[2], 2)
	if k2.Key != event.KeyRune || k2.Rune != '1' {
		t.Fatalf("event 2 = %#v, want '1'", k2)
	}
	k3 := ke(t, evs[3], 3)
	if k3.Key != event.KeyRune || k3.Rune != ';' {
		t.Fatalf("event 3 = %#v, want ';'", k3)
	}
}

func TestInputDecoder_Finalize_PartialSS3(t *testing.T) {
	d := &InputDecoder{}
	var evs []event.Event
	evs = d.PushByte(evs, 0x1b) // ESC
	evs = d.PushByte(evs, 'O')  // SS3 start

	evs = d.Finalize(evs)

	// ESC, O
	if len(evs) != 2 {
		t.Fatalf("got %d events, want 2: %#v", len(evs), evs)
	}
	k0 := ke(t, evs[0], 0)
	if k0.Key != event.KeyEsc {
		t.Fatalf("event 0 = %#v, want KeyEsc", k0)
	}
	k1 := ke(t, evs[1], 1)
	if k1.Key != event.KeyRune || k1.Rune != 'O' {
		t.Fatalf("event 1 = %#v, want 'O'", k1)
	}
}

func TestInputDecoder_Finalize_IncompleteAltUTF8(t *testing.T) {
	d := &InputDecoder{}
	var evs []event.Event
	evs = d.PushByte(evs, 0x1b) // ESC
	evs = d.PushByte(evs, 0xC3) // Start Alt+UTF-8 (é is C3 A9)

	evs = d.Finalize(evs)

	if len(evs) != 1 {
		t.Fatalf("got %d events, want 1: %#v", len(evs), evs)
	}
	k := ke(t, evs[0], 0)
	if k.Key != event.KeyRune || k.Rune != utf8.RuneError {
		t.Fatalf("got %#v, want RuneError with ModAlt", k)
	}
	if k.Mod != event.ModAlt {
		t.Fatalf("got Mod %v, want ModAlt", k.Mod)
	}
}

func TestInputDecoder_ESC_Esc(t *testing.T) {
	d := &InputDecoder{}
	var evs []event.Event

	// First ESC -> stateEsc
	evs = d.PushByte(evs, 0x1b)
	if len(evs) != 0 {
		t.Fatalf("after first ESC: got %d events, want 0", len(evs))
	}
	if d.state != stateEsc {
		t.Fatalf("after first ESC: state = %v, want stateEsc", d.state)
	}

	// Second ESC -> emit one ESC, stay in stateEsc
	evs = d.PushByte(evs, 0x1b)
	if len(evs) != 1 {
		t.Fatalf("after second ESC: got %d events, want 1", len(evs))
	}
	k := ke(t, evs[0], 0)
	if k.Key != event.KeyEsc {
		t.Fatalf("got %#v, want KeyEsc", k)
	}
	if d.state != stateEsc {
		t.Fatalf("after second ESC: state = %v, want stateEsc", d.state)
	}

	// Finalize should emit the pending ESC (we're still in stateEsc)
	evs = d.Finalize(evs)
	// Total 2 ESCs: one from second ESC byte, one from Finalize
	if len(evs) != 2 {
		t.Fatalf("after Finalize: got %d events, want 2", len(evs))
	}
	k0 := ke(t, evs[0], 0)
	k1 := ke(t, evs[1], 1)
	if k0.Key != event.KeyEsc || k1.Key != event.KeyEsc {
		t.Fatalf("got %#v, want two KeyEsc", evs)
	}
}

func TestInputDecoder_ESC_Esc_AltRune(t *testing.T) {
	d := &InputDecoder{}
	var evs []event.Event

	// ESC ESC a
	evs = d.PushByte(evs, 0x1b)
	evs = d.PushByte(evs, 0x1b)
	evs = d.PushByte(evs, 'a')

	if len(evs) != 2 {
		t.Fatalf("got %d events, want 2: %#v", len(evs), evs)
	}
	k0 := ke(t, evs[0], 0)
	if k0.Key != event.KeyEsc {
		t.Fatalf("event 0 = %#v, want KeyEsc", k0)
	}
	k1 := ke(t, evs[1], 1)
	if k1.Key != event.KeyRune || k1.Rune != 'a' || k1.Mod != event.ModAlt {
		t.Fatalf("event 1 = %#v, want Alt+'a'", k1)
	}
}

func TestInputDecoder_ESC_AltRune(t *testing.T) {
	d := &InputDecoder{}
	var evs []event.Event
	evs = d.PushByte(evs, 0x1b)
	evs = d.PushByte(evs, 'a')

	if len(evs) != 1 {
		t.Fatalf("got %d events, want 1: %#v", len(evs), evs)
	}
	k := ke(t, evs[0], 0)
	if k.Key != event.KeyRune || k.Rune != 'a' {
		t.Fatalf("got %#v, want KeyRune('a')", k)
	}
	if k.Mod != event.ModAlt {
		t.Fatalf("got Mod %v, want ModAlt", k.Mod)
	}
}

func TestInputDecoder_ESC_AltUTF8(t *testing.T) {
	d := &InputDecoder{}
	var evs []event.Event
	// ESC C3 A9 -> Alt+é
	evs = d.PushByte(evs, 0x1b)
	evs = d.PushByte(evs, 0xC3)
	evs = d.PushByte(evs, 0xA9)

	if len(evs) != 1 {
		t.Fatalf("got %d events, want 1: %#v", len(evs), evs)
	}
	k := ke(t, evs[0], 0)
	if k.Key != event.KeyRune || k.Rune != 'é' {
		t.Fatalf("got %#v, want KeyRune('é')", k)
	}
	if k.Mod != event.ModAlt {
		t.Fatalf("got Mod %v, want ModAlt", k.Mod)
	}
}

func TestInputDecoder_UTF8_InvalidContinuation(t *testing.T) {
	d := &InputDecoder{}
	var evs []event.Event
	// C3 28 -> invalid UTF-8 (0x28 is not a valid continuation byte)
	evs = d.PushByte(evs, 0xC3)
	evs = d.PushByte(evs, 0x28) // '('

	if len(evs) != 2 {
		t.Fatalf("got %d events, want 2: %#v", len(evs), evs)
	}
	k0 := ke(t, evs[0], 0)
	if k0.Key != event.KeyRune || k0.Rune != utf8.RuneError {
		t.Fatalf("event 0 = %#v, want RuneError", k0)
	}
	k1 := ke(t, evs[1], 1)
	if k1.Key != event.KeyRune || k1.Rune != '(' {
		t.Fatalf("event 1 = %#v, want '('", k1)
	}
}

func TestInputDecoder_ESC_InvalidUTF8(t *testing.T) {
	d := &InputDecoder{}
	var evs []event.Event
	// ESC C3 28 -> invalid Alt+UTF-8
	evs = d.PushByte(evs, 0x1b)
	evs = d.PushByte(evs, 0xC3)
	evs = d.PushByte(evs, 0x28) // '('

	if len(evs) != 2 {
		t.Fatalf("got %d events, want 2: %#v", len(evs), evs)
	}
	k0 := ke(t, evs[0], 0)
	if k0.Key != event.KeyRune || k0.Rune != utf8.RuneError {
		t.Fatalf("event 0 = %#v, want RuneError with ModAlt", k0)
	}
	if k0.Mod != event.ModAlt {
		t.Fatalf("event 0 Mod = %v, want ModAlt", k0.Mod)
	}
	k1 := ke(t, evs[1], 1)
	if k1.Key != event.KeyRune || k1.Rune != '(' {
		t.Fatalf("event 1 = %#v, want '('", k1)
	}
	if k1.Mod != 0 {
		t.Fatalf("event 1 Mod = %v, want 0", k1.Mod)
	}
}

func TestInputDecoder_ESC_ControlChar(t *testing.T) {
	d := &InputDecoder{}
	var evs []event.Event
	// ESC 0x03 -> ESC followed by Ctrl+C
	evs = d.PushByte(evs, 0x1b)
	evs = d.PushByte(evs, 0x03)

	if len(evs) != 2 {
		t.Fatalf("got %d events, want 2: %#v", len(evs), evs)
	}
	k0 := ke(t, evs[0], 0)
	if k0.Key != event.KeyEsc {
		t.Fatalf("event 0 = %#v, want KeyEsc", k0)
	}
	k1 := ke(t, evs[1], 1)
	if k1.Key != event.KeyCtrlC {
		t.Fatalf("event 1 = %#v, want KeyCtrlC", k1)
	}
}

func TestInputDecoder_FlushPending(t *testing.T) {
	d := &InputDecoder{}
	var evs []event.Event

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
	k := ke(t, evs[0], 0)
	if k.Key != event.KeyEsc {
		t.Fatalf("got %#v, want KeyEsc", k)
	}

	// Decoder should be back in ground state
	if d.state != stateGround {
		t.Fatalf("state = %v, want stateGround", d.state)
	}
}

func TestInputDecoder_FlushPending_PartialCSI(t *testing.T) {
	d := &InputDecoder{}
	var evs []event.Event

	evs = d.PushByte(evs, 0x1b)
	evs = d.PushByte(evs, '[')

	// FlushPending should NOT flush partial CSI
	evs = d.FlushPending(evs)
	if len(evs) != 0 {
		t.Fatalf("FlushPending flushed partial CSI: got %d events, want 0", len(evs))
	}
	if d.state != stateCSI {
		t.Fatalf("state = %v, want stateCSI", d.state)
	}
}

func TestInputDecoder_FlushPending_PartialSS3(t *testing.T) {
	d := &InputDecoder{}
	var evs []event.Event

	evs = d.PushByte(evs, 0x1b)
	evs = d.PushByte(evs, 'O')

	// FlushPending should NOT flush partial SS3
	evs = d.FlushPending(evs)
	if len(evs) != 0 {
		t.Fatalf("FlushPending flushed partial SS3: got %d events, want 0", len(evs))
	}
	if d.state != stateSS3 {
		t.Fatalf("state = %v, want stateSS3", d.state)
	}
}

func TestInputDecoder_FlushPending_PartialUTF8(t *testing.T) {
	d := &InputDecoder{}
	var evs []event.Event

	evs = d.PushByte(evs, 0xC3) // Start of 2-byte UTF-8

	// FlushPending should NOT flush partial UTF-8
	evs = d.FlushPending(evs)
	if len(evs) != 0 {
		t.Fatalf("FlushPending flushed partial UTF-8: got %d events, want 0", len(evs))
	}
	if d.state != stateUTF8 {
		t.Fatalf("state = %v, want stateUTF8", d.state)
	}
}

func TestInputDecoder_FinalizeTrailingEsc(t *testing.T) {
	d := &InputDecoder{}
	var evs []event.Event
	evs = d.PushByte(evs, 0x1b)
	evs = d.Finalize(evs)

	if len(evs) != 1 {
		t.Fatalf("got %d events, want 1: %#v", len(evs), evs)
	}
	k := ke(t, evs[0], 0)
	if k.Key != event.KeyEsc {
		t.Fatalf("got %#v, want KeyEsc", k)
	}
}

func TestInputDecoder_FinalizeTruncatedUTF8(t *testing.T) {
	d := &InputDecoder{}
	var evs []event.Event
	evs = d.PushByte(evs, 0xE2) // start of 3-byte UTF-8 sequence
	if len(evs) != 0 {
		t.Fatalf("got %#v before finalize, want none", evs)
	}

	evs = d.Finalize(evs)
	if len(evs) != 1 {
		t.Fatalf("got %d events, want 1", len(evs))
	}
	k := ke(t, evs[0], 0)
	if k.Key != event.KeyRune || k.Rune != utf8.RuneError {
		t.Fatalf("got %#v, want RuneError", k)
	}
}

func TestInputDecoder_UTF8(t *testing.T) {
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
			d := &InputDecoder{}
			var evs []event.Event

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
			k := ke(t, evs[0], 0)
			if k.Key != event.KeyRune {
				t.Errorf("%s: key = %v, want KeyRune", tt.name, k.Key)
			}
			if k.Rune != tt.r {
				t.Errorf("%s: rune = %c, want %c", tt.name, k.Rune, tt.r)
			}
		})
	}
}

func TestInputDecoder_Reset(t *testing.T) {
	d := &InputDecoder{}

	// Start a CSI sequence
	var evs []event.Event
	evs = d.PushByte(evs, 0x1b)
	_ = d.PushByte(evs, '[')

	// Reset should return to ground state
	d.Reset()

	// Should now handle normal input
	evs = d.PushByte(nil, 'a')
	if len(evs) != 1 {
		t.Errorf("After reset, got %d events, want 1", len(evs))
	}
	k := ke(t, evs[0], 0)
	if k.Key != event.KeyRune || k.Rune != 'a' {
		t.Errorf("After reset, got unexpected event: key=%v rune=%c", k.Key, k.Rune)
	}
}

func TestInputDecoder_State(t *testing.T) {
	d := &InputDecoder{}

	// Initial state should be ground
	if d.state != stateGround {
		t.Errorf("Initial state = %v, want stateGround", d.state)
	}

	// ESC should change to Esc state
	d.PushByte(nil, 0x1b)
	if d.state != stateEsc {
		t.Errorf("After ESC, state = %v, want stateEsc", d.state)
	}

	// Reset should return to ground
	d.Reset()
	if d.state != stateGround {
		t.Errorf("After reset, state = %v, want stateGround", d.state)
	}

	// Starting UTF-8 sequence should change state
	d.PushByte(nil, 0xE2) // Start of 3-byte UTF-8
	if d.state != stateUTF8 {
		t.Errorf("After UTF-8 start, state = %v, want stateUTF8", d.state)
	}
}

// --- SGR Mouse Tests ---

func TestInputDecoder_SGRMouse_LeftPress(t *testing.T) {
	d := &InputDecoder{}
	var evs []event.Event
	// ESC [ < 0 ; 10 ; 20 M -> left press at (9, 19)
	for _, b := range []byte("\x1b[<0;10;20M") {
		evs = d.PushByte(evs, b)
	}
	if len(evs) != 1 {
		t.Fatalf("got %d events, want 1: %#v", len(evs), evs)
	}
	me, ok := evs[0].(event.MouseEvent)
	if !ok {
		t.Fatalf("got %T, want MouseEvent", evs[0])
	}
	if me.Button != event.MouseButtonLeft || me.Action != event.MousePress {
		t.Fatalf("got button=%v action=%v, want Left/Press", me.Button, me.Action)
	}
	if me.X != 9 || me.Y != 19 {
		t.Fatalf("got (%d,%d), want (9,19)", me.X, me.Y)
	}
}

func TestInputDecoder_SGRMouse_LeftRelease(t *testing.T) {
	d := &InputDecoder{}
	var evs []event.Event
	for _, b := range []byte("\x1b[<0;10;20m") {
		evs = d.PushByte(evs, b)
	}
	if len(evs) != 1 {
		t.Fatalf("got %d events, want 1", len(evs))
	}
	me := evs[0].(event.MouseEvent)
	if me.Button != event.MouseButtonLeft || me.Action != event.MouseRelease {
		t.Fatalf("got button=%v action=%v, want Left/Release", me.Button, me.Action)
	}
}

func TestInputDecoder_SGRMouse_MiddleRight(t *testing.T) {
	tests := []struct {
		seq    string
		button event.MouseButton
	}{
		{"\x1b[<1;5;5M", event.MouseButtonMiddle},
		{"\x1b[<2;5;5M", event.MouseButtonRight},
	}
	for _, tt := range tests {
		d := &InputDecoder{}
		var evs []event.Event
		for _, b := range []byte(tt.seq) {
			evs = d.PushByte(evs, b)
		}
		if len(evs) != 1 {
			t.Fatalf("%s: got %d events", tt.seq, len(evs))
		}
		me := evs[0].(event.MouseEvent)
		if me.Button != tt.button {
			t.Fatalf("%s: got button=%v, want %v", tt.seq, me.Button, tt.button)
		}
	}
}

func TestInputDecoder_SGRMouse_WheelUp(t *testing.T) {
	d := &InputDecoder{}
	var evs []event.Event
	for _, b := range []byte("\x1b[<64;10;20M") {
		evs = d.PushByte(evs, b)
	}
	if len(evs) != 1 {
		t.Fatalf("got %d events, want 1", len(evs))
	}
	me := evs[0].(event.MouseEvent)
	if me.Button != event.MouseButtonWheelUp {
		t.Fatalf("got button=%v, want WheelUp", me.Button)
	}
}

func TestInputDecoder_SGRMouse_WheelDown(t *testing.T) {
	d := &InputDecoder{}
	var evs []event.Event
	for _, b := range []byte("\x1b[<65;10;20M") {
		evs = d.PushByte(evs, b)
	}
	if len(evs) != 1 {
		t.Fatalf("got %d events, want 1", len(evs))
	}
	me := evs[0].(event.MouseEvent)
	if me.Button != event.MouseButtonWheelDown {
		t.Fatalf("got button=%v, want WheelDown", me.Button)
	}
}

func TestInputDecoder_SGRMouse_Modifiers(t *testing.T) {
	d := &InputDecoder{}
	var evs []event.Event
	// Shift=4, Alt=8, Ctrl=16 -> 4+8+16=28 -> button bits: 28
	for _, b := range []byte("\x1b[<28;5;5M") {
		evs = d.PushByte(evs, b)
	}
	if len(evs) != 1 {
		t.Fatalf("got %d events, want 1", len(evs))
	}
	me := evs[0].(event.MouseEvent)
	if me.Mod&event.ModShift == 0 {
		t.Error("missing ModShift")
	}
	if me.Mod&event.ModAlt == 0 {
		t.Error("missing ModAlt")
	}
	if me.Mod&event.ModCtrl == 0 {
		t.Error("missing ModCtrl")
	}
}

func TestInputDecoder_SGRMouse_LargeCoords(t *testing.T) {
	d := &InputDecoder{}
	var evs []event.Event
	for _, b := range []byte("\x1b[<0;300;200M") {
		evs = d.PushByte(evs, b)
	}
	if len(evs) != 1 {
		t.Fatalf("got %d events, want 1", len(evs))
	}
	me := evs[0].(event.MouseEvent)
	if me.X != 299 || me.Y != 199 {
		t.Fatalf("got (%d,%d), want (299,199)", me.X, me.Y)
	}
}

// --- Paste Tests ---

func TestInputDecoder_Paste_ASCII(t *testing.T) {
	d := &InputDecoder{}
	var evs []event.Event
	// ESC[200~ hello ESC[201~
	for _, b := range []byte("\x1b[200~hello\x1b[201~") {
		evs = d.PushByte(evs, b)
	}
	if len(evs) != 1 {
		t.Fatalf("got %d events, want 1: %#v", len(evs), evs)
	}
	pe, ok := evs[0].(event.PasteEvent)
	if !ok {
		t.Fatalf("got %T, want PasteEvent", evs[0])
	}
	if pe.Text != "hello" {
		t.Fatalf("got text=%q, want %q", pe.Text, "hello")
	}
}

func TestInputDecoder_Paste_Empty(t *testing.T) {
	d := &InputDecoder{}
	var evs []event.Event
	for _, b := range []byte("\x1b[200~\x1b[201~") {
		evs = d.PushByte(evs, b)
	}
	if len(evs) != 1 {
		t.Fatalf("got %d events, want 1", len(evs))
	}
	pe := evs[0].(event.PasteEvent)
	if pe.Text != "" {
		t.Fatalf("got text=%q, want empty", pe.Text)
	}
}

func TestInputDecoder_Paste_UTF8(t *testing.T) {
	d := &InputDecoder{}
	var evs []event.Event
	for _, b := range []byte("\x1b[200~café\x1b[201~") {
		evs = d.PushByte(evs, b)
	}
	if len(evs) != 1 {
		t.Fatalf("got %d events, want 1", len(evs))
	}
	pe := evs[0].(event.PasteEvent)
	if pe.Text != "café" {
		t.Fatalf("got text=%q, want %q", pe.Text, "café")
	}
}

func TestInputDecoder_Paste_InvalidUTF8(t *testing.T) {
	d := &InputDecoder{}
	var evs []event.Event
	// Insert invalid UTF-8 byte in paste content
	paste := append([]byte("\x1b[200~ab"), 0xFF)
	paste = append(paste, []byte("cd\x1b[201~")...)
	for _, b := range paste {
		evs = d.PushByte(evs, b)
	}
	if len(evs) != 1 {
		t.Fatalf("got %d events, want 1", len(evs))
	}
	pe := evs[0].(event.PasteEvent)
	// Invalid byte should be replaced with U+FFFD
	if pe.Text != "ab\uFFFDcd" {
		t.Fatalf("got text=%q, want %q", pe.Text, "ab\uFFFDcd")
	}
}

func TestInputDecoder_Paste_EmbeddedESC(t *testing.T) {
	d := &InputDecoder{}
	var evs []event.Event
	// ESC inside paste that doesn't form end marker
	for _, b := range []byte("\x1b[200~a\x1bb\x1b[201~") {
		evs = d.PushByte(evs, b)
	}
	if len(evs) != 1 {
		t.Fatalf("got %d events, want 1", len(evs))
	}
	pe := evs[0].(event.PasteEvent)
	if pe.Text != "a\x1bb" {
		t.Fatalf("got text=%q, want %q", pe.Text, "a\x1bb")
	}
}

func TestInputDecoder_Paste_Finalize(t *testing.T) {
	d := &InputDecoder{}
	var evs []event.Event
	// Start paste but don't finish it
	for _, b := range []byte("\x1b[200~partial") {
		evs = d.PushByte(evs, b)
	}
	if len(evs) != 0 {
		t.Fatalf("got %d events before finalize, want 0", len(evs))
	}
	evs = d.Finalize(evs)
	if len(evs) != 1 {
		t.Fatalf("got %d events after finalize, want 1", len(evs))
	}
	pe := evs[0].(event.PasteEvent)
	if pe.Text != "partial" {
		t.Fatalf("got text=%q, want %q", pe.Text, "partial")
	}
}
