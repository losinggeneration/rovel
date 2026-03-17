package render

import (
	"testing"

	"github.com/losinggeneration/tui/geom"
	"github.com/losinggeneration/tui/style"
)

func TestDiffRuns_EmptyDamage(t *testing.T) {
	back := NewBuffer(10, 5)
	front := NewBuffer(10, 5)
	dmg := NewDamage(10, 5)

	runs := DiffRuns(back, front, dmg)

	if runs != nil {
		t.Errorf("DiffRuns() with empty damage = %v, want nil", runs)
	}
}

func TestDiffRuns_NoChanges(t *testing.T) {
	back := NewBuffer(10, 5)
	front := NewBuffer(10, 5)
	dmg := NewDamage(10, 5)

	// Set same content in both buffers
	for y := range 5 {
		for x := range 10 {
			*back.At(x, y) = Cell{R: 'A', Style: style.Style{FG: style.ColorRed}}
			*front.At(x, y) = Cell{R: 'A', Style: style.Style{FG: style.ColorRed}}
		}
	}

	dmg.AddRect(geom.Rect{X: 0, Y: 0, W: 10, H: 5})

	runs := DiffRuns(back, front, dmg)

	if len(runs) != 0 {
		t.Errorf("DiffRuns() with identical buffers = %d runs, want 0", len(runs))
	}
}

func TestDiffRuns_NilBuffers(t *testing.T) {
	dmg := NewDamage(10, 5)
	dmg.AddRect(geom.Rect{X: 0, Y: 0, W: 10, H: 5})

	if runs := DiffRuns(nil, NewBuffer(10, 5), dmg); runs != nil {
		t.Errorf("DiffRuns() with nil back = %v, want nil", runs)
	}

	if runs := DiffRuns(NewBuffer(10, 5), nil, dmg); runs != nil {
		t.Errorf("DiffRuns() with nil front = %v, want nil", runs)
	}
}

func TestDiffRuns_SizeMismatch(t *testing.T) {
	back := NewBuffer(10, 5)
	front := NewBuffer(8, 5)
	dmg := NewDamage(10, 5)
	dmg.AddRect(geom.Rect{X: 0, Y: 0, W: 10, H: 5})

	runs := DiffRuns(back, front, dmg)

	if runs != nil {
		t.Errorf("DiffRuns() with mismatched buffer sizes = %v, want nil", runs)
	}
}

func TestDiffRuns_SingleCellChange(t *testing.T) {
	back := NewBuffer(10, 5)
	front := NewBuffer(10, 5)
	dmg := NewDamage(10, 5)

	*back.At(5, 2) = Cell{R: 'X', Style: style.Style{FG: style.ColorBlue}}
	*front.At(5, 2) = Cell{R: 'A', Style: style.Style{FG: style.ColorRed}}

	dmg.AddSpan(2, 5, 6)

	runs := DiffRuns(back, front, dmg)

	if len(runs) != 1 {
		t.Fatalf("DiffRuns() returned %d runs, want 1", len(runs))
	}

	if runs[0].Y != 2 || runs[0].X0 != 5 || runs[0].X1 != 6 {
		t.Errorf("DiffRuns() = %+v, want {Y:2, X0:5, X1:6}", runs[0])
	}
}

func TestDiffRuns_ContiguousChangesMerged(t *testing.T) {
	back := NewBuffer(10, 5)
	front := NewBuffer(10, 5)
	dmg := NewDamage(10, 5)

	// Change cells 3,4,5 in row 2
	for x := 3; x <= 5; x++ {
		*back.At(x, 2) = Cell{R: 'X', Style: style.Style{FG: style.ColorBlue}}
		*front.At(x, 2) = Cell{R: 'A', Style: style.Style{FG: style.ColorRed}}
	}

	dmg.AddSpan(2, 3, 6)

	runs := DiffRuns(back, front, dmg)

	if len(runs) != 1 {
		t.Fatalf("DiffRuns() returned %d runs, want 1", len(runs))
	}

	if runs[0].Y != 2 || runs[0].X0 != 3 || runs[0].X1 != 6 {
		t.Errorf("DiffRuns() = %+v, want {Y:2, X0:3, X1:6}", runs[0])
	}
}

