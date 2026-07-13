//go:build !unix

package rovel

import (
	"errors"

	"github.com/losinggeneration/rovel/backend"
	"github.com/losinggeneration/rovel/internal/errbuf"
)

// defaultBackend returns an error on non-Unix platforms.
// Custom backends must be provided via AppOpts.Backend.
func defaultBackend(_ *errbuf.ErrorBuffer, _ AppOpts) (backend.Backend, error) {
	return nil, errors.New("no default backend available on this platform; provide AppOpts.Backend")
}
