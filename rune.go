package rovel

import "github.com/losinggeneration/rovel/text"

// RuneWidth returns the cell width of a rune (0 for control, 1 for narrow, 2 for wide).
func RuneWidth(r rune) int {
	return text.WidthRune(r)
}
