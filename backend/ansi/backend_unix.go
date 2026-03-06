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
const escTimeout = 10 * time.Millisecond

// ansiBackend implements the Backend interface for Unix terminals using ANSI escape sequences.
type ansiBackend struct {
	origTermios *unix.Termios
	r           *os.File
	w           *bufio.Writer
	decoder     KeyDecoder
	size        geom.Size
	signals     *signalHandler
	eventCh     chan event.Event
	pendingByte byte     // Byte to process before next Read
	hasPending  bool     // Whether pendingByte is valid
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
// It always attempts to restore the terminal even if flushing fails.
func (b *ansiBackend) Restore() error {
	var firstErr error

	// Flush any pending output BEFORE restoring terminal settings
	if err := b.w.Flush(); err != nil && firstErr == nil {
		firstErr = err
	}

	// Stop signal handler
	if b.signals != nil {
		b.signals.Stop()
	}

	// Restore terminal settings
	if b.origTermios != nil {
		if err := restore(b.origTermios); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
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

// readByteWithTimeout tries to read a single byte with a timeout using poll.
// Returns (byte, true, nil) if a byte was read, (0, false, nil) on timeout, or error.
func (b *ansiBackend) readByteWithTimeout(timeout time.Duration) (byte, bool, error) {
	fd := int(b.r.Fd())

	// Use poll to wait for input with timeout
	pollFds := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}

	// Convert timeout to milliseconds
	millis := int(timeout.Milliseconds())
	if millis < 1 {
		millis = 1
	}

	n, err := unix.Poll(pollFds, millis)
	if err != nil {
		return 0, false, err
	}
	if n == 0 {
		// Timeout
		return 0, false, nil
	}

	// Data is available, read one byte
	var buf [1]byte
	nn, err := b.r.Read(buf[:])
	if err != nil {
		return 0, false, err
	}
	if nn == 0 {
		return 0, false, io.EOF
	}
	if nn != 1 {
		return 0, false, io.ErrShortBuffer
	}
	return buf[0], true, nil
}

func emitKeyEvents(ch chan event.Event, evs []event.KeyEvent) {
	for _, ev := range evs {
		ch <- ev
	}
}

// readEvents runs in a goroutine, reading input and publishing events.
func (b *ansiBackend) readEvents() {
	defer func() {
		// Resolve incomplete decoder state at EOF.
		var evs []event.KeyEvent
		evs = b.decoder.Finalize(evs)
		emitKeyEvents(b.eventCh, evs)

		// Close event channel to signal graceful shutdown
		close(b.eventCh)
	}()

	buf := make([]byte, 256)

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

		var n int
		var err error

		// If we have a pending byte, process it before reading
		if b.hasPending {
			ch := b.pendingByte
			b.hasPending = false
			buf[0] = ch
			n = 1
		} else {
			// Read input
			n, err = b.r.Read(buf)
			if err != nil {
				if err == io.EOF {
					return
				}
				continue
			}
			if n == 0 {
				continue
			}
		}

		// Process each byte
		for i := 0; i < n; i++ {
			ch := buf[i]

			// Special-case ESC for timeout-based disambiguation
			if ch == 0x1b {
				// Only perform ESC disambiguation if decoder is in ground state.
				// If decoder is mid-sequence (UTF-8 or CSI), we need to abort that
				// state first before handling the new ESC.
				if b.decoder.State() != stateGround {
					var evs []event.KeyEvent
					evs = b.decoder.Abort(evs)
					emitKeyEvents(b.eventCh, evs)
					// Now decoder is in ground state, continue to normal ESC handling
					// (fall through to the same ESC handling logic below)
				}

				var next byte
				var ok bool
				var fromTimeout bool

				// Check if next byte is already in the buffer
				if i+1 < n {
					i++
					next = buf[i]
					ok = true
					fromTimeout = false
				} else {
					// At end of buffer - wait a bit to see if more is coming
					next, ok, err = b.readByteWithTimeout(escTimeout)
					fromTimeout = true
					if err != nil {
						if err == io.EOF {
							b.eventCh <- event.KeyEvent{Key: event.KeyEsc}
							return
						}
						b.eventCh <- event.KeyEvent{Key: event.KeyEsc}
						continue
					}
				}

				if !ok {
					// Timeout => standalone Escape
					b.eventCh <- event.KeyEvent{Key: event.KeyEsc}
					continue
				}

				if next == '[' {
					// ESC [ => start CSI sequence
					b.decoder.StartCSI()
					continue
				}

				// Emit the standalone Escape
				b.eventCh <- event.KeyEvent{Key: event.KeyEsc}

				// Push back the lookahead byte for normal processing
				if fromTimeout {
					// Store in pending slot for next iteration
					b.pendingByte = next
					b.hasPending = true
				} else {
					// Decrement index so the byte is reprocessed in this loop
					i--
				}
				continue
			}

			// Normal byte processing
			var evs []event.KeyEvent
			evs = b.decoder.PushByte(evs, ch)
			emitKeyEvents(b.eventCh, evs)
		}
	}
}