func TestDiffRuns_NonContiguousChanges(t *testing.T) {
	back := NewBuffer(10, 5)
	front := NewBuffer(10, 5)
	dmg := NewDamage(10, 5)

	// Change cells 2 and 7 in row 3
	*back.At(2, 3) = Cell{R: 'X'}
	*front.At(2, 3) = Cell{R: 'A'}
	*back.At(7, 3) = Cell{R: 'Y'}
	*front.At(7, 3) = Cell{R: 'B'}

	dmg.AddSpan(3, 0, 10)

	runs := DiffRuns(back, front, dmg)

	if len(runs) != 2 {
		t.Fatalf("DiffRuns() returned %d runs, want 2", len(runs))
	}
}

func TestDiffRuns_MultipleRows(t *testing.T) {
	back := NewBuffer(10, 5)
	front := NewBuffer(10, 5)
	dmg := NewDamage(10, 5)

	// Change cell in row 1
	*back.At(3, 1) = Cell{R: 'X'}
	*front.At(3, 1) = Cell{R: 'A'}
	// Change cell in row 3
	*back.At(5, 3) = Cell{R: 'Y'}
	*front.At(5, 3) = Cell{R: 'B'}

	dmg.AddRect(geom.Rect{X: 0, Y: 0, W: 10, H: 5})

	runs := DiffRuns(back, front, dmg)

	if len(runs) != 2 {
		t.Fatalf("DiffRuns() returned %d runs, want 2", len(runs))
	}
}

func TestDiffRuns_WideCharExpansion(t *testing.T) {
	back := NewBuffer(10, 5)
	front := NewBuffer(10, 5)
	dmg := NewDamage(10, 5)

	// Wide char at position 3 (occupies 3 and 4)
	*back.At(3, 2) = Cell{R: '日', Wide: true}
	*back.At(4, 2) = Cell{R: 0, WideCont: true}
	*front.At(3, 2) = Cell{R: 'A'}
	*front.At(4, 2) = Cell{R: ' '}

	dmg.AddSpan(2, 3, 5)

	runs := DiffRuns(back, front, dmg)

	if len(runs) != 1 {
		t.Fatalf("DiffRuns() returned %d runs, want 1", len(runs))
	}
	// Should include both cells (wide lead + continuation)
	if runs[0].X0 != 3 || runs[0].X1 != 5 {
		t.Errorf("DiffRuns() = %+v, want {X0:3, X1:5}", runs[0])
	}
}

func TestDiffRuns_WideCharBoundary(t *testing.T) {
	back := NewBuffer(10, 5)
	front := NewBuffer(10, 5)
	dmg := NewDamage(10, 5)

	// Wide char at position 2 (occupies 2 and 3)
	*back.At(2, 2) = Cell{R: '日', Wide: true}
	*back.At(3, 2) = Cell{R: 0, WideCont: true}
	*front.At(2, 2) = Cell{R: 'A'}
	*front.At(3, 2) = Cell{R: ' '}
	// Change only position 3
	dmg.AddSpan(2, 3, 4)

	runs := DiffRuns(back, front, dmg)

	// Should expand to include the lead cell
	if len(runs) != 1 {
		t.Fatalf("DiffRuns() returned %d runs, want 1", len(runs))
	}

	if runs[0].X0 != 2 || runs[0].X1 != 4 {
		t.Errorf("DiffRuns() = %+v, want {X0:2, X1:4}", runs[0])
	}
}

func TestDiffRuns_NonOverlappingSpans(t *testing.T) {
	back := NewBuffer(10, 5)
	front := NewBuffer(10, 5)
	dmg := NewDamage(10, 5)

	// Change in two separate damaged spans
	*back.At(2, 2) = Cell{R: 'X'}
	*front.At(2, 2) = Cell{R: 'A'}
	*back.At(7, 2) = Cell{R: 'Y'}
	*front.At(7, 2) = Cell{R: 'B'}

	dmg.AddSpan(2, 0, 4)
	dmg.AddSpan(2, 6, 9)

	runs := DiffRuns(back, front, dmg)

	if len(runs) != 2 {
		t.Fatalf("DiffRuns() returned %d runs, want 2", len(runs))
	}
}

