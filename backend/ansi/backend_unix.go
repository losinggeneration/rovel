//go:build unix

package ansi

import (
	"bufio"
	"errors"
	"io"
	"os"
	"sync"
	"sync/atomic"

	"github.com/losinggeneration/tui/event"
	"github.com/losinggeneration/tui/geom"
	"github.com/losinggeneration/tui/internal/errbuf"
	"golang.org/x/sys/unix"
)

var ErrEmitKeyEventsShutdown = errors.New("emitKeyEvents already shutdown")

// Backend implements the backend.Backend interface for Unix terminals using ANSI escape sequences.
type Backend struct {
	origTermios *unix.Termios
	r           *os.File
	w           *bufio.Writer
	decoder     InputDecoder
	inputFeats  inputFeatures
	signals     *signalHandler
	eventCh     chan event.Event
	errs        *errbuf.ErrorBuffer

	// Self-pipe for waking poll on resize/shutdown (raw fds, -1 = unset)
	pipeR int // read end; owned/closed by readEvents()
	pipeW int // write end; owned/closed by Restore()

	// Read-loop lifecycle
	readStarted  atomic.Bool
	readDone     chan struct{}
	shutdownCh   chan struct{}
	shutdownOnce sync.Once

	// Terminal size, protected by sizeMu
	sizeMu sync.RWMutex
	size   geom.Size
}

// New creates a new ANSI backend. The provided ErrorBuffer is used to record
// non-fatal errors from signal handling and cleanup operations.
func New(errs *errbuf.ErrorBuffer) (*Backend, error) {
	w := bufio.NewWriterSize(os.Stdout, 64*1024)

	size, err := getTerminalSize()
	if err != nil {
		return nil, err
	}

	b := &Backend{
		r:       os.Stdin,
		w:       w,
		size:    size,
		eventCh: make(chan event.Event, 8),
		errs:    errs,
		pipeR:   -1,
		pipeW:   -1,
	}

	return b, nil
}

