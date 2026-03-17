package render

import (
	"math/rand/v2"
	"testing"

	"github.com/losinggeneration/tui/geom"
	"github.com/losinggeneration/tui/style"
)

// TestSetCell_NeverWritesOutsideClipRect verifies that SetCell with a clip rect
// never modifies cells outside that rect.
func TestSetCell_NeverWritesOutsideClipRect(t *testing.T) {
	const W, H = 20, 10

	rng := rand.New(rand.NewPCG(42, 0))

	for range 200 {
		// Random clip rect within buffer.
		cx := rng.IntN(W)
		cy := rng.IntN(H)
		cw := rng.IntN(W-cx) + 1
		ch := rng.IntN(H-cy) + 1
		clip := geom.Rect{X: cx, Y: cy, W: cw, H: ch}

		buf := NewBuffer(W, H)
		p := NewPainter(buf, clip, style.Style{})

		// Write to random positions, some inside clip, some outside.
		for range 50 {
			x := rng.IntN(W+4) - 2 // may be negative or beyond buffer
			y := rng.IntN(H+4) - 2
			p.SetCell(x, y, 'X', style.Style{})
		}

		// Verify no cell outside clip was modified.
		for y := range H {
			for x := range W {
				if x >= clip.X && x < clip.X+clip.W && y >= clip.Y && y < clip.Y+clip.H {
					continue // inside clip, may have been modified
				}

				cell := buf.At(x, y)
				if cell.R != 0 && cell.R != ' ' {
					t.Fatalf("cell at (%d,%d) was modified outside clip %v: R=%q", x, y, clip, cell.R)
				}
			}
		}
	}
}

// TestDiffRuns_FrontMatchesBackAfterFlush verifies that after DiffRuns and
// copying back->front for flushed cells, those cells match.
func TestDiffRuns_FrontMatchesBackAfterFlush(t *testing.T) {
	const W, H = 30, 10

	rng := rand.New(rand.NewPCG(99, 0))

	back := NewBuffer(W, H)
	front := NewBuffer(W, H)
	dmg := NewDamage(W, H)

	// Write random content to back buffer.
	for range 100 {
		x := rng.IntN(W)
		y := rng.IntN(H)
		cell := back.At(x, y)
		cell.R = rune('A' + rng.IntN(26))
	}

	// Add some damage regions.
	dmg.AddRect(geom.Rect{X: 0, Y: 0, W: W, H: H})

	runs := DiffRuns(back, front, dmg)

	// Simulate flush: copy back->front for all run cells.
	for _, run := range runs {
		for x := run.X0; x < run.X1; x++ {
			*front.At(x, run.Y) = *back.At(x, run.Y)
		}
	}

	// After flush, all cells in damaged region should match.
	for y := range H {
		for x := range W {
			b := *back.At(x, y)

			f := *front.At(x, y)
			if b != f {
				t.Fatalf("cell (%d,%d) differs after flush: back=%+v front=%+v", x, y, b, f)
			}
		}
	}
}

// TestWideChar_ContinuationClearing verifies that writing over a wide char
// clears the continuation cell.
func TestWideChar_ContinuationClearing(t *testing.T) {
	buf := NewBuffer(20, 5)
	clip := geom.Rect{X: 0, Y: 0, W: 20, H: 5}
	p := NewPainter(buf, clip, style.Style{})

	// Write a wide character at (3,1).
	p.SetCell(3, 1, '日', style.Style{}) // 2 columns wide

	// Verify lead and continuation cells.
	lead := buf.At(3, 1)
	cont := buf.At(4, 1)

	if !lead.Wide {
		t.Fatal("lead cell should be Wide")
	}

	if !cont.WideCont {
		t.Fatal("continuation cell should be WideCont")
	}

	// Overwrite the lead cell with a narrow character.
	p.SetCell(3, 1, 'a', style.Style{})

	lead = buf.At(3, 1)
	cont = buf.At(4, 1)

	if lead.Wide {
		t.Error("lead cell should no longer be Wide after narrow overwrite")
	}

	if cont.WideCont {
		t.Error("continuation cell should be cleared after lead overwrite")
	}

	if cont.R != ' ' {
		t.Errorf("continuation cell R should be space, got %q", cont.R)
	}
}

// TestWideChar_OverwriteContinuation verifies that writing over a continuation
// cell clears the lead cell.
func TestWideChar_OverwriteContinuation(t *testing.T) {
	buf := NewBuffer(20, 5)
	clip := geom.Rect{X: 0, Y: 0, W: 20, H: 5}
	p := NewPainter(buf, clip, style.Style{})

	// Write a wide character at (3,1).
	p.SetCell(3, 1, '本', style.Style{})

	// Overwrite the continuation cell (4,1) with a narrow character.
	p.SetCell(4, 1, 'b', style.Style{})

	lead := buf.At(3, 1)
	over := buf.At(4, 1)

	if lead.Wide {
		t.Error("lead cell should be cleared to narrow")
	}

	if lead.R != ' ' {
		t.Errorf("lead cell R should be space, got %q", lead.R)
	}

	if over.R != 'b' {
		t.Errorf("overwritten cell should be 'b', got %q", over.R)
	}
}

