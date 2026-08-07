//go:build unix

package ansi

import (
	"encoding/base64"
	"testing"
	"unicode/utf8"

	"github.com/losinggeneration/rovel/event"
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
	// ESC [ 1 ; 2 A -> Shift+Up (modifier param 2 = Shift)
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
		t.Fatalf("got key %#v, want KeyUp", k)
	}

	if k.Mod != event.ModShift {
		t.Fatalf("got mod %d, want ModShift (%d)", k.Mod, event.ModShift)
	}
}

func TestInputDecoder_ESCLFCombinesAsAltEnter(t *testing.T) {
	t.Run("ESC+LF", func(t *testing.T) {
		d := &InputDecoder{}

		// ESC + LF is the Alt-prefix encoding of a modified Enter, consistent
		// with ESC + rune -> Alt+rune. It must be a single modified-Enter
		// event, not KeyEsc + KeyEnter (which would quit the app via the
		// ActionCancel dispatch).
		var evs []event.Event
		evs = d.PushByte(evs, 0x1b)
		if len(evs) != 0 {
			t.Fatalf("ESC alone should buffer, got %d events: %#v", len(evs), evs)
		}

		evs = d.PushByte(evs, 0x0a)
		if len(evs) != 1 {
			t.Fatalf("ESC+LF produced %d events, want 1: %#v", len(evs), evs)
		}
		k := ke(t, evs[0], 0)
		if k.Key != event.KeyEnter {
			t.Fatalf("got key %v, want KeyEnter", k.Key)
		}
		if k.Mod != event.ModAlt {
			t.Fatalf("got mod %d, want ModAlt (%d)", k.Mod, event.ModAlt)
		}
	})

	t.Run("ESC+CR", func(t *testing.T) {
		d := &InputDecoder{}

		var evs []event.Event
		evs = d.PushByte(evs, 0x1b)
		evs = d.PushByte(evs, 0x0d)
		if len(evs) != 1 {
			t.Fatalf("ESC+CR produced %d events, want 1: %#v", len(evs), evs)
		}
		k := ke(t, evs[0], 0)
		if k.Key != event.KeyEnter || k.Mod != event.ModAlt {
			t.Fatalf("ESC+CR got (%v, mod=%d), want (KeyEnter, ModAlt)", k.Key, k.Mod)
		}
	})

	t.Run("StandaloneEscapeFlushes", func(t *testing.T) {
		d := &InputDecoder{}

		var evs []event.Event
		evs = d.PushByte(evs, 0x1b)
		if len(evs) != 0 {
			t.Fatalf("ESC alone should buffer, got %d events", len(evs))
		}
		evs = d.Finalize(evs)
		k := ke(t, evs[0], 0)
		if k.Key != event.KeyEsc {
			t.Fatalf("flushed ESC got key %v, want KeyEsc", k.Key)
		}
	})
}

func TestInputDecoder_CSIU_ModifiedEnter(t *testing.T) {
	tests := []struct {
		name string
		seq  string
		mod  event.ModMask
	}{
		{"Enter", "\x1b[13u", 0},
		{"Shift+Enter CR", "\x1b[13;2u", event.ModShift},
		{"Shift+Enter LF", "\x1b[10;2u", event.ModShift},
		{"Kitty event-type Shift+Enter CR", "\x1b[13;2:1u", event.ModShift},
		{"Kitty alternate-code Shift+Enter CR", "\x1b[13:10;2u", event.ModShift},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &InputDecoder{}
			evs := pushAll(d, []byte(tt.seq))

			if len(evs) != 1 {
				t.Fatalf("got %d events, want 1: %#v", len(evs), evs)
			}

			k := ke(t, evs[0], 0)
			if k.Key != event.KeyEnter {
				t.Fatalf("got key %v, want KeyEnter", k.Key)
			}
			if k.Mod != tt.mod {
				t.Fatalf("got mod %d, want %d", k.Mod, tt.mod)
			}
		})
	}
}

