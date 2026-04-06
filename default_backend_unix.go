//go:build unix

package tui

import (
	"github.com/losinggeneration/tui/backend"
	"github.com/losinggeneration/tui/backend/ansi"
	"github.com/losinggeneration/tui/internal/errbuf"
)

// defaultBackend creates the default ANSI backend for Unix systems.
func defaultBackend(errs *errbuf.ErrorBuffer) (backend.Backend, error) {
	return ansi.New(errs, ansi.Options{})
}
