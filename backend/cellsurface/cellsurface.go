// Package cellsurface provides presentation helpers for windowed or embedded
// cell-grid backends.
//
// It consumes logical cell frames and resolves them into metrics and styled
// glyph runs that can be mapped onto a pixel surface.
//
// This package is under active development and its API is not yet stable.
//
// # Unstable API
//
// Before v0.1.0, the API may change without notice. Use with caution.
package cellsurface

import (
	"github.com/losinggeneration/tui/backend"
	"github.com/losinggeneration/tui/style"
)

// Metrics describes a logical-cell to pixel mapping.
type Metrics struct {
	CellWidth  int
	CellHeight int
}

// PixelSize is a concrete surface size in pixels.
type PixelSize struct {
	W int
	H int
}

// PixelRect is a concrete surface rect in pixels.
type PixelRect struct {
	X int
	Y int
	W int
	H int
}

// ResolvedGlyphRun is a surface-ready glyph run.
type ResolvedGlyphRun struct {
	X    int
	Y    int
	Text string
	FG   style.RGBA
	BG   style.RGBA
	Attr style.AttrMask
}

// FramePixelSize returns the pixel size required to display a frame.
func (m Metrics) FramePixelSize(frame backend.CellFrame) PixelSize {
	return PixelSize{
		W: frame.W * m.CellWidth,
		H: frame.H * m.CellHeight,
	}
}

// CellRect maps a logical cell position to a pixel rect.
func (m Metrics) CellRect(x, y int) PixelRect {
	return PixelRect{
		X: x * m.CellWidth,
		Y: y * m.CellHeight,
		W: m.CellWidth,
		H: m.CellHeight,
	}
}

// PixelToCell maps a pixel coordinate to a logical cell coordinate.
func (m Metrics) PixelToCell(x, y int) (cx, cy int) {
	if m.CellWidth > 0 {
		cx = x / m.CellWidth
	}

	if m.CellHeight > 0 {
		cy = y / m.CellHeight
	}

	return cx, cy
}

// ResolveGlyphRuns resolves a frame row into styled runs ready for surface
// presentation, including default-color resolution and reverse handling.
func ResolveGlyphRuns(frame backend.CellFrame, y int, base style.Style, defaultFG, defaultBG style.RGBA) []ResolvedGlyphRun {
	rawRuns := frame.GlyphRuns(y)
	out := make([]ResolvedGlyphRun, 0, len(rawRuns))

	for _, run := range rawRuns {
		fg, bg := run.Style.DisplayRGBA(base, defaultFG, defaultBG)
		resolved := style.Merge(base, run.Style)

		out = append(out, ResolvedGlyphRun{
			X:    run.X,
			Y:    y,
			Text: run.Text,
			FG:   fg,
			BG:   bg,
			Attr: resolved.Attr,
		})
	}

	return out
}
