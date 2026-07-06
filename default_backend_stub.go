//go:build !unix

package tui

import (
	"errors"

	"github.com/losinggeneration/tui/backend"
	"github.com/losinggeneration/tui/internal/errbuf"
)

// defaultBackend returns an error on non-Unix platforms.
// Custom backends must be provided via AppOpts.Backend.
func defaultBackend(_ *errbuf.ErrorBuffer, _ AppOpts) (backend.Backend, error) {
	return nil, errors.New("no default backend available on this platform; provide AppOpts.Backend")
}