func TestCoalesceRuns_Adjacent(t *testing.T) {
	runs := []Run{
		{Y: 2, X0: 3, X1: 5},
		{Y: 2, X0: 5, X1: 7}, // adjacent
	}

	result := coalesceRuns(runs)

	if len(result) != 1 {
		t.Errorf("coalesceRuns() adjacent = %d runs, want 1", len(result))
	}

	if len(result) == 1 && (result[0].X0 != 3 || result[0].X1 != 7) {
		t.Errorf("coalesceRuns() adjacent = %+v, want {X0:3, X1:7}", result[0])
	}
}

func TestCoalesceRuns_Overlapping(t *testing.T) {
	runs := []Run{
		{Y: 2, X0: 3, X1: 6},
		{Y: 2, X0: 5, X1: 8}, // overlapping
	}

	result := coalesceRuns(runs)

	if len(result) != 1 {
		t.Errorf("coalesceRuns() overlapping = %d runs, want 1", len(result))
	}

	if len(result) == 1 && (result[0].X0 != 3 || result[0].X1 != 8) {
		t.Errorf("coalesceRuns() overlapping = %+v, want {X0:3, X1:8}", result[0])
	}
}

func TestCoalesceRuns_Separate(t *testing.T) {
	runs := []Run{
		{Y: 2, X0: 2, X1: 4},
		{Y: 2, X0: 7, X1: 9}, // separate
	}

	result := coalesceRuns(runs)

	if len(result) != 2 {
		t.Errorf("coalesceRuns() separate = %d runs, want 2", len(result))
	}
}

func TestCoalesceRuns_DifferentRows(t *testing.T) {
	runs := []Run{
		{Y: 1, X0: 3, X1: 5},
		{Y: 2, X0: 3, X1: 5}, // different row
	}

	result := coalesceRuns(runs)

	if len(result) != 2 {
		t.Errorf("coalesceRuns() different rows = %d runs, want 2", len(result))
	}
}

func TestCoalesceRuns_Empty(t *testing.T) {
	result := coalesceRuns([]Run{})
	if len(result) != 0 {
		t.Errorf("coalesceRuns() empty = %d runs, want 0", len(result))
	}

	result = coalesceRuns(nil)
	if len(result) != 0 {
		t.Errorf("coalesceRuns() nil = %d runs, want 0", len(result))
	}
}

func TestCoalesceRuns_Single(t *testing.T) {
	runs := []Run{
		{Y: 2, X0: 3, X1: 5},
	}

	result := coalesceRuns(runs)

	if len(result) != 1 {
		t.Errorf("coalesceRuns() single = %d runs, want 1", len(result))
	}
}

func TestCellsDiffer_AllFields(t *testing.T) {
	base := Cell{R: 'A', Style: style.Style{FG: style.ColorRed}, Wide: true, WideCont: false}

	tests := []struct {
		name string
		a, b Cell
		want bool
	}{
		{
			name: "identical",
			a:    base,
			b:    base,
			want: false,
		},
		{
			name: "different rune",
			a:    base,
			b:    Cell{R: 'B', Style: style.Style{FG: style.ColorRed}, Wide: true, WideCont: false},
			want: true,
		},
		{
			name: "different style FG",
			a:    base,
			b:    Cell{R: 'A', Style: style.Style{FG: style.ColorBlue}, Wide: true, WideCont: false},
			want: true,
		},
		{
			name: "different style BG",
			a:    base,
			b:    Cell{R: 'A', Style: style.Style{FG: style.ColorRed, BG: style.ColorBlue}, Wide: true, WideCont: false},
			want: true,
		},
		{
			name: "different style Attr",
			a:    base,
			b:    Cell{R: 'A', Style: style.Style{FG: style.ColorRed, Attr: style.AttrBold}, Wide: true, WideCont: false},
			want: true,
		},
		{
			name: "different Wide",
			a:    base,
			b:    Cell{R: 'A', Style: style.Style{FG: style.ColorRed}, Wide: false, WideCont: false},
			want: true,
		},
		{
			name: "different WideCont",
			a:    base,
			b:    Cell{R: 'A', Style: style.Style{FG: style.ColorRed}, Wide: true, WideCont: true},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cellsDiffer(tt.a, tt.b); got != tt.want {
				t.Errorf("cellsDiffer() = %v, want %v", got, tt.want)
			}
		})
	}
}
