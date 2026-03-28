package backend

import "github.com/losinggeneration/tui/style"

var zeroFrameCell = FrameCell{
	R:     ' ',
	Style: style.Style{},
}

// GlyphRun is a contiguous sequence of visible glyphs with the same style.
// Wide continuation cells are skipped; presenters can position the run at X and
// advance per glyph using cell widths.
type GlyphRun struct {
	X     int
	Text  string
	Style style.Style
}

// At returns the cell at x, y. Out-of-bounds coordinates return the zero cell.
func (f CellFrame) At(x, y int) FrameCell {
	if x < 0 || x >= f.W || y < 0 || y >= f.H {
		return zeroFrameCell
	}

	i := y*f.W + x
	if i < 0 || i >= len(f.Cells) {
		return zeroFrameCell
	}

	return f.Cells[i]
}

// RowRunes returns the runes for a row. Out-of-bounds rows return nil.
func (f CellFrame) RowRunes(y int) []rune {
	if y < 0 || y >= f.H {
		return nil
	}

	out := make([]rune, f.W)
	for x := range f.W {
		out[x] = f.At(x, y).R
	}

	return out
}

// GlyphRuns returns styled visible-glyph runs for a row. Wide continuation
// cells are omitted so consumers can draw by glyph rather than by raw cells.
func (f CellFrame) GlyphRuns(y int) []GlyphRun {
	if y < 0 || y >= f.H {
		return nil
	}

	var runs []GlyphRun

	for x := range f.W {
		cell := f.At(x, y)
		if cell.WideCont {
			continue
		}

		if len(runs) == 0 || !runs[len(runs)-1].Style.Equals(cell.Style) {
			runs = append(runs, GlyphRun{
				X:     x,
				Text:  string(cell.R),
				Style: cell.Style,
			})

			continue
		}

		runs[len(runs)-1].Text += string(cell.R)
	}

	return runs
}