// TestBufferResize_PreservesContent verifies that Buffer.Resize preserves
// content within the old/new intersection.
func TestBufferResize_PreservesContent(t *testing.T) {
	buf := NewBuffer(10, 5)

	// Write known content.
	for y := range 5 {
		for x := range 10 {
			cell := buf.At(x, y)
			cell.R = rune('A' + y*10 + x)
		}
	}

	// Resize to larger.
	buf.Resize(15, 8)

	if buf.W != 15 || buf.H != 8 {
		t.Fatalf("resize to 15x8 failed: got %dx%d", buf.W, buf.H)
	}

	// Check old content preserved.
	for y := range 5 {
		for x := range 10 {
			cell := buf.At(x, y)

			want := rune('A' + y*10 + x)
			if cell.R != want {
				t.Errorf("at (%d,%d): got %q, want %q", x, y, cell.R, want)
			}
		}
	}

	// Check new area is zero-initialized.
	for x := 10; x < 15; x++ {
		cell := buf.At(x, 0)
		if cell.R != 0 {
			t.Errorf("new area at (%d,0) should be zero, got %q", x, cell.R)
		}
	}
}

// TestBufferResize_Shrink verifies resize to smaller preserves intersection.
func TestBufferResize_Shrink(t *testing.T) {
	buf := NewBuffer(10, 5)

	for y := range 5 {
		for x := range 10 {
			buf.At(x, y).R = rune('A' + y*10 + x)
		}
	}

	buf.Resize(5, 3)

	for y := range 3 {
		for x := range 5 {
			cell := buf.At(x, y)

			want := rune('A' + y*10 + x)
			if cell.R != want {
				t.Errorf("at (%d,%d): got %q, want %q", x, y, cell.R, want)
			}
		}
	}
}

// TestDamage_NormalizationNoOverlap verifies that after AddSpan/AddRect,
// no spans on the same row overlap.
func TestDamage_NormalizationNoOverlap(t *testing.T) {
	rng := rand.New(rand.NewPCG(7, 0))

	for range 100 {
		dmg := NewDamage(80, 24)

		// Add random rects and spans.
		for range 20 {
			x := rng.IntN(80)
			y := rng.IntN(24)
			w := rng.IntN(40) + 1
			h := rng.IntN(12) + 1
			dmg.AddRect(geom.Rect{X: x, Y: y, W: w, H: h})
		}

		for range 10 {
			y := rng.IntN(24)
			x0 := rng.IntN(80)
			x1 := x0 + rng.IntN(40) + 1
			dmg.AddSpan(y, x0, x1)
		}

		// Verify no overlaps per row, and sorted order.
		for y, row := range dmg.Rows {
			for i := 1; i < len(row); i++ {
				if row[i].X0 < row[i-1].X1 {
					t.Fatalf("row %d: overlapping spans: [%d,%d) and [%d,%d)",
						y, row[i-1].X0, row[i-1].X1, row[i].X0, row[i].X1)
				}

				if row[i].X0 < row[i-1].X0 {
					t.Fatalf("row %d: unsorted spans", y)
				}
			}
		}
	}
}

// TestDamage_SpansClamped verifies that AddSpan and AddRect clamp to buffer bounds.
func TestDamage_SpansClamped(t *testing.T) {
	dmg := NewDamage(10, 5)

	// Negative and out-of-bounds rects.
	dmg.AddRect(geom.Rect{X: -5, Y: -3, W: 20, H: 10})

	for y, row := range dmg.Rows {
		for _, sp := range row {
			if sp.X0 < 0 || sp.X1 > dmg.W || sp.X0 >= sp.X1 {
				t.Fatalf("row %d: span out of bounds: [%d,%d)", y, sp.X0, sp.X1)
			}
		}
	}

	// Out-of-bounds spans.
	dmg.AddSpan(-1, 0, 5)
	dmg.AddSpan(100, 0, 5)
	dmg.AddSpan(2, -3, 15)

	for y, row := range dmg.Rows {
		for _, sp := range row {
			if sp.X0 < 0 || sp.X1 > dmg.W {
				t.Fatalf("row %d: span out of bounds after AddSpan: [%d,%d)", y, sp.X0, sp.X1)
			}
		}
	}
}

// TestSetCell_NegativeAndOOBCoords verifies clipping with negative/out-of-bounds.
func TestSetCell_NegativeAndOOBCoords(t *testing.T) {
	buf := NewBuffer(10, 5)
	clip := geom.Rect{X: 0, Y: 0, W: 10, H: 5}
	p := NewPainter(buf, clip, style.Style{})

	// These should not panic.
	p.SetCell(-1, 0, 'X', style.Style{})
	p.SetCell(0, -1, 'X', style.Style{})
	p.SetCell(10, 0, 'X', style.Style{})
	p.SetCell(0, 5, 'X', style.Style{})
	p.SetCell(100, 100, 'X', style.Style{})

	// Verify nothing was written.
	for y := range 5 {
		for x := range 10 {
			cell := buf.At(x, y)
			if cell.R != 0 {
				t.Errorf("cell at (%d,%d) unexpectedly modified: R=%q", x, y, cell.R)
			}
		}
	}
}
