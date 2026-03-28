//go:build unix

package ansi

import (
	"io"
	"os"
	"testing"
	"time"

	"github.com/losinggeneration/tui/event"
	"golang.org/x/sys/unix"
)

// testBackend creates a test backend with a pipe for input.
// The write end is returned for writing test data.
type testBackend struct {
	backend *Backend
	r       *os.File // read end of pipe (used by backend)
	w       *os.File // write end of pipe (for test input)
	pipeR   int      // wake pipe read end (raw fd)
	pipeW   int      // wake pipe write end (raw fd)
	t       *testing.T
}

// newTestBackend creates a new test backend.
func newTestBackend(t *testing.T) (*testBackend, error) {
	t.Helper()
	// Create input pipe
	r, w, err := os.Pipe()
	if err != nil {
		return nil, err
	}

	// Create wake pipe using raw fds (matches production)
	var pipeFds [2]int
	if err := unix.Pipe2(pipeFds[:], unix.O_NONBLOCK|unix.O_CLOEXEC); err != nil {
		closer(t, r)
		closer(t, w)

		return nil, err
	}

	b := &Backend{
		r:        r,
		eventCh:  make(chan event.Event, 10),
		pipeR:    pipeFds[0],
		pipeW:    pipeFds[1],
		readDone: make(chan struct{}),
	}
	b.readStarted.Store(true)
	b.decoder.Reset()

	tb := &testBackend{
		backend: b,
		r:       r,
		w:       w,
		pipeR:   pipeFds[0],
		pipeW:   pipeFds[1],
		t:       t,
	}

	return tb, nil
}

func closer(t *testing.T, c io.Closer) {
	t.Helper()

	if c == nil {
		return
	}

	if err := c.Close(); err != nil {
		t.Error(err)
	}
}

// writeInput writes bytes to the input pipe.
func (tb *testBackend) writeInput(data []byte) error {
	_, err := tb.w.Write(data)

	return err
}

// closeInput closes the write end of the input pipe (simulating EOF).
func (tb *testBackend) closeInput() {
	closer(tb.t, tb.w)
	tb.w = nil
}

// readAllEvents reads all events from the channel until it closes or timeout.
func (tb *testBackend) readAllEvents(timeout time.Duration) []event.Event {
	return readAllEventsFromChannel(tb.backend.eventCh, timeout)
}

// readAllEventsFromChannel reads all events from the channel until it closes or timeout.
func readAllEventsFromChannel(ch <-chan event.Event, timeout time.Duration) []event.Event {
	var events []event.Event

	timeoutCh := time.After(timeout)

	for {
		select {
		case e, ok := <-ch:
			if !ok {
				return events
			}

			events = append(events, e)
		case <-timeoutCh:
			return events
		}
	}
}

// TestBackend_SplitCSISequence verifies that a CSI sequence within
// one read becomes KeyUp. This is a regression test for
// timeout-based flushing misdecode.
func TestBackend_SplitCSISequence(t *testing.T) {
	tb, err := newTestBackend(t)
	if err != nil {
		t.Fatal(err)
	}

	go tb.backend.readEvents()

	// Write CSI sequence
	if err := tb.writeInput([]byte{0x1b, '[', 'A'}); err != nil {
		t.Fatal(err)
	}
	// Small delay to ensure data is written before closing
	time.Sleep(5 * time.Millisecond)
	tb.closeInput()

	events := tb.readAllEvents(100 * time.Millisecond)

	// Should receive KeyUp, NOT KeyEsc + [ + A
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1: %#v", len(events), events)
	}

	ke, ok := events[0].(event.KeyEvent)
	if !ok {
		t.Fatalf("got non-KeyEvent: %#v", events[0])
	}

	if ke.Key != event.KeyUp {
		t.Errorf("got %v, want KeyUp", ke.Key)
	}
}

