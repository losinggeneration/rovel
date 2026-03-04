//go:build unix

package ansi

import (
	"testing"

	"github.com/losinggeneration/tui/event"
)

func TestKeyDecoder_PrintableASCII(t *testing.T) {
	d := &KeyDecoder{}

	tests := []struct {
		b   byte
		r   rune
		key event.Key
	}{
		{'a', 'a', event.KeyRune},
		{'Z', 'Z', event.KeyRune},
		{' ', ' ', event.KeyRune},
		{'@', '@', event.KeyRune},
	}

	for _, tt := range tests {
		evt, ok := d.PushByte(tt.b)
		if !ok {
			t.Errorf("PushByte(%c) returned false, want true", tt.b)
			continue
		}
		if evt.Key != tt.key {
			t.Errorf("PushByte(%c) key = %v, want %v", tt.b, evt.Key, tt.key)
		}
		if evt.Rune != tt.r {
			t.Errorf("PushByte(%c) rune = %c, want %c", tt.b, evt.Rune, tt.r)
		}
	}
}

func TestKeyDecoder_SpecialKeys(t *testing.T) {
	d := &KeyDecoder{}

	tests := []struct {
		b   byte
		key event.Key
	}{
		{'\t', event.KeyTab},
		{'\n', event.KeyEnter},
		{'\r', event.KeyEnter},
		{0x7f, event.KeyBackspace},
	}

	for _, tt := range tests {
		d.Reset()
		evt, ok := d.PushByte(tt.b)
		if !ok {
			t.Errorf("PushByte(0x%02x) returned false, want true", tt.b)
			continue
		}
		if evt.Key != tt.key {
			t.Errorf("PushByte(0x%02x) key = %v, want %v", tt.b, evt.Key, tt.key)
		}
	}
}

func TestKeyDecoder_ArrowKeys(t *testing.T) {
	tests := []struct {
		seq []byte
		key event.Key
	}{
		{[]byte{0x1b, '[', 'A'}, event.KeyUp},
		{[]byte{0x1b, '[', 'B'}, event.KeyDown},
		{[]byte{0x1b, '[', 'C'}, event.KeyRight},
		{[]byte{0x1b, '[', 'D'}, event.KeyLeft},
	}

	for _, tt := range tests {
		d := &KeyDecoder{}
		var evt event.KeyEvent
		var ok bool

		for i, b := range tt.seq {
			evt, ok = d.PushByte(b)
			if i == len(tt.seq)-1 {
				if !ok {
					t.Errorf("Arrow key sequence %v: last byte returned false", tt.seq)
				}
				if evt.Key != tt.key {
					t.Errorf("Arrow key sequence %v: key = %v, want %v", tt.seq, evt.Key, tt.key)
				}
			} else {
				if ok {
					t.Errorf("Arrow key sequence %v: byte %d returned event early", tt.seq, i)
				}
			}
		}
	}
}

func TestKeyDecoder_Esc(t *testing.T) {
	d := &KeyDecoder{}

	// ESC followed by non-[ character should emit ESC
	evt, ok := d.PushByte(0x1b)
	if ok {
		t.Errorf("ESC alone returned event, want false")
	}

	evt, ok = d.PushByte('x')
	if !ok {
		t.Errorf("ESC + 'x' returned false, want true")
	}
	if evt.Key != event.KeyEsc {
		t.Errorf("ESC + 'x' key = %v, want KeyEsc", evt.Key)
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
		d := &KeyDecoder{}
		var evt event.KeyEvent
		var ok bool

		for i, b := range tt.seq {
			evt, ok = d.PushByte(b)
			if i == len(tt.seq)-1 {
				if !ok {
					t.Errorf("%s: last byte returned false", tt.name)
				}
				if evt.Key != event.KeyRune {
					t.Errorf("%s: key = %v, want KeyRune", tt.name, evt.Key)
				}
				if evt.Rune != tt.r {
					t.Errorf("%s: rune = %c, want %c", tt.name, evt.Rune, tt.r)
				}
			} else {
				if ok {
					t.Errorf("%s: byte %d returned event early", tt.name, i)
				}
			}
		}
	}
}

func TestKeyDecoder_Reset(t *testing.T) {
	d := &KeyDecoder{}

	// Start an ESC sequence
	d.PushByte(0x1b)
	d.PushByte('[')

	// Reset should return to ground state
	d.Reset()

	// Should now handle normal input
	evt, ok := d.PushByte('a')
	if !ok {
		t.Errorf("After reset, PushByte('a') returned false")
	}
	if evt.Key != event.KeyRune || evt.Rune != 'a' {
		t.Errorf("After reset, got unexpected event: key=%v rune=%c", evt.Key, evt.Rune)
	}
}
