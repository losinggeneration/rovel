//go:build unix

package ansi

import (
	"os"
	"os/signal"
	"sync"

	"github.com/losinggeneration/tui/geom"
	"golang.org/x/sys/unix"
)

// signalHandler handles SIGWINCH for terminal resize events.
type signalHandler struct {
	resizeCh chan geom.Size
	stopCh   chan struct{}
	once     sync.Once
}

// setupResizeHandler sets up a SIGWINCH signal handler.
// It returns an initial size event on the channel.
func setupResizeHandler() (*signalHandler, error) {
	size, err := getTerminalSize()
	if err != nil {
		return nil, err
	}

	h := &signalHandler{
		resizeCh: make(chan geom.Size, 1),
		stopCh:   make(chan struct{}),
	}

	// Send initial size
	h.resizeCh <- size

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
					default:
						// Channel full, drop old size
						select {
						case <-h.resizeCh:
							h.resizeCh <- newSize
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