// TestBackend_StandaloneESCatEOF verifies that a standalone ESC at EOF
// becomes KeyEsc.
func TestBackend_StandaloneESCatEOF(t *testing.T) {
	tb, err := newTestBackend(t)
	if err != nil {
		t.Fatal(err)
	}

	go tb.backend.readEvents()

	if err := tb.writeInput([]byte{0x1b}); err != nil {
		t.Fatal(err)
	}

	tb.closeInput()

	events := tb.readAllEvents(100 * time.Millisecond)

	// Should receive KeyEsc
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1: %#v", len(events), events)
	}

	ke, ok := events[0].(event.KeyEvent)
	if !ok {
		t.Fatalf("got non-KeyEvent: %#v", events[0])
	}

	if ke.Key != event.KeyEsc {
		t.Errorf("got %v, want KeyEsc", ke.Key)
	}
}

// TestBackend_PartialCSIFinalized verifies that a partial CSI at EOF
// is finalized literally.
func TestBackend_PartialCSIFinalized(t *testing.T) {
	tb, err := newTestBackend(t)
	if err != nil {
		t.Fatal(err)
	}

	go tb.backend.readEvents()

	if err := tb.writeInput([]byte{0x1b, '[', '1'}); err != nil {
		t.Fatal(err)
	}
	// Give goroutine time to read and process data before closing
	time.Sleep(50 * time.Millisecond)
	tb.closeInput()

	events := tb.readAllEvents(100 * time.Millisecond)

	// Should receive ESC, [, 1 as literal events
	expected := []event.Key{event.KeyEsc, event.KeyRune, event.KeyRune}
	expectedRunes := []rune{'[', '1'}

	if len(events) != 3 {
		t.Fatalf("got %d events, want 3: %#v", len(events), events)
	}

	for i := range 3 {
		ke, ok := events[i].(event.KeyEvent)
		if !ok {
			t.Fatalf("event %d: non-KeyEvent: %#v", i, events[i])
		}

		if ke.Key != expected[i] {
			t.Errorf("event %d: got %v, want %v", i, ke.Key, expected[i])
		}

		if ke.Key == event.KeyRune && ke.Rune != expectedRunes[i-1] {
			t.Errorf("event %d: got rune %c, want %c", i, ke.Rune, expectedRunes[i-1])
		}
	}
}

// TestBackend_EscAltRune verifies that ESC + letter becomes Alt+Rune.
func TestBackend_EscAltRune(t *testing.T) {
	tb, err := newTestBackend(t)
	if err != nil {
		t.Fatal(err)
	}

	go tb.backend.readEvents()

	if err := tb.writeInput([]byte{0x1b, 'a'}); err != nil {
		t.Fatal(err)
	}
	// Small delay to ensure data is written before closing
	time.Sleep(5 * time.Millisecond)
	tb.closeInput()

	events := tb.readAllEvents(100 * time.Millisecond)

	// Should receive Alt+a
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1: %#v", len(events), events)
	}

	ke, ok := events[0].(event.KeyEvent)
	if !ok {
		t.Fatalf("got non-KeyEvent: %#v", events[0])
	}

	if ke.Key != event.KeyRune || ke.Rune != 'a' {
		t.Errorf("got %v/%c, want KeyRune/'a'", ke.Key, ke.Rune)
	}

	if ke.Mod != event.ModAlt {
		t.Errorf("got Mod %v, want ModAlt", ke.Mod)
	}
}

// TestBackend_PartialSS3Finalized verifies that a partial SS3 at EOF
// is finalized literally.
func TestBackend_PartialSS3Finalized(t *testing.T) {
	tb, err := newTestBackend(t)
	if err != nil {
		t.Fatal(err)
	}

	go tb.backend.readEvents()

	if err := tb.writeInput([]byte{0x1b, 'O'}); err != nil {
		t.Fatal(err)
	}
	// Give goroutine time to read and process data before closing
	time.Sleep(50 * time.Millisecond)
	tb.closeInput()

	events := tb.readAllEvents(100 * time.Millisecond)

	// Should receive ESC, O as literal events
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2: %#v", len(events), events)
	}

	ke1, ok := events[0].(event.KeyEvent)
	if !ok {
		t.Fatalf("event 0: non-KeyEvent: %#v", events[0])
	}

	if ke1.Key != event.KeyEsc {
		t.Errorf("event 0: got %v, want KeyEsc", ke1.Key)
	}

	ke2, ok := events[1].(event.KeyEvent)
	if !ok {
		t.Fatalf("event 1: non-KeyEvent: %#v", events[1])
	}

	if ke2.Key != event.KeyRune || ke2.Rune != 'O' {
		t.Errorf("event 1: got %v/%c, want KeyRune/'O'", ke2.Key, ke2.Rune)
	}
}

