package tui

import "github.com/losinggeneration/tui/render"

// RuneWidth returns the cell width of a rune (1 for narrow, 2 for wide).
func RuneWidth(r rune) int {
	return render.RuneWidth(r)
}
