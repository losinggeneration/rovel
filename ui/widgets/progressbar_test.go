package widgets

import (
	"testing"

	"github.com/losinggeneration/rovel"
	"github.com/losinggeneration/rovel/geom"
	"github.com/losinggeneration/rovel/render"
	"github.com/losinggeneration/rovel/style"
)

// paintBar paints one progress bar of the given width and value and
// returns the rendered row.
func paintBar(t *testing.T, width int, value float64) []rune {
	t.Helper()
	bar := NewProgressBarOpts(ProgressBarOpts{Value: value})
	bar.Layout(geom.Rect{X: 0, Y: 0, W: width, H: 1})
	base := style.Style{}
	buf := render.NewBuffer(width, 1)
	buf.Clear(render.Cell{R: ' ', Style: base})
	painter := render.NewPainter(buf, geom.Rect{X: 0, Y: 0, W: width, H: 1}, base)
	ctx := &rovel.Ctx{Theme: rovel.DefaultTheme()}
	bar.Paint(rovel.NewDrawer(rovel.NewPainter(painter, base)), ctx)
	row := make([]rune, width)
	for x := range width {
		row[x] = buf.At(x, 0).R
	}
	return row
}

func countFilled(row []rune) int {
	count := 0
	for _, r := range row {
		if r == '█' {
			count++
		}
	}
	return count
}

// TestProgressBarRoundsToNearestCell pins the quantization rule: the fill
// count rounds to the nearest cell, so a value a hair under complete (the
// sine gauges peak just shy of 1.0) paints the full bar instead of coming
// up one cell short at "100%".
func TestProgressBarRoundsToNearestCell(t *testing.T) {
	if got := countFilled(paintBar(t, 10, 0.999)); got != 10 {
		t.Fatalf("99.9%% of 10 cells filled %d, want 10 (rounds to nearest)", got)
	}
	if got := countFilled(paintBar(t, 10, 1.0)); got != 10 {
		t.Fatalf("100%% of 10 cells filled %d, want 10", got)
	}
	if got := countFilled(paintBar(t, 10, 0.0)); got != 0 {
		t.Fatalf("0%% of 10 cells filled %d, want 0", got)
	}
	if got := countFilled(paintBar(t, 10, 0.001)); got != 0 {
		t.Fatalf("0.1%% of 10 cells filled %d, want 0 (rounds to nearest)", got)
	}
	if got := countFilled(paintBar(t, 3, 0.5)); got != 2 {
		t.Fatalf("50%% of 3 cells filled %d, want 2 (rounds to nearest)", got)
	}
}