// TestBackend_EscEscDouble verifies that two consecutive ESC keys
// emit two KeyEsc events.
func TestBackend_EscEscDouble(t *testing.T) {
	tb, err := newTestBackend(t)
	if err != nil {
		t.Fatal(err)
	}

	go tb.backend.readEvents()

	if err := tb.writeInput([]byte{0x1b, 0x1b}); err != nil {
		t.Fatal(err)
	}
	// Small delay to ensure data is written before closing
	time.Sleep(5 * time.Millisecond)
	tb.closeInput()

	events := tb.readAllEvents(100 * time.Millisecond)

	// Should receive two KeyEsc events
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2: %#v", len(events), events)
	}

	for i := range 2 {
		ke, ok := events[i].(event.KeyEvent)
		if !ok {
			t.Fatalf("event %d: non-KeyEvent: %#v", i, events[i])
		}

		if ke.Key != event.KeyEsc {
			t.Errorf("event %d: got %v, want KeyEsc", i, ke.Key)
		}
	}
}

// TestEOFBehavior verifies EOF handling with various decoder states.
func TestEOFBehavior(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		wantKeys []event.Key
	}{
		{
			name:     "empty input",
			input:    []byte{},
			wantKeys: []event.Key{},
		},
		{
			name:     "single ESC",
			input:    []byte{0x1b},
			wantKeys: []event.Key{event.KeyEsc},
		},
		{
			name:     "complete CSI",
			input:    []byte{0x1b, '[', 'A'},
			wantKeys: []event.Key{event.KeyUp},
		},
		{
			name:     "partial CSI",
			input:    []byte{0x1b, '['},
			wantKeys: []event.Key{event.KeyEsc, event.KeyRune},
		},
		{
			name:     "partial SS3",
			input:    []byte{0x1b, 'O'},
			wantKeys: []event.Key{event.KeyEsc, event.KeyRune},
		},
		{
			name:     "simple rune",
			input:    []byte{'a'},
			wantKeys: []event.Key{event.KeyRune},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tb, err := newTestBackend(t)
			if err != nil {
				t.Fatal(err)
			}

			go tb.backend.readEvents()

			if err := tb.writeInput(tt.input); err != nil {
				t.Fatal(err)
			}
			// Small delay to ensure data is written before closing
			time.Sleep(5 * time.Millisecond)
			tb.closeInput()

			events := tb.readAllEvents(100 * time.Millisecond)

			if len(events) != len(tt.wantKeys) {
				t.Fatalf("got %d events, want %d: %#v", len(events), len(tt.wantKeys), events)
			}

			for i, wantKey := range tt.wantKeys {
				ke, ok := events[i].(event.KeyEvent)
				if !ok {
					t.Fatalf("event %d: non-KeyEvent: %#v", i, events[i])
				}

				if ke.Key != wantKey {
					t.Errorf("event %d: got %v, want %v", i, ke.Key, wantKey)
				}
			}
		})
	}
}