func TestInputDecoder_CSI_TildeModifiedEnter(t *testing.T) {
	tests := []struct {
		name string
		seq  string
		mod  event.ModMask
	}{
		{"modifyOtherKeys Shift+Enter CR", "\x1b[27;2;13~", event.ModShift},
		{"modifyOtherKeys Shift+Enter LF", "\x1b[27;2;10~", event.ModShift},
		{"modifyOtherKeys Alt+Enter", "\x1b[27;3;13~", event.ModAlt},
		{"tilde Shift+Enter CR", "\x1b[13;2~", event.ModShift},
		{"tilde Shift+Enter LF", "\x1b[10;2~", event.ModShift},
		{"tilde Ctrl+Enter CR", "\x1b[13;5~", event.ModCtrl},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &InputDecoder{}
			evs := pushAll(d, []byte(tt.seq))

			if len(evs) != 1 {
				t.Fatalf("got %d events, want 1: %#v", len(evs), evs)
			}

			k := ke(t, evs[0], 0)
			if k.Key != event.KeyEnter {
				t.Fatalf("got key %v, want KeyEnter", k.Key)
			}
			if k.Mod != tt.mod {
				t.Fatalf("got mod %d, want %d", k.Mod, tt.mod)
			}
		})
	}
}

func TestInputDecoder_CSIU_ControlRunes(t *testing.T) {
	tests := []struct {
		name string
		seq  string
		r    rune
	}{
		{"Ctrl+C", "\x1b[99;5u", 'c'},
		{"Ctrl+D", "\x1b[100;5u", 'd'},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &InputDecoder{}
			evs := pushAll(d, []byte(tt.seq))

			if len(evs) != 1 {
				t.Fatalf("got %d events, want 1: %#v", len(evs), evs)
			}

			k := ke(t, evs[0], 0)
			if k.Key != event.KeyRune || k.Rune != tt.r || k.Mod != event.ModCtrl {
				t.Fatalf("got %#v, want KeyRune %q ModCtrl", k, tt.r)
			}
		})
	}
}

func TestInputDecoder_CSI_ShiftArrow_Modifiers(t *testing.T) {
	tests := []struct {
		name  string
		param byte // modifier param
		mod   event.ModMask
	}{
		{"Shift+Left", '2', event.ModShift},
		{"Alt+Left", '3', event.ModAlt},
		{"Shift+Alt+Left", '4', event.ModShift | event.ModAlt},
		{"Ctrl+Left", '5', event.ModCtrl},
		{"Ctrl+Shift+Left", '6', event.ModCtrl | event.ModShift},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &InputDecoder{}

			var evs []event.Event
			// CSI 1;<mod> D
			for _, b := range []byte{0x1b, '[', '1', ';', tt.param, 'D'} {
				evs = d.PushByte(evs, b)
			}

			if len(evs) != 1 {
				t.Fatalf("got %d events, want 1", len(evs))
			}

			k := ke(t, evs[0], 0)
			if k.Key != event.KeyLeft {
				t.Fatalf("got key %v, want KeyLeft", k.Key)
			}

			if k.Mod != tt.mod {
				t.Fatalf("got mod %d, want %d", k.Mod, tt.mod)
			}
		})
	}
}

