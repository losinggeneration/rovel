package sdl

import "github.com/losinggeneration/tui/backend"

// New creates a windowed SDL backend.
func New(opts Options) (backend.Backend, error) {
	return newBackend(opts)
}
