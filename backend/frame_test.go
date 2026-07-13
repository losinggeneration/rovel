package backend

import (
	"testing"

	"github.com/losinggeneration/rovel/style"
)

func TestCellFrameAtAndRowRunes(t *testing.T) {
	frame := CellFrame{
		W: 3,
		H: 2,
		Cells: []FrameCell{
			{R: 'a'},
			{R: 'b'},
			{R: 'c'},
			{R: 'x'},
			{R: 'y'},
			{R: 'z'},
		},
	}

	if got := frame.At(1, 0).R; got != 'b' {
		t.Fatalf("At(1, 0) = %q, want %q", got, 'b')
	}

	if got := frame.At(99, 99).R; got != ' ' {
		t.Fatalf("At(99, 99) = %q, want zero cell", got)
	}

	if got := string(frame.RowRunes(1)); got != "xyz" {
		t.Fatalf("RowRunes(1) = %q, want %q", got, "xyz")
	}
}

func TestCellFrameGlyphRuns(t *testing.T) {
	s1 := style.Style{}.WithAttr(style.AttrBold)
	s2 := style.Style{}.WithAttr(style.AttrUnderline)

	frame := CellFrame{
		W: 6,
		H: 1,
		Cells: []FrameCell{
			{R: 'A', Style: s1},
			{R: 'B', Style: s1},
			{R: '界', Style: s2, Wide: true},
			{R: 0, Style: s2, WideCont: true},
			{R: 'C', Style: s2},
			{R: 'D', Style: s1},
		},
	}

	runs := frame.GlyphRuns(0)
	if len(runs) != 3 {
		t.Fatalf("GlyphRuns len = %d, want 3", len(runs))
	}

	if runs[0].X != 0 || runs[0].Text != "AB" || !runs[0].Style.Equals(s1) {
		t.Fatalf("run[0] = %+v, want X=0 Text=AB style=s1", runs[0])
	}

	if runs[1].X != 2 || runs[1].Text != "界C" || !runs[1].Style.Equals(s2) {
		t.Fatalf("run[1] = %+v, want X=2 Text=界C style=s2", runs[1])
	}

	if runs[2].X != 5 || runs[2].Text != "D" || !runs[2].Style.Equals(s1) {
		t.Fatalf("run[2] = %+v, want X=5 Text=D style=s1", runs[2])
	}
}
