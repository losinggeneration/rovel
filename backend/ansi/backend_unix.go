//go:build unix

package ansi

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"

	"github.com/losinggeneration/tui/backend"
	"github.com/losinggeneration/tui/event"
	"github.com/losinggeneration/tui/geom"
	"github.com/losinggeneration/tui/internal/errbuf"
	"golang.org/x/sys/unix"
)

// errKeyEventsAfterShutdown is an internal diagnostic recorded when key event
// emission is attempted after the backend has begun shutting down.
var errKeyEventsAfterShutdown = errors.New("ansi: key event emitted after shutdown")

// ErrInputNotTerminal is returned when the input file descriptor is not a terminal.
var ErrInputNotTerminal = errors.New("input fd is not a terminal")

// ErrOutputNotTerminal is returned when the output file descriptor is not a terminal.
var ErrOutputNotTerminal = errors.New("output fd is not a terminal")

// Options configures the ANSI backend.
// The zero value is valid and uses os.Stdin for input and os.Stdout for output.
type Options struct {
	// Input overrides the file used for reading terminal input.
	// If nil, os.Stdin is used. The file must be a terminal (tty).
	Input *os.File

	// Output overrides the file used for writing terminal output.
	// If nil, os.Stdout is used. The file must be a terminal (tty).
	Output *os.File

	// Mode selects the terminal input mode. The zero value (ModeRaw) puts the
	// terminal into full raw mode for interactive TUI apps. ModeCBreak puts
	// the terminal into cbreak mode (character-at-a-time input with ECHO
	// disabled but ISIG/OPOST intact), suitable for interactive CLI tools
	// like dialog boxes, prompts, and script-driven TUI components that draw
	// inline without clearing the screen.
	Mode backend.TerminalMode

	// HandleSignals enables built-in handling of the terminal lifecycle
	// signals SIGTSTP (suspend/resume) and SIGTERM/SIGHUP (surfaced as
	// SignalTerminate). When set, the backend catches these signals and
	// exposes them via the SignalController interface (Signals/Suspend) for
	// the app loop to orchestrate. SIGWINCH and (in cbreak mode) SIGINT are
	// always handled regardless of this flag.
	HandleSignals bool

	// OwnFiles indicates that the backend owns Input and Output and must close
	// them on Restore. Set this when the caller opened the files itself (for
	// example /dev/tty). Leave it false when passing os.Stdin/os.Stdout or
	// files whose lifetime the caller manages.
	OwnFiles bool
}

// Backend implements the backend.Backend interface for Unix terminals using ANSI escape sequences.
type Backend struct {
	mode          backend.TerminalMode
	handleSignals bool
	origTermios   *unix.Termios
	r             *os.File
	out           *os.File // underlying output file (for fd-based operations)
	w             *bufio.Writer
	decoder       InputDecoder
	inputFeats    inputFeatures
	signals       *signalHandler
	eventCh       chan event.Event
	errs          *errbuf.ErrorBuffer

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

	// ownedFiles are files the backend must close on Restore (deduplicated).
	// Populated from Options.OwnFiles; nil when the caller owns the files.
	ownedFiles []*os.File
}

// New creates a new ANSI backend. The provided ErrorBuffer is used to record
// non-fatal errors from signal handling and cleanup operations.
// opts allows overriding the input/output files; zero value uses os.Stdin/os.Stdout.
func New(errs *errbuf.ErrorBuffer, opts Options) (*Backend, error) {
	in := opts.Input
	if in == nil {
		in = os.Stdin
	}

	out := opts.Output
	if out == nil {
		out = os.Stdout
	}

	inFd := int(in.Fd())
	outFd := int(out.Fd())

	if !isTerminal(inFd) {
		return nil, fmt.Errorf("ansi input fd %d is not a terminal: %w", inFd, ErrInputNotTerminal)
	}

	if !isTerminal(outFd) {
		return nil, fmt.Errorf("ansi output fd %d is not a terminal: %w", outFd, ErrOutputNotTerminal)
	}

	w := bufio.NewWriterSize(out, 64*1024)

	size, err := getTerminalSize(outFd)
	if err != nil {
		return nil, fmt.Errorf("getTerminalSize: %w", err)
	}

	b := &Backend{
		mode:          opts.Mode,
		handleSignals: opts.HandleSignals,
		r:             in,
		out:           out,
		w:             w,
		size:          size,
		eventCh:       make(chan event.Event, 8),
		errs:          errs,
		pipeR:         -1,
		pipeW:         -1,
	}

	if opts.OwnFiles {
		b.ownedFiles = append(b.ownedFiles, in)
		if out != in {
			b.ownedFiles = append(b.ownedFiles, out)
		}
	}

	return b, nil
}

// Mode returns the terminal mode this backend was configured with.
func (b *Backend) Mode() backend.TerminalMode {
	return b.mode
}

