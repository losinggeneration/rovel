package cellsurface

import (
	"testing"

	"github.com/losinggeneration/tui/backend"
	"github.com/losinggeneration/tui/style"
)

func TestMetricsMapping(t *testing.T) {
	m := Metrics{CellWidth: 8, CellHeight: 16}
	frame := backend.CellFrame{W: 10, H: 4}

	if got := m.FramePixelSize(frame); got != (PixelSize{W: 80, H: 64}) {
		t.Fatalf("FramePixelSize = %+v, want {80 64}", got)
	}

	if got := m.CellRect(2, 3); got != (PixelRect{X: 16, Y: 48, W: 8, H: 16}) {
		t.Fatalf("CellRect = %+v, want {16 48 8 16}", got)
	}

	cx, cy := m.PixelToCell(23, 35)
	if cx != 2 || cy != 2 {
		t.Fatalf("PixelToCell = %d,%d, want 2,2", cx, cy)
	}
}

func TestResolveGlyphRuns(t *testing.T) {
	base := style.Style{
		FG: style.ColorWhite,
		BG: style.ColorBlue,
	}
	defaultFG := style.RGBA{R: 1, G: 1, B: 1, A: 0xFF}
	defaultBG := style.RGBA{R: 2, G: 2, B: 2, A: 0xFF}

	frame := backend.CellFrame{
		W: 4,
		H: 1,
		Cells: []backend.FrameCell{
			{R: 'A', Style: style.Style{FG: style.ColorRed}},
			{R: 'B', Style: style.Style{FG: style.ColorRed}},
			{R: 'C', Style: style.Style{Attr: style.AttrReverse}},
			{R: 'D', Style: style.Style{FG: style.ColorGreen}},
		},
	}

	runs := ResolveGlyphRuns(frame, 0, base, defaultFG, defaultBG)
	if len(runs) != 3 {
		t.Fatalf("ResolveGlyphRuns len = %d, want 3", len(runs))
	}

	if runs[0].X != 0 || runs[0].Text != "AB" {
		t.Fatalf("run[0] = %+v, want X=0 Text=AB", runs[0])
	}

	if runs[1].X != 2 || runs[1].Text != "C" {
		t.Fatalf("run[1] = %+v, want X=2 Text=C", runs[1])
	}

	wantFG := base.BG.DisplayRGBA(defaultFG)

	wantBG := base.FG.DisplayRGBA(defaultBG)
	if runs[1].FG != wantFG || runs[1].BG != wantBG {
		t.Fatalf("reversed run colors = %+v/%+v, want %+v/%+v", runs[1].FG, runs[1].BG, wantFG, wantBG)
	}

	if runs[2].X != 3 || runs[2].Text != "D" {
		t.Fatalf("run[2] = %+v, want X=3 Text=D", runs[2])
	}
}
