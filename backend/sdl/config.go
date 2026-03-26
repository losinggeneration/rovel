package sdl

import "github.com/losinggeneration/tui/style"

// Options configures the future SDL backend.
type Options struct {
	Title string

	// Window size in pixels.
	WindowWidth  int
	WindowHeight int

	// Logical cell metrics in pixels.
	CellWidth  int
	CellHeight int

	// Font configuration for text rendering.
	FontPath string
	FontSize int

	// Surface default colors used when frame styles leave fg/bg as default.
	DefaultFG style.RGBA
	DefaultBG style.RGBA
}

// DefaultOptions returns a conservative default SDL configuration suitable for
// an initial cell-surface spike.
func DefaultOptions() Options {
	return Options{
		Title:        "tui",
		WindowWidth:  960,
		WindowHeight: 640,
		CellWidth:    8,
		CellHeight:   16,
		FontSize:     16,
		DefaultFG:    style.RGBA{R: 229, G: 229, B: 229, A: 0xFF},
		DefaultBG:    style.RGBA{R: 0, G: 0, B: 0, A: 0xFF},
	}
}