// TestBackend_AltKey verifies Alt key combinations.
func TestBackend_AltKey(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		wantKey  event.Key
		wantRune rune
		wantMod  event.ModMask
	}{
		{
			name:     "Alt+a",
			input:    []byte{0x1b, 'a'},
			wantKey:  event.KeyRune,
			wantRune: 'a',
			wantMod:  event.ModAlt,
		},
		{
			name:     "Alt+Z",
			input:    []byte{0x1b, 'Z'},
			wantKey:  event.KeyRune,
			wantRune: 'Z',
			wantMod:  event.ModAlt,
		},
		{
			name:     "Alt+0",
			input:    []byte{0x1b, '0'},
			wantKey:  event.KeyRune,
			wantRune: '0',
			wantMod:  event.ModAlt,
		},
		{
			name:     "Alt+space",
			input:    []byte{0x1b, ' '},
			wantKey:  event.KeyRune,
			wantRune: ' ',
			wantMod:  event.ModAlt,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tb, err := newTestBackend(t)
			if err != nil {
				t.Fatal(err)
			}

			go tb.backend.readEvents()

			if err := tb.writeInput(tt.input); err != nil {
				t.Fatal(err)
			}
			// Small delay to ensure data is written before closing
			time.Sleep(5 * time.Millisecond)
			tb.closeInput()

			events := tb.readAllEvents(100 * time.Millisecond)

			if len(events) != 1 {
				t.Fatalf("got %d events, want 1: %#v", len(events), events)
			}

			ke, ok := events[0].(event.KeyEvent)
			if !ok {
				t.Fatalf("got non-KeyEvent: %#v", events[0])
			}

			if ke.Key != tt.wantKey {
				t.Errorf("got Key %v, want %v", ke.Key, tt.wantKey)
			}

			if ke.Rune != tt.wantRune {
				t.Errorf("got Rune %c, want %c", ke.Rune, tt.wantRune)
			}

			if ke.Mod != tt.wantMod {
				t.Errorf("got Mod %v, want %v", ke.Mod, tt.wantMod)
			}
		})
	}
}

// TestBackend_UnknownCSI verifies that unknown CSI sequences are
// preserved literally.
func TestBackend_UnknownCSI(t *testing.T) {
	tb, err := newTestBackend(t)
	if err != nil {
		t.Fatal(err)
	}

	go tb.backend.readEvents()

	// CSI with unknown final byte 'Q'
	if err := tb.writeInput([]byte{0x1b, '[', 'Q'}); err != nil {
		t.Fatal(err)
	}
	// Small delay to ensure data is written before closing
	time.Sleep(5 * time.Millisecond)
	tb.closeInput()

	events := tb.readAllEvents(100 * time.Millisecond)

	// Should receive ESC, [, Q as literal events
	if len(events) != 3 {
		t.Fatalf("got %d events, want 3: %#v", len(events), events)
	}

	expectedKeys := []event.Key{event.KeyEsc, event.KeyRune, event.KeyRune}
	expectedRunes := []rune{'[', 'Q'}

	for i := range 3 {
		ke, ok := events[i].(event.KeyEvent)
		if !ok {
			t.Fatalf("event %d: non-KeyEvent: %#v", i, events[i])
		}

		if ke.Key != expectedKeys[i] {
			t.Errorf("event %d: got %v, want %v", i, ke.Key, expectedKeys[i])
		}

		if ke.Key == event.KeyRune && ke.Rune != expectedRunes[i-1] {
			t.Errorf("event %d: got rune %c, want %c", i, ke.Rune, expectedRunes[i-1])
		}
	}
}

// TestBackend_CSIPrivateMarker verifies that CSI sequences with
// private markers (like CSI ? A) are preserved literally, not
// misinterpreted as arrow keys.
func TestBackend_CSIPrivateMarker(t *testing.T) {
	tests := []struct {
		name      string
		input     []byte
		numEvents int
	}{
		{
			name:      "CSI ? A (private modifier)",
			input:     []byte{0x1b, '[', '?', 'A'},
			numEvents: 4, // ESC, [, ?, A as literals
		},
		{
			name:      "CSI > A (private modifier)",
			input:     []byte{0x1b, '[', '>', 'A'},
			numEvents: 4, // ESC, [, >, A as literals
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tb, err := newTestBackend(t)
			if err != nil {
				t.Fatal(err)
			}

			go tb.backend.readEvents()

			if err := tb.writeInput(tt.input); err != nil {
				t.Fatal(err)
			}

			time.Sleep(5 * time.Millisecond)
			tb.closeInput()

			events := tb.readAllEvents(100 * time.Millisecond)

			if len(events) != tt.numEvents {
				t.Fatalf("got %d events, want %d: %#v", len(events), tt.numEvents, events)
			}

			// First event should always be KeyEsc
			ke, ok := events[0].(event.KeyEvent)
			if !ok || ke.Key != event.KeyEsc {
				t.Errorf("event 0: want KeyEsc, got %#v", events[0])
			}
		})
	}
}

