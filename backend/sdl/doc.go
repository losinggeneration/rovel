// Package sdl will provide a windowed cell-surface backend built on SDL.
//
// The intended implementation model is:
//
//   - SDL owns window lifecycle and event polling
//   - tui still owns the retained widget/runtime model
//   - logical cell frames are consumed through backend.CellFrameSink
//   - backend/cellsurface provides the frame traversal and cell-to-pixel helpers
//
// If this backend is implemented, prefer github.com/veandco/go-sdl2 for now.
// The SDL3 Go bindings are still experimental and should not be the default
// target yet.
//
// This package is under active development and its API is not yet stable.
//
// # Unstable API
//
// Before v0.1.0, the API may change without notice. Use with caution.
package sdl
