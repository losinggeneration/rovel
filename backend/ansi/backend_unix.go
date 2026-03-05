//go:build unix

package ansi

import (
	"bufio"
	"io"
	"os"
	"time"

	"github.com/losinggeneration/tui/event"
	"github.com/losinggeneration/tui/geom"
	"golang.org/x/sys/unix"
)

// escTimeout is the timeout for determining if ESC is standalone or part of a sequence.
const escTimeout = 20 * time.Millisecond

// ansiBackend implements the Backend interface for Unix terminals using ANSI escape sequences.
type ansiBackend struct {
	origTermios *unix.Termios
	r           io.Reader
	w           *bufio.Writer
	decoder     KeyDecoder
	size        geom.Size
	signals     *signalHandler
	eventCh     chan event.Event
	lastWasESC  bool
}

// New creates a new ANSI backend.
func New() (*ansiBackend, error) {
	w := bufio.NewWriterSize(os.Stdout, 64*1024)

	size, err := getTerminalSize()
	if err != nil {
		return nil, err
	}

	b := &ansiBackend{
		r:       os.Stdin,
		w:       w,
		size:    size,
		eventCh: make(chan event.Event, 8),
	}

	return b, nil
}

// Enable enables the terminal and returns the initial size.
func (b *ansiBackend) Enable() (geom.Size, error) {
	orig, err := enableRaw()
	if err != nil {
		return geom.Size{}, err
	}
	b.origTermios = orig

	// Setup signal handler for resize events
	signals, err := setupResizeHandler()
	if err != nil {
		restore(orig)
		return geom.Size{}, err
	}
	b.signals = signals

	// Start event reader goroutine
	go b.readEvents()

	// Get initial size
	size, err := getTerminalSize()
	if err != nil {
		return geom.Size{}, err
	}
	b.size = size

	return size, nil
}

// Restore restores the terminal to its original state.
func (b *ansiBackend) Restore() error {
	// Stop signal handler
	if b.signals != nil {
		b.signals.Stop()
	}

	// Restore terminal settings
	if b.origTermios != nil {
		return restore(b.origTermios)
	}
	return nil
}

// Write writes raw ANSI output to the terminal.
func (b *ansiBackend) Write(p []byte) (int, error) {
	return b.w.Write(p)
}

// Flush flushes any buffered output.
func (b *ansiBackend) Flush() error {
	return b.w.Flush()
}

// ReadEvent reads and returns the next event, blocking until one is available.
func (b *ansiBackend) ReadEvent() event.Event {
	return <-b.eventCh
}

// Size returns the current terminal size.
func (b *ansiBackend) Size() geom.Size {
	return b.size
}

// readEvents runs in a goroutine, reading input and publishing events.
func (b *ansiBackend) readEvents() {
	buf := make([]byte, 256)
	escTimeoutCh := make(chan struct{}, 1)

	for {
		// Check for resize events first (non-blocking)
		select {
		case size, ok := <-b.signals.ResizeChan():
			if ok {
				b.size = size
				b.eventCh <- event.ResizeEvent{W: size.W, H: size.H}
			}
		default:
		}

		// Check for ESC timeout
		select {
		case <-escTimeoutCh:
			// ESC timeout expired - treat as standalone ESC
			b.eventCh <- event.KeyEvent{Key: event.KeyEsc}
			b.decoder.Reset()
		default:
		}

		// Set read deadline for ESC timeout handling
		if b.lastWasESC {
			b.r.(interface{ SetReadDeadline(time.Time) error }).SetReadDeadline(time.Now().Add(escTimeout))
		} else {
			b.r.(interface{ SetReadDeadline(time.Time) error }).SetReadDeadline(time.Time{})
		}

		// Read input
		n, err := b.r.Read(buf)
		if err != nil {
			if netErr, ok := err.(interface{ Timeout() bool }); ok && netErr.Timeout() {
				// Read timeout - handle ESC timeout
				if b.lastWasESC {
					select {
					case escTimeoutCh <- struct{}{}:
					default:
					}
				}
				b.lastWasESC = false
				continue
			}
			if err == io.EOF {
				return
			}
			continue
		}

		// Process each byte
		for i := 0; i < n; i++ {
			evt, ok := b.decoder.PushByte(buf[i])
			if ok {
				b.lastWasESC = false
				b.eventCh <- evt
				continue
			}

			// Track ESC for timeout handling
			if buf[i] == 0x1b {
				b.lastWasESC = true
			}
		}
	}
}
