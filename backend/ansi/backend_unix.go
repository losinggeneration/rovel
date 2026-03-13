//go:build unix

package ansi

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"

	"github.com/losinggeneration/tui/errors"
	"github.com/losinggeneration/tui/event"
	"github.com/losinggeneration/tui/geom"
	"golang.org/x/sys/unix"
)

// ansiBackend implements the Backend interface for Unix terminals using ANSI escape sequences.
type ansiBackend struct {
	origTermios *unix.Termios
	r           *os.File
	w           *bufio.Writer
	decoder     KeyDecoder
	signals     *signalHandler
	eventCh     chan event.Event

	// Self-pipe for waking poll on resize/shutdown
	pipeR *os.File // read end; owned/closed by readEvents()
	pipeW *os.File // write end; owned/closed by Restore()

	// Read-loop lifecycle
	readStarted  atomic.Bool
	readDone     chan struct{}
	shutdownCh   chan struct{}
	shutdownOnce sync.Once

	// Terminal size, protected by sizeMu
	sizeMu sync.RWMutex
	size   geom.Size
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

	// Create self-pipe for waking poll (after raw mode succeeds)
	pipeR, pipeW, err := os.Pipe()
	if err != nil {
		errors.Add(restore(orig))

		return geom.Size{}, err
	}

	// Set both ends to non-blocking
	if err := unix.SetNonblock(int(pipeR.Fd()), true); err != nil {
		errors.Add(pipeR.Close())
		errors.Add(pipeW.Close())
		errors.Add(restore(orig))
		pipeR = nil
		pipeW = nil

		return geom.Size{}, err
	}
	if err := unix.SetNonblock(int(pipeW.Fd()), true); err != nil {
		errors.Add(pipeR.Close())
		errors.Add(pipeW.Close())
		errors.Add(restore(orig))
		pipeR = nil
		pipeW = nil

		return geom.Size{}, err
	}

	b.pipeR = pipeR
	b.pipeW = pipeW

	// Setup signal handler for resize events
	signals, err := setupResizeHandler(b.pipeW)
	if err != nil {
		errors.Add(pipeR.Close())
		errors.Add(pipeW.Close())
		errors.Add(restore(orig))
		pipeR = nil
		pipeW = nil

		return geom.Size{}, err
	}
	b.signals = signals

	// Get initial size before starting background goroutines.
	size, err := getTerminalSize()
	if err != nil {
		b.signals.Stop()
		b.signals = nil

		errors.Add(b.pipeR.Close())
		errors.Add(b.pipeW.Close())
		b.pipeR = nil
		b.pipeW = nil

		errors.Add(restore(orig))
		b.origTermios = nil
		return geom.Size{}, err
	}

	b.sizeMu.Lock()
	b.size = size
	b.sizeMu.Unlock()

	// Start event reader goroutine
	b.readDone = make(chan struct{})
	b.shutdownOnce = sync.Once{}
	b.shutdownCh = make(chan struct{})
	b.readStarted.Store(true)
	go b.readEvents()

	return size, nil
}

// Restore restores the terminal to its original state.
// Shutdown ordering: flush output, stop signals, request read-loop exit,
// wait for read-loop to finish, then restore termios last.
func (b *ansiBackend) Restore() error {
	var firstErr error

	// 1. Flush any pending output.
	if err := b.w.Flush(); err != nil {
		firstErr = err
	}

	// 2. Stop signal delivery so the signal goroutine cannot write to the
	//    wake pipe after we close it. Stop() is idempotent via sync.Once.
	if b.signals != nil {
		b.signals.Stop()
	}

	// 3. Tell readEvents() that shutdown has started, so blocked event sends
	//    can abort instead of deadlocking Restore().
	b.shutdownOnce.Do(func() {
		if b.shutdownCh != nil {
			close(b.shutdownCh)
		}
	})

	// 4. Close write end of pipe to wake poll and request shutdown.
	if b.pipeW != nil {
		if err := b.pipeW.Close(); err != nil && firstErr == nil {
			firstErr = err
		}

		b.pipeW = nil
	}

	// 5. Wait for read loop to exit before restoring termios, so the
	//    read loop isn't running against a non-raw terminal.
	if b.readStarted.Load() && b.readDone != nil {
		<-b.readDone
	} else if b.pipeR != nil {
		// Read loop never started; no one else will close pipeR.
		if err := b.pipeR.Close(); err != nil && firstErr == nil {
			firstErr = err
		}

		b.pipeR = nil
	}

	// 6. Restore terminal settings last.
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
	b.sizeMu.RLock()
	defer b.sizeMu.RUnlock()
	return b.size
}

// sendEvent sends an event to the event channel, aborting if shutdown has started.
// Returns false if the send was aborted due to shutdown.
func (b *ansiBackend) sendEvent(ev event.Event) bool {
	select {
	case b.eventCh <- ev:
		return true
	case <-b.shutdownCh:
		return false
	}
}

// emitKeyEvents sends all key events, returning false if shutdown aborted a send.
func (b *ansiBackend) emitKeyEvents(evs []event.KeyEvent) bool {
	for _, ev := range evs {
		if !b.sendEvent(ev) {
			return false
		}
	}
	return true
}