// TestBackend_ChunkedInput verifies that input arriving in chunks
// (simulating split reads) is handled correctly.
func TestBackend_ChunkedInput(t *testing.T) {
	tests := []struct {
		name     string
		chunks   [][]byte
		wantKeys []event.Key
	}{
		{
			name:     "CSI sequence in one read",
			chunks:   [][]byte{{0x1b, '[', 'A'}},
			wantKeys: []event.Key{event.KeyUp},
		},
		{
			name:     "Alt+a in one read",
			chunks:   [][]byte{{0x1b, 'a'}},
			wantKeys: []event.Key{event.KeyRune},
		},
		{
			name:     "SS3 sequence in one read",
			chunks:   [][]byte{{0x1b, 'O', 'A'}},
			wantKeys: []event.Key{event.KeyUp},
		},
		{
			name:     "two ESC keys in one read",
			chunks:   [][]byte{{0x1b, 0x1b}},
			wantKeys: []event.Key{event.KeyEsc, event.KeyEsc},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tb, err := newTestBackend(t)
			if err != nil {
				t.Fatal(err)
			}

			go tb.backend.readEvents()

			// Write all data at once (simulates data arriving within same readiness cycle)
			for _, chunk := range tt.chunks {
				if err := tb.writeInput(chunk); err != nil {
					t.Fatal(err)
				}
			}
			// Small delay to ensure data is written before closing
			time.Sleep(5 * time.Millisecond)
			tb.closeInput()

			events := tb.readAllEvents(100 * time.Millisecond)

			if len(events) != len(tt.wantKeys) {
				t.Fatalf("got %d events, want %d: %#v", len(events), len(tt.wantKeys), events)
			}

			for i, wantKey := range tt.wantKeys {
				ke, ok := events[i].(event.KeyEvent)
				if !ok {
					t.Fatalf("event %d: non-KeyEvent: %#v", i, events[i])
				}

				if ke.Key != wantKey {
					t.Errorf("event %d: got %v, want %v", i, ke.Key, wantKey)
				}
			}
		})
	}
}

// TestBackend_MultipleCSI verifies that multiple CSI sequences in one
// read are decoded correctly.
func TestBackend_MultipleCSI(t *testing.T) {
	tb, err := newTestBackend(t)
	if err != nil {
		t.Fatal(err)
	}

	go tb.backend.readEvents()

	// Up, Down, Left, Right
	input := []byte{
		0x1b, '[', 'A', // Up
		0x1b, '[', 'B', // Down
		0x1b, '[', 'D', // Left
		0x1b, '[', 'C', // Right
	}
	if err := tb.writeInput(input); err != nil {
		t.Fatal(err)
	}
	// Small delay to ensure data is written before closing
	time.Sleep(5 * time.Millisecond)
	tb.closeInput()

	events := tb.readAllEvents(100 * time.Millisecond)

	wantKeys := []event.Key{event.KeyUp, event.KeyDown, event.KeyLeft, event.KeyRight}
	if len(events) != len(wantKeys) {
		t.Fatalf("got %d events, want %d: %#v", len(events), len(wantKeys), events)
	}

	for i, wantKey := range wantKeys {
		ke, ok := events[i].(event.KeyEvent)
		if !ok {
			t.Fatalf("event %d: non-KeyEvent: %#v", i, events[i])
		}

		if ke.Key != wantKey {
			t.Errorf("event %d: got %v, want %v", i, ke.Key, wantKey)
		}
	}
}

