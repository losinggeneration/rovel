package render

import "github.com/losinggeneration/tui"

// zeroCell is the canonical empty cell.
var zeroCell = Cell{
	R:     ' ',
	Style: tui.Style{},
	Wide:  false,
}

// Cell represents a single cell in the terminal buffer.
type Cell struct {
	R        rune      // The rune (0 for continuation cells)
	Style    tui.Style // FG, BG, attributes
	Wide     bool      // True if this is the lead cell of a wide character
	WideCont bool      // True if this is a continuation cell
}

// IsZero returns true if the cell is the zero value.
func (c *Cell) IsZero() bool {
	return c.R == ' ' && c.Style == tui.Style{} && !c.Wide && !c.WideCont
}
