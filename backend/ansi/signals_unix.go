//go:build unix

package ansi

import (
	"os"
	"os/signal"
	"sync"

	"github.com/losinggeneration/tui/geom"
	"github.com/losinggeneration/tui/internal/errbuf"
	"golang.org/x/sys/unix"
)

// signalHandler handles SIGWINCH for terminal resize events.
type signalHandler struct {
	resizeCh chan geom.Size
	stopCh   chan struct{}
	once     sync.Once
	wakeFd   int // write end of wake pipe (raw fd)
	errs     *errbuf.ErrorBuffer
}

// setupResizeHandler sets up a SIGWINCH signal handler for future resize events.
// It does not synthesize an initial ResizeEvent; the initial terminal size is
// obtained synchronously from Enable() / Size().
func setupResizeHandler(wakeFd int, errs *errbuf.ErrorBuffer) (*signalHandler, error) {
	h := &signalHandler{
		resizeCh: make(chan geom.Size, 1),
		stopCh:   make(chan struct{}),
		wakeFd:   wakeFd,
		errs:     errs,
	}

	// Start signal listener
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, unix.SIGWINCH)

	go func() {
		for {
			select {
			case <-sigCh:
				if newSize, err := getTerminalSize(); err == nil {
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
			case <-h.stopCh:
				signal.Stop(sigCh)
				close(sigCh)

				return
			}
		}
	}()

	return h, nil
}

// ResizeChan returns the read-only channel for resize events.
func (h *signalHandler) ResizeChan() <-chan geom.Size {
	return h.resizeCh
}

// Stop stops the signal handler.
func (h *signalHandler) Stop() {
	h.once.Do(func() {
		close(h.stopCh)
		close(h.resizeCh)
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