// TestBackend_MixedEvents verifies mixed keyboard and other events.
func TestBackend_MixedEvents(t *testing.T) {
	tb, err := newTestBackend(t)
	if err != nil {
		t.Fatal(err)
	}

	go tb.backend.readEvents()

	// 'a', Up, 'b', Down
	input := []byte{
		'a',
		0x1b, '[', 'A',
		'b',
		0x1b, '[', 'B',
	}
	if err := tb.writeInput(input); err != nil {
		t.Fatal(err)
	}
	// Small delay to ensure data is written before closing
	time.Sleep(5 * time.Millisecond)
	tb.closeInput()

	events := tb.readAllEvents(100 * time.Millisecond)

	wantKeys := []event.Key{event.KeyRune, event.KeyUp, event.KeyRune, event.KeyDown}
	wantRunes := []rune{'a', 0, 'b', 0}

	if len(events) != len(wantKeys) {
		t.Fatalf("got %d events, want %d: %#v", len(events), len(wantKeys), events)
	}

	for i, wantKey := range wantKeys {
		ke, ok := events[i].(event.KeyEvent)
		if !ok {
			t.Fatalf("event %d: non-KeyEvent: %#v", i, events[i])
		}

		if ke.Key != wantKey {
			t.Errorf("event %d: got %v, want %v", i, ke.Key, wantKey)
		}

		if ke.Rune != wantRunes[i] {
			t.Errorf("event %d: got rune %c, want %c", i, ke.Rune, wantRunes[i])
		}
	}
}

// TestBackend_SingleESCatEOF verifies that a standalone ESC at EOF
// becomes KeyEsc. This is the correct EOF-driven behavior, not timeout-based.
func TestBackend_SingleESCatEOF(t *testing.T) {
	tb, err := newTestBackend(t)
	if err != nil {
		t.Fatal(err)
	}

	go tb.backend.readEvents()

	if err := tb.writeInput([]byte{0x1b}); err != nil {
		t.Fatal(err)
	}

	tb.closeInput()

	events := tb.readAllEvents(100 * time.Millisecond)

	// Should receive KeyEsc (from Finalize in defer)
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1: %#v", len(events), events)
	}

	ke, ok := events[0].(event.KeyEvent)
	if !ok {
		t.Fatalf("got non-KeyEvent: %#v", events[0])
	}

	if ke.Key != event.KeyEsc {
		t.Errorf("got %v, want KeyEsc", ke.Key)
	}
}

// TestBackend_StandaloneESCPrompt verifies that a standalone ESC
// is delivered promptly without EOF, via FlushPending at the read boundary.
// This tests the interactive behavior where a user presses Escape and then
// does nothing else - ESC should be emitted when stdin is no longer readable.
func TestBackend_StandaloneESCPrompt(t *testing.T) {
	tb, err := newTestBackend(t)
	if err != nil {
		t.Fatal(err)
	}

	go tb.backend.readEvents()

	// Write ESC byte (no EOF - input stays open)
	if err := tb.writeInput([]byte{0x1b}); err != nil {
		t.Fatal(err)
	}

	// Should receive KeyEsc promptly (from FlushPending at read boundary)
	// even though we haven't closed the input
	events := tb.readAllEvents(50 * time.Millisecond)

	if len(events) != 1 {
		t.Fatalf("got %d events, want 1: %#v", len(events), events)
	}

	ke, ok := events[0].(event.KeyEvent)
	if !ok {
		t.Fatalf("got non-KeyEvent: %#v", events[0])
	}

	if ke.Key != event.KeyEsc {
		t.Errorf("got %v, want KeyEsc", ke.Key)
	}
}

// TestBackend_EscEscPrompt verifies that two consecutive ESC keys
// emit two KeyEsc events promptly without EOF.
func TestBackend_EscEscPrompt(t *testing.T) {
	tb, err := newTestBackend(t)
	if err != nil {
		t.Fatal(err)
	}

	go tb.backend.readEvents()

	// Write two ESC bytes (no EOF)
	if err := tb.writeInput([]byte{0x1b, 0x1b}); err != nil {
		t.Fatal(err)
	}

	// Should receive two KeyEsc events
	events := tb.readAllEvents(50 * time.Millisecond)

	if len(events) != 2 {
		t.Fatalf("got %d events, want 2: %#v", len(events), events)
	}

	for i := range 2 {
		ke, ok := events[i].(event.KeyEvent)
		if !ok {
			t.Fatalf("event %d: non-KeyEvent: %#v", i, events[i])
		}

		if ke.Key != event.KeyEsc {
			t.Errorf("event %d: got %v, want KeyEsc", i, ke.Key)
		}
	}
}

