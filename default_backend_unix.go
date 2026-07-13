//go:build unix

package rovel

import (
	"fmt"
	"os"

	"github.com/losinggeneration/rovel/backend"
	"github.com/losinggeneration/rovel/backend/ansi"
	"github.com/losinggeneration/rovel/internal/errbuf"
)

// defaultBackend creates the default ANSI backend for Unix systems.
// When stdin or stdout is not a terminal (pipe, redirect), /dev/tty is
// opened and used instead so the TUI still drives the real terminal while
// the program's own stdout/stdin remain free for data I/O.
func defaultBackend(errs *errbuf.ErrorBuffer, appOpts AppOpts) (backend.Backend, error) {
	opts := ansi.Options{
		Mode:          appOpts.TerminalMode,
		HandleSignals: !appOpts.DisableSignalHandling,
	}

	if !ansi.IsTerminal(int(os.Stdin.Fd())) || !ansi.IsTerminal(int(os.Stdout.Fd())) {
		tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
		if err != nil {
			return nil, fmt.Errorf("opening /dev/tty: %w", err)
		}

		opts.Input = tty
		opts.Output = tty
		opts.OwnFiles = true // the backend closes the tty we opened, on Restore
	}

	return ansi.New(errs, opts)
}