func TestInputDecoder_CtrlA(t *testing.T) {
	d := &InputDecoder{}

	var evs []event.Event

	evs = d.PushByte(evs, 0x01) // Ctrl+A
	if len(evs) != 1 {
		t.Fatalf("got %d events, want 1", len(evs))
	}

	k := ke(t, evs[0], 0)
	if k.Key != event.KeyRune || k.Rune != 'a' || k.Mod != event.ModCtrl {
		t.Fatalf("got %#v, want KeyRune 'a' ModCtrl", k)
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
	for i := range 34 {
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

func TestParseCSIParams_BoundsHugeValuesWithoutOverflow(t *testing.T) {
	// Eighteen '9' digits fit in int64 without wrapping, so unchecked
	// accumulation yields ~1e18 — a nonsense parameter far beyond any real
	// terminal value that would flow straight to a coordinate consumer.
	// Parsing must clamp it to a sane bound.
	huge := []byte("999999999999999999")

	p0, _, n, ok := parseCSIParams2(huge)
	if !ok || n != 1 {
		t.Fatalf("parseCSIParams2: ok=%v n=%d, want ok=true n=1", ok, n)
	}

	if p0 < 0 || p0 > maxCSIParam {
		t.Fatalf("parseCSIParams2 param not bounded: got %d, want 0..%d", p0, maxCSIParam)
	}

	q0, q1, q2, m, ok := parseCSIParams3(append(append(append([]byte{}, huge...), ';'), append(huge, append([]byte{';'}, huge...)...)...))
	if !ok || m != 3 {
		t.Fatalf("parseCSIParams3: ok=%v n=%d, want ok=true n=3", ok, m)
	}

	for _, p := range []int{q0, q1, q2} {
		if p < 0 || p > maxCSIParam {
			t.Fatalf("parseCSIParams3 param not bounded: got %d, want 0..%d", p, maxCSIParam)
		}
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

func TestInputDecoder_SGRMouse_ButtonMotion(t *testing.T) {
	// Button-motion events have bit 5 (32) set.
	// Left button held + motion: pb = 32 | 0 = 32
	d := &InputDecoder{}

	var evs []event.Event
	for _, b := range []byte("\x1b[<32;15;25M") {
		evs = d.PushByte(evs, b)
	}

	if len(evs) != 1 {
		t.Fatalf("got %d events, want 1", len(evs))
	}

	me := evs[0].(event.MouseEvent)
	if me.Action != event.MouseMove {
		t.Errorf("action: got %v, want MouseMove", me.Action)
	}

	if me.Button != event.MouseButtonLeft {
		t.Errorf("button: got %v, want Left", me.Button)
	}

	if me.X != 14 || me.Y != 24 {
		t.Errorf("pos: got (%d,%d), want (14,24)", me.X, me.Y)
	}
}

func TestInputDecoder_SGRMouse_RightButtonMotion(t *testing.T) {
	// Right button held + motion: pb = 32 | 2 = 34
	d := &InputDecoder{}

	var evs []event.Event
	for _, b := range []byte("\x1b[<34;5;5M") {
		evs = d.PushByte(evs, b)
	}

	if len(evs) != 1 {
		t.Fatalf("got %d events, want 1", len(evs))
	}

	me := evs[0].(event.MouseEvent)
	if me.Action != event.MouseMove {
		t.Errorf("action: got %v, want MouseMove", me.Action)
	}

	if me.Button != event.MouseButtonRight {
		t.Errorf("button: got %v, want Right", me.Button)
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

// ── OSC 52 clipboard response tests ─────────────────────────────────

func pushAll(d *InputDecoder, data []byte) []event.Event {
	var evs []event.Event
	for _, b := range data {
		evs = d.PushByte(evs, b)
	}

	return evs
}

func TestOSC52_BELTerminator(t *testing.T) {
	d := &InputDecoder{}
	// "hello" → base64 "aGVsbG8="
	// ESC ] 52;c;aGVsbG8= BEL
	evs := pushAll(d, []byte("\x1b]52;c;aGVsbG8=\x07"))

	if len(evs) != 1 {
		t.Fatalf("got %d events, want 1", len(evs))
	}

	cr, ok := evs[0].(event.ClipboardResponseEvent)
	if !ok {
		t.Fatalf("event is %T, want ClipboardResponseEvent", evs[0])
	}

	if cr.Text != "hello" {
		t.Fatalf("got text=%q, want %q", cr.Text, "hello")
	}
}

func TestOSC52_STTerminator(t *testing.T) {
	d := &InputDecoder{}
	// ESC ] 52;c;aGVsbG8= ESC \
	evs := pushAll(d, []byte("\x1b]52;c;aGVsbG8=\x1b\\"))

	if len(evs) != 1 {
		t.Fatalf("got %d events, want 1", len(evs))
	}

	cr, ok := evs[0].(event.ClipboardResponseEvent)
	if !ok {
		t.Fatalf("event is %T, want ClipboardResponseEvent", evs[0])
	}

	if cr.Text != "hello" {
		t.Fatalf("got text=%q, want %q", cr.Text, "hello")
	}
}

func TestOSC52_EmptyPayload(t *testing.T) {
	d := &InputDecoder{}
	// Empty clipboard: ESC ] 52;c; BEL
	// base64 of "" is ""
	evs := pushAll(d, []byte("\x1b]52;c;\x07"))

	if len(evs) != 1 {
		t.Fatalf("got %d events, want 1", len(evs))
	}

	cr := evs[0].(event.ClipboardResponseEvent)
	if cr.Text != "" {
		t.Fatalf("got text=%q, want empty", cr.Text)
	}
}

func TestOSC52_InvalidBase64(t *testing.T) {
	d := &InputDecoder{}
	// Invalid base64 payload should be silently discarded
	evs := pushAll(d, []byte("\x1b]52;c;!!!invalid!!!\x07"))

	if len(evs) != 0 {
		t.Fatalf("got %d events for invalid base64, want 0", len(evs))
	}
}

func TestOSC52_SanitizesControlBytesAndInvalidUTF8(t *testing.T) {
	d := &InputDecoder{}

	// Clipboard contents are attacker-controlled: embedded escape sequences,
	// C1 controls, and invalid UTF-8 must not reach the app raw. Whitespace
	// controls (\t \n \r) are legitimate clipboard content and stay.
	// U+009B is a UTF-8-encoded C1 CSI (stripped as a control); the raw
	// 0xff/0xfe bytes are invalid UTF-8 (replaced with U+FFFD).
	payload := "safe\x1b[31m\u009btext\x07 keep\tthese\nlines\r" + string([]byte{0xff, 0xfe})
	seq := "\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte(payload)) + "\x07"

	evs := pushAll(d, []byte(seq))
	if len(evs) != 1 {
		t.Fatalf("got %d events, want 1", len(evs))
	}

	cr, ok := evs[0].(event.ClipboardResponseEvent)
	if !ok {
		t.Fatalf("event is %T, want ClipboardResponseEvent", evs[0])
	}

	if !utf8.ValidString(cr.Text) {
		t.Errorf("clipboard text is not valid UTF-8: %q", cr.Text)
	}

	for _, r := range cr.Text {
		if r == '\t' || r == '\n' || r == '\r' {
			continue
		}

		if r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f) {
			t.Errorf("clipboard text contains control rune %q: %q", r, cr.Text)
		}
	}

	want := "safe[31mtext keep\tthese\nlines\r��"
	if cr.Text != want {
		t.Errorf("got text=%q, want %q", cr.Text, want)
	}
}

func TestOSC52_NonClipboardOSC(t *testing.T) {
	d := &InputDecoder{}
	// OSC 0 (set title) should be silently discarded
	evs := pushAll(d, []byte("\x1b]0;my title\x07"))

	if len(evs) != 0 {
		t.Fatalf("got %d events for non-clipboard OSC, want 0", len(evs))
	}
}

func TestOSC52_InterleavedWithInput(t *testing.T) {
	d := &InputDecoder{}
	// Type 'a', then clipboard response, then 'b'
	var evs []event.Event

	evs = d.PushByte(evs, 'a')
	for _, b := range []byte("\x1b]52;c;aGVsbG8=\x07") {
		evs = d.PushByte(evs, b)
	}

	evs = d.PushByte(evs, 'b')

	if len(evs) != 3 {
		t.Fatalf("got %d events, want 3", len(evs))
	}

	k0 := ke(t, evs[0], 0)
	if k0.Key != event.KeyRune || k0.Rune != 'a' {
		t.Fatalf("event 0: got %#v, want 'a'", k0)
	}

	cr, ok := evs[1].(event.ClipboardResponseEvent)
	if !ok {
		t.Fatalf("event 1 is %T, want ClipboardResponseEvent", evs[1])
	}

	if cr.Text != "hello" {
		t.Fatalf("got text=%q, want %q", cr.Text, "hello")
	}

	k2 := ke(t, evs[2], 2)
	if k2.Key != event.KeyRune || k2.Rune != 'b' {
		t.Fatalf("event 2: got %#v, want 'b'", k2)
	}
}

func TestOSC52_IncompleteAtFinalize(t *testing.T) {
	d := &InputDecoder{}
	// Incomplete OSC at end-of-stream should be discarded
	evs := pushAll(d, []byte("\x1b]52;c;aGVsbG8="))
	if len(evs) != 0 {
		t.Fatalf("got %d events before finalize, want 0", len(evs))
	}

	evs = d.Finalize(evs)
	if len(evs) != 0 {
		t.Fatalf("got %d events after finalize, want 0 (incomplete OSC discarded)", len(evs))
	}
}

func TestOSC52_Overflow(t *testing.T) {
	d := &InputDecoder{}
	// Push ESC ] to enter OSC state
	pushAll(d, []byte("\x1b]52;c;"))
	// Push more than maxOSCBytes
	big := make([]byte, maxOSCBytes+100)
	for i := range big {
		big[i] = 'A'
	}

	evs := pushAll(d, big)
	// Should have discarded and returned to ground
	if d.state != stateGround {
		t.Fatalf("state = %d, want stateGround after overflow", d.state)
	}
	// The 'A' bytes after overflow should have been processed as ground input
	if len(evs) == 0 {
		t.Fatal("expected some events from overflow bytes processed in ground state")
	}
}