// TestBackend_EOFAfterData verifies that decoded events are not
// lost when EOF occurs after a successful read.
func TestBackend_EOFAfterData(t *testing.T) {
	tb, err := newTestBackend(t)
	if err != nil {
		t.Fatal(err)
	}

	go tb.backend.readEvents()

	// Write data with multiple valid sequences and immediately close (no sleep)
	// Input: 'a', CSI A (Up), ESC (standalone)
	if err := tb.writeInput([]byte{'a', 0x1b, '[', 'A', 0x1b}); err != nil {
		t.Fatal(err)
	}

	tb.closeInput()

	events := tb.readAllEvents(100 * time.Millisecond)

	// Should receive 'a', KeyUp, and ESC
	if len(events) != 3 {
		t.Fatalf("got %d events, want 3: %#v", len(events), events)
	}

	// First event: 'a'
	if ke, ok := events[0].(event.KeyEvent); !ok || ke.Key != event.KeyRune || ke.Rune != 'a' {
		t.Errorf("event 0: want 'a', got %#v", events[0])
	}

	// Second event: KeyUp
	if ke, ok := events[1].(event.KeyEvent); !ok || ke.Key != event.KeyUp {
		t.Errorf("event 1: want KeyUp, got %#v", events[1])
	}

	// Third event: ESC (from Finalize on EOF)
	if ke, ok := events[2].(event.KeyEvent); !ok || ke.Key != event.KeyEsc {
		t.Errorf("event 2: want KeyEsc, got %#v", events[2])
	}
}

// TestBackend_SS3 verifies SS3 (application keypad) sequences.
func TestBackend_SS3(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		wantKey event.Key
	}{
		{
			name:    "SS3 Up (O A)",
			input:   []byte{0x1b, 'O', 'A'},
			wantKey: event.KeyUp,
		},
		{
			name:    "SS3 Down (O B)",
			input:   []byte{0x1b, 'O', 'B'},
			wantKey: event.KeyDown,
		},
		{
			name:    "SS3 Right (O C)",
			input:   []byte{0x1b, 'O', 'C'},
			wantKey: event.KeyRight,
		},
		{
			name:    "SS3 Left (O D)",
			input:   []byte{0x1b, 'O', 'D'},
			wantKey: event.KeyLeft,
		},
		{
			name:    "SS3 F1 (O P)",
			input:   []byte{0x1b, 'O', 'P'},
			wantKey: event.KeyF1,
		},
		{
			name:    "SS3 F2 (O Q)",
			input:   []byte{0x1b, 'O', 'Q'},
			wantKey: event.KeyF2,
		},
		{
			name:    "SS3 F3 (O R)",
			input:   []byte{0x1b, 'O', 'R'},
			wantKey: event.KeyF3,
		},
		{
			name:    "SS3 F4 (O S)",
			input:   []byte{0x1b, 'O', 'S'},
			wantKey: event.KeyF4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tb, err := newTestBackend(t)
			if err != nil {
				t.Fatal(err)
			}

			go tb.backend.readEvents()

			if err := tb.writeInput(tt.input); err != nil {
				t.Fatal(err)
			}
			// Small delay to ensure data is written before closing
			time.Sleep(5 * time.Millisecond)
			tb.closeInput()

			events := tb.readAllEvents(100 * time.Millisecond)

			if len(events) != 1 {
				t.Fatalf("got %d events, want 1: %#v", len(events), events)
			}

			ke, ok := events[0].(event.KeyEvent)
			if !ok {
				t.Fatalf("got non-KeyEvent: %#v", events[0])
			}

			if ke.Key != tt.wantKey {
				t.Errorf("got Key %v, want %v", ke.Key, tt.wantKey)
			}
		})
	}
}
