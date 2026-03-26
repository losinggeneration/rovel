package sdl

import "testing"

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	if opts.WindowWidth <= 0 || opts.WindowHeight <= 0 {
		t.Fatalf("window size = %dx%d, want positive values", opts.WindowWidth, opts.WindowHeight)
	}

	if opts.CellWidth <= 0 || opts.CellHeight <= 0 {
		t.Fatalf("cell size = %dx%d, want positive values", opts.CellWidth, opts.CellHeight)
	}

	if opts.FontSize <= 0 {
		t.Fatalf("font size = %d, want positive value", opts.FontSize)
	}

	if opts.DefaultFG.A != 0xFF || opts.DefaultBG.A != 0xFF {
		t.Fatalf("default colors should be opaque: fg=%+v bg=%+v", opts.DefaultFG, opts.DefaultBG)
	}
}
