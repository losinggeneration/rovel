package render

import "github.com/losinggeneration/tui"

// ClipRect returns the intersection of a drawing rect with a clip bounds.
func ClipRect(draw, clip tui.Rect) tui.Rect {
	return draw.Intersect(clip)
}

// IsClipped returns true if a point is outside clip bounds.
func IsClipped(x, y int, clip tui.Rect) bool {
	return x < clip.X || x >= clip.X+clip.W ||
		y < clip.Y || y >= clip.Y+clip.H
}