// readEvents runs in a goroutine, reading input and publishing events.
func (b *ansiBackend) readEvents() {
	defer close(b.readDone)
	defer func() {
		// Resolve incomplete decoder state at EOF.
		var evs []event.KeyEvent
		evs = b.decoder.Finalize(evs)
		alive := b.emitKeyEvents(evs)
		if !alive {
			errors.Add(fmt.Errorf("emitKeyEvents already shutdown"))
		}

		// Stop resize handling if the read loop exits before Restore() runs.
		// Stop() is idempotent via sync.Once.
		if b.signals != nil {
			b.signals.Stop()
		}

		// readEvents owns pipeR: close it here after the poll/read loop exits,
		// so Restore() never closes an fd that may still be in poll().
		if b.pipeR != nil {
			errors.Add(b.pipeR.Close())
			b.pipeR = nil
		}

		// Close event channel to signal graceful shutdown.
		close(b.eventCh)
	}()

	// IMPORTANT: Read readiness checks and actual reads must operate on the same
	// raw file descriptor. Do not wrap b.r in a buffered reader: bytes buffered
	// in user-space would make inputReadable() lie, causing FlushPending() to
	// misclassify ESC/Alt-prefixed input as standalone ESC.
	fd := int(b.r.Fd())
	var pipeFd int
	if b.pipeR != nil {
		pipeFd = int(b.pipeR.Fd())
	}
	buf := make([]byte, 256)
	var pipeBuf [64]byte

	shutdownRequested := false

	for {
		pollFds := []unix.PollFd{
			{Fd: int32(fd), Events: unix.POLLIN},
		}
		if b.pipeR != nil {
			pollFds = append(pollFds, unix.PollFd{Fd: int32(pipeFd), Events: unix.POLLIN})
		}

		_, err := unix.Poll(pollFds, -1) // -1 = block indefinitely
		if err != nil {
			if err == unix.EINTR {
				continue
			}
			return
		}

		// Check for shutdown signal: pipe write end was closed.
		// Don't return immediately - fall through to drain any already-readable
		// stdin so we don't silently drop input that arrived at shutdown time.
		if b.pipeR != nil && len(pollFds) > 1 && pollFds[1].Revents&(unix.POLLHUP|unix.POLLERR) != 0 {
			shutdownRequested = true
		}

		// Drain pipe (resize notifications, non-blocking)
		if b.pipeR != nil && len(pollFds) > 1 && pollFds[1].Revents&unix.POLLIN != 0 {
			for {
				n, err := b.pipeR.Read(pipeBuf[:])
				errors.Add(err)
				if n <= 0 {
					break
				}
			}
		}

		// Process all pending resize events
		if b.signals != nil {
		drainResize:
			for {
				select {
				case size, ok := <-b.signals.ResizeChan():
					if !ok {
						break drainResize
					}
					b.sizeMu.Lock()
					b.size = size
					b.sizeMu.Unlock()
					if !b.sendEvent(event.ResizeEvent{W: size.W, H: size.H}) {
						return
					}
				default:
					break drainResize
				}
			}
		}

		// If stdin not readable and no hangup/error, nothing to process.
		if pollFds[0].Revents&(unix.POLLIN|unix.POLLHUP|unix.POLLERR) == 0 {
			if shutdownRequested {
				return
			}
			continue
		}

		// Drain all available input
		var evs []event.KeyEvent
		eof := false
		hupSeen := pollFds[0].Revents&(unix.POLLHUP|unix.POLLERR) != 0
		for inputReadable(fd) {
			n, err := b.r.Read(buf)
			if err != nil {
				// On PTY/tty hangup, Read returns EIO rather than io.EOF.
				// Treat any read error as terminal closure when POLLHUP/POLLERR
				// was set, to avoid busy-spinning on a hung-up fd.
				if err == io.EOF || hupSeen {
					eof = true
				}
				break
			}
			// After POLLHUP/POLLERR, treat n==0 as EOF (terminal closure)
			if n == 0 {
				if hupSeen {
					eof = true
				}
				break
			}

			for i := range n {
				evs = b.decoder.PushByte(evs, buf[i])
			}
		}

		// After draining all immediately-readable input, flush any pending
		// state (like standalone ESC) at this safe boundary.
		// This ensures prompt ESC delivery without timeout-based handling.
		if !inputReadable(fd) {
			evs = b.decoder.FlushPending(evs)
		}

		// Emit decoded events BEFORE returning on EOF or shutdown
		b.emitKeyEvents(evs)
		if eof || shutdownRequested {
			return
		}
	}
}

// inputReadable does a non-blocking poll check for POLLIN.
// Returns true if POLLIN is set, or if POLLHUP/POLLERR is set (indicating EOF).
// Retries on EINTR to avoid prematurely stopping the drain loop.
func inputReadable(fd int) bool {
	for {
		pollFds := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
		n, err := unix.Poll(pollFds, 0)
		if err == unix.EINTR {
			continue
		}
		if err != nil || n == 0 {
			return false
		}
		revents := pollFds[0].Revents
		return revents&unix.POLLIN != 0 || revents&(unix.POLLHUP|unix.POLLERR) != 0
	}
}
