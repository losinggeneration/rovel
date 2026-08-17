//go:build unix

package ansi

import (
	"os"
	"os/signal"
	"sync"

	"github.com/losinggeneration/rovel/backend"
	"github.com/losinggeneration/rovel/geom"
	"github.com/losinggeneration/rovel/internal/errbuf"
	"golang.org/x/sys/unix"
)

// signalHandler handles SIGWINCH for terminal resize events and, in cbreak
// mode, SIGINT for translating Ctrl+C into a synthetic KeyCtrlC event. When
// signal handling is enabled it also catches the terminal lifecycle signals
// SIGTSTP (suspend) and SIGTERM/SIGHUP (terminate), surfacing them to the app
// loop over lifecycleCh.
type signalHandler struct {
	resizeCh    chan geom.Size
	sigintCh    chan struct{}                // capacity 1; nil in raw mode
	lifecycleCh chan backend.LifecycleSignal // nil when signal handling disabled
	sigCh       chan os.Signal
	stopCh      chan struct{}
	once        sync.Once
	wakeFd      int // write end of wake pipe (raw fd)
	outFd       int // output fd for TIOCGWINSZ
	errs        *errbuf.ErrorBuffer
}

// setupSignalHandler sets up signal handlers for terminal events:
//   - SIGWINCH for resize events (all modes)
//   - SIGINT for Ctrl+C translation (cbreak mode only)
//   - SIGTSTP/SIGTERM/SIGHUP lifecycle signals (when handleSignals is set)
//
// It does not synthesize an initial ResizeEvent; the initial terminal size is
// obtained synchronously from Enable() / Size().
func setupSignalHandler(wakeFd int, outFd int, mode backend.TerminalMode, handleSignals bool, errs *errbuf.ErrorBuffer) (*signalHandler, error) {
	h := &signalHandler{
		resizeCh: make(chan geom.Size, 1),
		stopCh:   make(chan struct{}),
		wakeFd:   wakeFd,
		outFd:    outFd,
		errs:     errs,
	}

	// In cbreak mode, ISIG is on so Ctrl+C delivers SIGINT instead of
	// appearing as input. We catch it and translate to a synthetic event.
	if mode == backend.ModeCBreak {
		h.sigintCh = make(chan struct{}, 1)
	}

	// Start signal listener
	sigCh := make(chan os.Signal, 1)
	h.sigCh = sigCh
	signal.Notify(sigCh, unix.SIGWINCH)

	if mode == backend.ModeCBreak {
		signal.Notify(sigCh, unix.SIGINT)
	}

	// Terminal lifecycle signals are delivered straight to the app loop over
	// lifecycleCh (no self-pipe wake: the app loop selects on the channel
	// directly). The channel is buffered so a signal arriving mid-render is
	// held until the next loop iteration rather than dropped.
	if handleSignals {
		h.lifecycleCh = make(chan backend.LifecycleSignal, 1)
		signal.Notify(sigCh, unix.SIGTSTP, unix.SIGTERM, unix.SIGHUP)
	}

	go func() {
		for {
			select {
			case sig := <-sigCh:
				switch sig {
				case unix.SIGWINCH:
					if newSize, err := getTerminalSize(h.outFd); err == nil {
						select {
						case h.resizeCh <- newSize:
							h.wakePoll()
						default:
							// Channel full, drop old size
							select {
							case <-h.resizeCh:
								h.resizeCh <- newSize

								h.wakePoll()
							default:
							}
						}
					}
				case unix.SIGINT:
					if h.sigintCh != nil {
						select {
						case h.sigintCh <- struct{}{}:
						default:
						}

						h.wakePoll()
					}
				case unix.SIGTSTP:
					h.sendLifecycle(backend.SignalSuspend)
				case unix.SIGTERM, unix.SIGHUP:
					h.sendLifecycle(backend.SignalTerminate)

					// A second terminate signal must take the default
					// disposition and kill a hung shutdown; otherwise
					// converting SIGTERM to a graceful quit would leave
					// SIGKILL as the only way to stop a stuck app.
					signal.Reset(unix.SIGTERM, unix.SIGHUP)
				}
			case <-h.stopCh:
				signal.Stop(sigCh)
				close(sigCh)

				return
			}
		}
	}()

	return h, nil
}

// sendLifecycle delivers a lifecycle signal to the app loop, dropping it if the
// buffered channel is already full (the pending signal is equivalent). No
// terminal writes happen here — the app loop performs all teardown.
func (h *signalHandler) sendLifecycle(sig backend.LifecycleSignal) {
	if h.lifecycleCh == nil {
		return
	}

	select {
	case h.lifecycleCh <- sig:
	default:
	}
}

// LifecycleChan returns the lifecycle-signal channel, or nil when signal
// handling is disabled.
func (h *signalHandler) LifecycleChan() <-chan backend.LifecycleSignal {
	return h.lifecycleCh
}

// disarmSuspend removes the SIGTSTP handler so a SIGTSTP arriving while the
// process is stopped (e.g. an external kill -TSTP) takes the default
// disposition instead of being delivered to sigCh. The stop itself uses
// SIGSTOP (see Backend.Suspend), so this is hygiene for the stopped window,
// not a prerequisite for stopping.
func (h *signalHandler) disarmSuspend() {
	signal.Reset(unix.SIGTSTP)
}

// rearmSuspend re-registers SIGTSTP delivery after a resume. signal.Notify is
// goroutine-safe; the signal goroutine only reads sigCh.
func (h *signalHandler) rearmSuspend() {
	signal.Notify(h.sigCh, unix.SIGTSTP)
}

// ResizeChan returns the read-only channel for resize events.
func (h *signalHandler) ResizeChan() <-chan geom.Size {
	return h.resizeCh
}

// SigintChan returns the SIGINT notification channel, or nil in raw mode.
func (h *signalHandler) SigintChan() <-chan struct{} {
	return h.sigintCh
}

// Stop stops the signal handler.
//
// resizeCh is deliberately NOT closed: the signal goroutine may be mid-SIGWINCH
// (having already selected the resize case) and about to send to resizeCh when
// Stop runs, so closing it here would race into a send-on-closed-channel panic.
// The goroutine stops sending once it observes stopCh, and resizeCh is garbage
// collected with the handler. The reader tolerates a never-closed channel (it
// drains via a non-blocking select).
func (h *signalHandler) Stop() {
	h.once.Do(func() {
		close(h.stopCh)
	})
}

// wakePoll writes a byte to the wake pipe to unblock a blocking poll call.
// Errors are buffered in the package-level error buffer for later inspection.
func (h *signalHandler) wakePoll() {
	if h.wakeFd >= 0 {
		var b [1]byte

		b[0] = 1
		_, err := unix.Write(h.wakeFd, b[:])
		h.errs.Add(err)
	}
}
