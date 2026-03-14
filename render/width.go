package render

import "github.com/losinggeneration/tui/internal/width"

// RuneWidth returns the terminal column width of a rune (0, 1, or 2).
//
// Public API note: package text is the public semantic source of truth for text
// measurement; this is a low-level wrapper used by render and friends.
func RuneWidth(r rune) int {
	return width.RuneWidth(r)
}