// Enable enables the terminal and returns the initial size.
func (b *Backend) Enable() (geom.Size, error) {
	var orig *unix.Termios
	var err error

	switch b.mode {
	case backend.ModeCBreak:
		orig, err = enableCBreak(int(b.r.Fd()))
		if err != nil {
			return geom.Size{}, fmt.Errorf("enableCBreak: %w", err)
		}
	default:
		orig, err = enableRaw(int(b.r.Fd()))
		if err != nil {
			return geom.Size{}, fmt.Errorf("enableRaw: %w", err)
		}
	}

	b.origTermios = orig

	// Create self-pipe for waking poll using raw fds (no Go runtime involvement)
	var pipeFds [2]int
	if err := newWakePipe(pipeFds[:]); err != nil {
		b.errs.Add(restore(int(b.r.Fd()), orig))

		return geom.Size{}, err
	}

	b.pipeR = pipeFds[0]
	b.pipeW = pipeFds[1]

	// Setup signal handler for resize events (and SIGINT in cooked mode)
	signals, err := setupSignalHandler(b.pipeW, int(b.out.Fd()), b.mode, b.handleSignals, b.errs)
	if err != nil {
		b.errs.Add(unix.Close(b.pipeR))
		b.errs.Add(unix.Close(b.pipeW))
		b.pipeR = -1
		b.pipeW = -1
		b.errs.Add(restore(int(b.r.Fd()), orig))

		return geom.Size{}, err
	}

	b.signals = signals

	// Get initial size before starting background goroutines.
	size, err := getTerminalSize(int(b.out.Fd()))
	if err != nil {
		b.signals.Stop()
		b.signals = nil

		b.errs.Add(unix.Close(b.pipeR))
		b.errs.Add(unix.Close(b.pipeW))
		b.pipeR = -1
		b.pipeW = -1

		b.errs.Add(restore(int(b.r.Fd()), orig))
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
	if err != nil {
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

	// 7. Restore terminal settings last (uses the fd, so before closing files).
	if b.origTermios != nil {
		err := restore(int(b.r.Fd()), b.origTermios)
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	// 8. Close any files the backend owns (e.g. an opened /dev/tty). Cleared
	//    afterward so a second Restore does not double-close.
	for _, f := range b.ownedFiles {
		err := f.Close()
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	b.ownedFiles = nil

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

// Signals returns the lifecycle-signal channel, or nil when signal handling is
// disabled (Options.HandleSignals is false) or the backend is not enabled. It
// implements backend.SignalController.
func (b *Backend) Signals() <-chan backend.LifecycleSignal {
	if b.signals == nil {
		return nil
	}

	return b.signals.LifecycleChan()
}

// Suspend restores cooked terminal state, stops the process via SIGTSTP, and on
// resume re-enters raw/cbreak mode and refreshes the cached terminal size. It
// blocks while the process is stopped. It implements backend.SignalController.
//
// The size re-query on resume is deliberate: the terminal is commonly resized
// while the process is stopped, and a pending SIGWINCH delivered on continue
// would otherwise race the app loop's resume repaint.
func (b *Backend) Suspend() error {
	fd := int(b.r.Fd())

	// Restore cooked terminal state before stopping so the shell behaves
	// normally while the process is suspended.
	if err := restore(fd, b.origTermios); err != nil {
		return fmt.Errorf("restore termios for suspend: %w", err)
	}

	// Remove our SIGTSTP handler (if armed) so the re-raised signal takes the
	// default disposition — actually stopping the process — instead of being
	// delivered back to the signal goroutine.
	if b.handleSignals && b.signals != nil {
		b.signals.disarmSuspend()
	}

	// Stop the process. Execution blocks here until SIGCONT (fg) resumes it.
	if err := unix.Kill(unix.Getpid(), unix.SIGTSTP); err != nil {
		// Best-effort recovery: re-arm and re-enter raw mode so the terminal
		// is not left cooked.
		if b.handleSignals && b.signals != nil {
			b.signals.rearmSuspend()
		}

		b.errs.Add(b.reenterRawMode(fd))

		return fmt.Errorf("kill SIGTSTP: %w", err)
	}

	// --- resumed here on SIGCONT ---

	if b.handleSignals && b.signals != nil {
		b.signals.rearmSuspend()
	}

	if err := b.reenterRawMode(fd); err != nil {
		return err
	}

	// Refresh the cached size; the terminal may have been resized while stopped.
	size, err := getTerminalSize(int(b.out.Fd()))
	if err != nil {
		return fmt.Errorf("getTerminalSize after resume: %w", err)
	}

	b.sizeMu.Lock()
	b.size = size
	b.sizeMu.Unlock()

	return nil
}

// reenterRawMode re-applies the configured terminal mode after a resume. The
// original termios (captured at Enable) is preserved; the state returned by
// enableRaw/enableCBreak here is the cooked state and is discarded.
func (b *Backend) reenterRawMode(fd int) error {
	var err error

	switch b.mode {
	case backend.ModeCBreak:
		_, err = enableCBreak(fd)
		if err != nil {
			return fmt.Errorf("enableCBreak after resume: %w", err)
		}
	default:
		_, err = enableRaw(fd)
		if err != nil {
			return fmt.Errorf("enableRaw after resume: %w", err)
		}
	}

	return nil
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
			b.errs.Add(errKeyEventsAfterShutdown)
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

			// In cooked mode, check for SIGINT and emit a synthetic KeyCtrlC.
			if sigintCh := b.signals.SigintChan(); sigintCh != nil {
			drainSigint:
				for {
					select {
					case <-sigintCh:
						if !b.sendEvent(event.KeyEvent{Key: event.KeyCtrlC}) {
							return
						}
					default:
						break drainSigint
					}
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