// Enable enables the terminal and returns the initial size.
func (b *Backend) Enable() (geom.Size, error) {
	orig, err := enableRaw()
	if err != nil {
		return geom.Size{}, err
	}

	b.origTermios = orig

	// Create self-pipe for waking poll using raw fds (no Go runtime involvement)
	var pipeFds [2]int
	if err := unix.Pipe2(pipeFds[:], unix.O_NONBLOCK|unix.O_CLOEXEC); err != nil {
		b.errs.Add(restore(orig))

		return geom.Size{}, err
	}

	b.pipeR = pipeFds[0]
	b.pipeW = pipeFds[1]

	// Setup signal handler for resize events
	signals, err := setupResizeHandler(b.pipeW, b.errs)
	if err != nil {
		b.errs.Add(unix.Close(b.pipeR))
		b.errs.Add(unix.Close(b.pipeW))
		b.pipeR = -1
		b.pipeW = -1
		b.errs.Add(restore(orig))

		return geom.Size{}, err
	}

	b.signals = signals

	// Get initial size before starting background goroutines.
	size, err := getTerminalSize()
	if err != nil {
		b.signals.Stop()
		b.signals = nil

		b.errs.Add(unix.Close(b.pipeR))
		b.errs.Add(unix.Close(b.pipeW))
		b.pipeR = -1
		b.pipeW = -1

		b.errs.Add(restore(orig))
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
func (b *Backend) Restore() error {
	var firstErr error

	// 1. Disable any enabled input features before flushing.
	err := b.disableInputFeatures()
	if err != nil && firstErr == nil {
		firstErr = err
	}

	// 2. Flush any pending output.
	err = b.w.Flush()
	if err != nil && firstErr == nil {
		firstErr = err
	}

	// 3. Stop signal delivery so the signal goroutine cannot write to the
	//    wake pipe after we close it. Stop() is idempotent via sync.Once.
	if b.signals != nil {
		b.signals.Stop()
	}

	// 4. Tell readEvents() that shutdown has started, so blocked event sends
	//    can abort instead of deadlocking Restore().
	b.shutdownOnce.Do(func() {
		if b.shutdownCh != nil {
			close(b.shutdownCh)
		}
	})

	// 5. Close write end of pipe to wake poll and request shutdown.
	if b.pipeW >= 0 {
		err := unix.Close(b.pipeW)
		if err != nil && firstErr == nil {
			firstErr = err
		}

		b.pipeW = -1
	}

	// 6. Wait for read loop to exit before restoring termios, so the
	//    read loop isn't running against a non-raw terminal.
	if b.readStarted.Load() && b.readDone != nil {
		<-b.readDone
	} else if b.pipeR >= 0 {
		// Read loop never started; no one else will close pipeR.
		err := unix.Close(b.pipeR)
		if err != nil && firstErr == nil {
			firstErr = err
		}

		b.pipeR = -1
	}

	// 7. Restore terminal settings last.
	if b.origTermios != nil {
		err := restore(b.origTermios)
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

// Write writes raw ANSI output to the terminal.
func (b *Backend) Write(p []byte) (int, error) {
	return b.w.Write(p)
}

// Flush flushes any buffered output.
func (b *Backend) Flush() error {
	return b.w.Flush()
}

// ReadEvent reads and returns the next event, blocking until one is available.
func (b *Backend) ReadEvent() event.Event {
	return <-b.eventCh
}

// Size returns the current terminal size.
func (b *Backend) Size() geom.Size {
	b.sizeMu.RLock()
	defer b.sizeMu.RUnlock()

	return b.size
}

// sendEvent sends an event to the event channel, aborting if shutdown has started.
// Returns false if the send was aborted due to shutdown.
func (b *Backend) sendEvent(ev event.Event) bool {
	select {
	case b.eventCh <- ev:
		return true
	case <-b.shutdownCh:
		return false
	}
}

// emitEvents sends all events, returning false if shutdown aborted a send.
func (b *Backend) emitEvents(evs []event.Event) bool {
	for _, ev := range evs {
		if !b.sendEvent(ev) {
			return false
		}
	}

	return true
}

// readEvents runs in a goroutine, reading input and publishing events.
func (b *Backend) readEvents() {
	defer close(b.readDone)
	defer func() {
		// Resolve incomplete decoder state at EOF.
		var evs []event.Event

		evs = b.decoder.Finalize(evs)

		alive := b.emitEvents(evs)
		if !alive {
			b.errs.Add(ErrEmitKeyEventsShutdown)
		}

		// Stop resize handling if the read loop exits before Restore() runs.
		// Stop() is idempotent via sync.Once.
		if b.signals != nil {
			b.signals.Stop()
		}

		// readEvents owns pipeR: close it here after the poll/read loop exits,
		// so Restore() never closes an fd that may still be in poll().
		if b.pipeR >= 0 {
			b.errs.Add(unix.Close(b.pipeR))
			b.pipeR = -1
		}

		// Close event channel to signal graceful shutdown.
		close(b.eventCh)
	}()

	// IMPORTANT: Read readiness checks and actual reads must operate on the same
	// raw file descriptor. Do not wrap b.r in a buffered reader: bytes buffered
	// in user-space would make inputReadable() lie, causing FlushPending() to
	// misclassify ESC/Alt-prefixed input as standalone ESC.
	fd := int(b.r.Fd())
	pipeFd := b.pipeR
	buf := make([]byte, 256)

	var pipeBuf [64]byte

	shutdownRequested := false

	for {
		pollFds := []unix.PollFd{
			{Fd: int32(fd), Events: unix.POLLIN},
		}
		if pipeFd >= 0 {
			pollFds = append(pollFds, unix.PollFd{Fd: int32(pipeFd), Events: unix.POLLIN})
		}

		_, err := unix.Poll(pollFds, -1) // -1 = block indefinitely
		if err != nil {
			if errors.Is(err, unix.EINTR) {
				continue
			}

			return
		}

		// Check for shutdown signal: pipe write end was closed.
		// Don't return immediately - fall through to drain any already-readable
		// stdin so we don't silently drop input that arrived at shutdown time.
		if pipeFd >= 0 && len(pollFds) > 1 && pollFds[1].Revents&(unix.POLLHUP|unix.POLLERR) != 0 {
			shutdownRequested = true
		}

		// Drain pipe (resize notifications, non-blocking)
		if pipeFd >= 0 && len(pollFds) > 1 && pollFds[1].Revents&unix.POLLIN != 0 {
			for {
				n, err := unix.Read(pipeFd, pipeBuf[:])
				if n <= 0 || err != nil {
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
		var evs []event.Event

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
		b.emitEvents(evs)

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
		if errors.Is(err, unix.EINTR) {
			continue
		}

		if err != nil || n == 0 {
			return false
		}

		revents := pollFds[0].Revents

		return revents&unix.POLLIN != 0 || revents&(unix.POLLHUP|unix.POLLERR) != 0
	}
}
