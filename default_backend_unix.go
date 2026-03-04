//go:build unix

package tui

import (
	"github.com/losinggeneration/tui/backend"
	"github.com/losinggeneration/tui/backend/ansi"
)

// defaultBackend creates the default ANSI backend for Unix systems.
func defaultBackend() (backend.Backend, error) {
	return ansi.New()
}
