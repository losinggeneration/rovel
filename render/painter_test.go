package render

import (
	"testing"

	"github.com/losinggeneration/tui/geom"
	"github.com/losinggeneration/tui/style"
)

func TestNewPainter(t *testing.T) {
	buf := NewBuffer(10, 5)
	clip := geom.Rect{X: 0, Y: 0, W: 10, H: 5}
	base := style.Style{FG: style.ColorRed}

	p := NewPainter(buf, clip, base)

	if p.buf != buf {
		t.Error("NewPainter() buf not set correctly")
	}

	if p.clip != clip {
		t.Error("NewPainter() clip not set correctly")
	}

	if p.baseStyle != base {
		t.Error("NewPainter() baseStyle not set correctly")
	}
}

func TestPainter_SetCell(t *testing.T) {
	buf := NewBuffer(10, 5)
	clip := geom.Rect{X: 0, Y: 0, W: 10, H: 5}
	p := NewPainter(buf, clip, style.Style{})

	style := style.Style{FG: style.ColorRed}
	p.SetCell(5, 3, 'A', style)

	cell := buf.At(5, 3)
	if cell.R != 'A' {
		t.Errorf("SetCell() R = %v, want 'A'", cell.R)
	}

	if cell.Style != style {
		t.Errorf("SetCell() Style = %+v, want %+v", cell.Style, style)
	}

	if cell.Wide {
		t.Error("SetCell() Wide should be false for narrow rune")
	}

	if cell.WideCont {
		t.Error("SetCell() WideCont should be false for narrow rune")
	}
}

func TestPainter_ClipClampedToBuffer_DoesNotCorruptZeroCell(t *testing.T) {
	buf := NewBuffer(2, 2)

	// A clip larger than the buffer must not let writes escape the backing
	// array. Out-of-bounds Buffer.At returns the shared zeroCell sentinel; a
	// write through it would corrupt every future out-of-bounds read
	// process-wide.
	p := NewPainter(buf, geom.Rect{X: 0, Y: 0, W: 100, H: 100}, style.Style{})
	p.SetCell(50, 50, 'X', style.Style{FG: style.ColorRed})

	oob := buf.At(1000, 1000) // returns &zeroCell
	if oob.R != ' ' || oob.Style != (style.Style{}) || oob.Wide || oob.WideCont {
		// Repair so a buggy run doesn't poison sibling tests.
		*oob = Cell{R: ' '}
		t.Fatalf("out-of-bounds write corrupted the shared zero cell: %+v", *oob)
	}
}

func TestPainter_SetClipRectClampedToBuffer(t *testing.T) {
	buf := NewBuffer(2, 2)
	p := NewPainter(buf, geom.Rect{X: 0, Y: 0, W: 2, H: 2}, style.Style{})

	p.SetClipRect(geom.Rect{X: 0, Y: 0, W: 100, H: 100})
	p.SetCell(50, 50, 'X', style.Style{})

	oob := buf.At(1000, 1000)
	if oob.R != ' ' || oob.Style != (style.Style{}) || oob.Wide || oob.WideCont {
		*oob = Cell{R: ' '}
		t.Fatalf("SetClipRect did not clamp to buffer; zero cell corrupted: %+v", *oob)
	}
}

func TestPainter_SetCell_Clipped(t *testing.T) {
	buf := NewBuffer(10, 5)
	clip := geom.Rect{X: 2, Y: 2, W: 5, H: 3}
	p := NewPainter(buf, clip, style.Style{})

	style := style.Style{FG: style.ColorRed}

	// Should be clipped (outside clip region)
	p.SetCell(0, 0, 'A', style)

	if buf.At(0, 0).R != 0 {
		t.Error("SetCell() should not write outside clip region")
	}

	// Should be clipped (outside clip region)
	p.SetCell(8, 3, 'B', style)

	if buf.At(8, 3).R != 0 {
		t.Error("SetCell() should not write outside clip region")
	}

	// Should work (inside clip region)
	p.SetCell(5, 3, 'C', style)

	if buf.At(5, 3).R != 'C' {
		t.Error("SetCell() should write inside clip region")
	}
}

func TestPainter_SetCell_Wide(t *testing.T) {
	buf := NewBuffer(10, 5)
	clip := geom.Rect{X: 0, Y: 0, W: 10, H: 5}
	p := NewPainter(buf, clip, style.Style{})

	style := style.Style{FG: style.ColorRed}

	// Write a wide character (CJK)
	p.SetCell(5, 3, '中', style)

	leadCell := buf.At(5, 3)
	contCell := buf.At(6, 3)

	if leadCell.R != '中' {
		t.Errorf("SetCell() wide lead R = %v, want '中'", leadCell.R)
	}

	if !leadCell.Wide {
		t.Error("SetCell() wide lead Wide should be true")
	}

	if leadCell.WideCont {
		t.Error("SetCell() wide lead WideCont should be false")
	}

	if contCell.R != 0 {
		t.Errorf("SetCell() wide cont R = %v, want 0", contCell.R)
	}

	if contCell.Wide {
		t.Error("SetCell() wide cont Wide should be false")
	}

	if !contCell.WideCont {
		t.Error("SetCell() wide cont WideCont should be true")
	}
}

func TestPainter_SetCell_WideAtEdge(t *testing.T) {
	buf := NewBuffer(10, 5)
	clip := geom.Rect{X: 0, Y: 0, W: 10, H: 5}
	p := NewPainter(buf, clip, style.Style{})

	style := style.Style{FG: style.ColorRed}

	// Write wide char at right edge - should become placeholder
	p.SetCell(9, 3, '中', style)

	cell := buf.At(9, 3)
	if cell.R != '?' {
		t.Errorf("SetCell() wide at edge R = %v, want '?'", cell.R)
	}

	if cell.Wide {
		t.Error("SetCell() wide at edge Wide should be false (placeholder)")
	}
}

func TestPainter_SetCell_OverwriteWideLead(t *testing.T) {
	buf := NewBuffer(10, 5)
	clip := geom.Rect{X: 0, Y: 0, W: 10, H: 5}
	p := NewPainter(buf, clip, style.Style{})

	style := style.Style{FG: style.ColorRed}

	// Write wide character
	p.SetCell(5, 3, '中', style)

	// Overwrite lead with narrow
	p.SetCell(5, 3, 'A', style)

	leadCell := buf.At(5, 3)
	contCell := buf.At(6, 3)

	if leadCell.R != 'A' {
		t.Errorf("SetCell() overwritten lead R = %v, want 'A'", leadCell.R)
	}

	if leadCell.Wide {
		t.Error("SetCell() overwritten lead Wide should be false")
	}

	// Continuation should be cleared
	if contCell.R != 0 && contCell.R != ' ' {
		t.Errorf("SetCell() continuation should be cleared, got R = %v", contCell.R)
	}

	if contCell.WideCont {
		t.Error("SetCell() continuation WideCont should be false after lead overwrite")
	}
}

func TestPainter_SetCell_OverwriteWideCont(t *testing.T) {
	buf := NewBuffer(10, 5)
	clip := geom.Rect{X: 0, Y: 0, W: 10, H: 5}
	p := NewPainter(buf, clip, style.Style{})

	style := style.Style{FG: style.ColorRed}

	// Write wide character
	p.SetCell(5, 3, '中', style)

	// Overwrite continuation
	p.SetCell(6, 3, 'B', style)

	leadCell := buf.At(5, 3)
	contCell := buf.At(6, 3)

	// Lead should be cleared
	if leadCell.R != 0 && leadCell.R != ' ' {
		t.Errorf("SetCell() lead should be cleared when continuation overwritten, got R = %v", leadCell.R)
	}

	if leadCell.Wide {
		t.Error("SetCell() lead Wide should be false when continuation overwritten")
	}

	if contCell.R != 'B' {
		t.Errorf("SetCell() continuation R = %v, want 'B'", contCell.R)
	}
}

func TestPainter_Text(t *testing.T) {
	buf := NewBuffer(10, 5)
	clip := geom.Rect{X: 0, Y: 0, W: 10, H: 5}
	p := NewPainter(buf, clip, style.Style{})

	style := style.Style{FG: style.ColorRed}
	p.Text(2, 3, "Hello", style)

	// Check each character
	expected := "Hello"
	for i, ch := range expected {
		cell := buf.At(2+i, 3)
		if cell.R != ch {
			t.Errorf("Text() at x=%d R = %v, want %v", 2+i, cell.R, ch)
		}
	}
}

func TestPainter_Text_Clipped(t *testing.T) {
	buf := NewBuffer(10, 5)
	clip := geom.Rect{X: 2, Y: 2, W: 5, H: 3}
	p := NewPainter(buf, clip, style.Style{})

	style := style.Style{FG: style.ColorRed}

	// Text that extends beyond clip region
	p.Text(5, 3, "HelloWorld", style)

	// Clip covers X from 2 to 6 (exclusive of 7)
	// Text starts at X=5, so we get: H at 5, e at 6, then stop
	if buf.At(5, 3).R != 'H' {
		t.Errorf("Text() clipped at x=5 R = %v, want 'H'", buf.At(5, 3).R)
	}

	if buf.At(6, 3).R != 'e' {
		t.Errorf("Text() clipped at x=6 R = %v, want 'e'", buf.At(6, 3).R)
	}

	// Character at X=7 should be empty (outside clip)
	if buf.At(7, 3).R != 0 {
		t.Errorf("Text() should not write outside clip at x=7, got %v", buf.At(7, 3).R)
	}
}

func TestPainter_Fill(t *testing.T) {
	buf := NewBuffer(10, 5)
	clip := geom.Rect{X: 0, Y: 0, W: 10, H: 5}
	p := NewPainter(buf, clip, style.Style{})

	style := style.Style{FG: style.ColorRed}
	rect := geom.Rect{X: 2, Y: 1, W: 5, H: 3}
	p.Fill(rect, 'X', style)

	// Check filled area
	for y := rect.Y; y < rect.Y+rect.H; y++ {
		for x := rect.X; x < rect.X+rect.W; x++ {
			cell := buf.At(x, y)
			if cell.R != 'X' {
				t.Errorf("Fill() at (%d,%d) R = %v, want 'X'", x, y, cell.R)
			}
		}
	}

	// Check area outside fill
	if buf.At(1, 2).R != 0 {
		t.Error("Fill() should not write outside rect")
	}
}

func TestPainter_Fill_Clipped(t *testing.T) {
	buf := NewBuffer(10, 5)
	clip := geom.Rect{X: 2, Y: 2, W: 5, H: 3}
	p := NewPainter(buf, clip, style.Style{})

	style := style.Style{FG: style.ColorRed}

	// Fill that extends beyond clip region
	rect := geom.Rect{X: 0, Y: 0, W: 10, H: 5}
	p.Fill(rect, 'X', style)

	// Should only fill within clip
	for y := clip.Y; y < clip.Y+clip.H; y++ {
		for x := clip.X; x < clip.X+clip.W; x++ {
			cell := buf.At(x, y)
			if cell.R != 'X' {
				t.Errorf("Fill() clipped at (%d,%d) R = %v, want 'X'", x, y, cell.R)
			}
		}
	}

	// Outside clip should be empty
	if buf.At(0, 0).R != 0 {
		t.Error("Fill() should not write outside clip")
	}
}

func TestPainter_HLine(t *testing.T) {
	buf := NewBuffer(10, 5)
	clip := geom.Rect{X: 0, Y: 0, W: 10, H: 5}
	p := NewPainter(buf, clip, style.Style{})

	style := style.Style{FG: style.ColorRed}
	p.HLine(2, 3, 5, '-', style)

	for x := 2; x < 7; x++ {
		cell := buf.At(x, 3)
		if cell.R != '-' {
			t.Errorf("HLine() at x=%d R = %v, want '-'", x, cell.R)
		}
	}

	// Check cells before and after
	if buf.At(1, 3).R != 0 {
		t.Error("HLine() should not write before start")
	}

	if buf.At(7, 3).R != 0 {
		t.Error("HLine() should not write after end")
	}
}

func TestPainter_VLine(t *testing.T) {
	buf := NewBuffer(10, 10)
	clip := geom.Rect{X: 0, Y: 0, W: 10, H: 10}
	p := NewPainter(buf, clip, style.Style{})

	style := style.Style{FG: style.ColorRed}
	p.VLine(5, 2, 4, '|', style)

	for y := 2; y < 6; y++ {
		cell := buf.At(5, y)
		if cell.R != '|' {
			t.Errorf("VLine() at y=%d R = %v, want '|'", y, cell.R)
		}
	}

	// Check cells before and after
	if buf.At(5, 1).R != 0 {
		t.Error("VLine() should not write before start")
	}

	if buf.At(5, 6).R != 0 {
		t.Error("VLine() should not write after end")
	}
}

func TestPainter_Box(t *testing.T) {
	buf := NewBuffer(10, 10)
	clip := geom.Rect{X: 0, Y: 0, W: 10, H: 10}
	p := NewPainter(buf, clip, style.Style{})

	style := style.Style{FG: style.ColorRed}
	rect := geom.Rect{X: 2, Y: 3, W: 6, H: 4}
	p.Box(rect, style)

	// Check corners
	corners := []struct {
		x, y int
		ch   rune
	}{
		{2, 3, '┌'}, // Top-left
		{7, 3, '┐'}, // Top-right
		{2, 6, '└'}, // Bottom-left
		{7, 6, '┘'}, // Bottom-right
	}

	for _, c := range corners {
		cell := buf.At(c.x, c.y)
		if cell.R != c.ch {
			t.Errorf("Box() corner at (%d,%d) R = %v, want %v", c.x, c.y, cell.R, c.ch)
		}
	}

	// Check top edge
	for x := 3; x < 7; x++ {
		if buf.At(x, 3).R != '─' {
			t.Errorf("Box() top edge at x=%d R = %v, want '─'", x, buf.At(x, 3).R)
		}
	}

	// Check bottom edge
	for x := 3; x < 7; x++ {
		if buf.At(x, 6).R != '─' {
			t.Errorf("Box() bottom edge at x=%d R = %v, want '─'", x, buf.At(x, 6).R)
		}
	}

	// Check left edge
	for y := 4; y < 6; y++ {
		if buf.At(2, y).R != '│' {
			t.Errorf("Box() left edge at y=%d R = %v, want '│'", y, buf.At(2, y).R)
		}
	}

	// Check right edge
	for y := 4; y < 6; y++ {
		if buf.At(7, y).R != '│' {
			t.Errorf("Box() right edge at y=%d R = %v, want '│'", y, buf.At(7, y).R)
		}
	}
}

func TestPainter_BoxStyled_ASCII(t *testing.T) {
	buf := NewBuffer(10, 10)
	clip := geom.Rect{X: 0, Y: 0, W: 10, H: 10}
	p := NewPainter(buf, clip, style.Style{})

	st := style.Style{FG: style.ColorRed}
	rect := geom.Rect{X: 2, Y: 3, W: 6, H: 4}
	p.BoxStyled(rect, BoxStyle{
		Glyphs: BoxGlyphsASCII,
		Edges:  BoxEdgesAll,
		Style:  st,
	})

	// Corners should use '+'
	corners := []struct {
		x, y int
		ch   rune
	}{
		{2, 3, '+'},
		{7, 3, '+'},
		{2, 6, '+'},
		{7, 6, '+'},
	}
	for _, c := range corners {
		cell := buf.At(c.x, c.y)
		if cell.R != c.ch {
			t.Errorf("BoxStyled(ASCII) corner at (%d,%d) R = %v, want %v", c.x, c.y, cell.R, c.ch)
		}

		if cell.Style != st {
			t.Errorf("BoxStyled(ASCII) corner at (%d,%d) Style = %+v, want %+v", c.x, c.y, cell.Style, st)
		}
	}

	// Top/bottom edges should use '-'
	for x := 3; x < 7; x++ {
		if buf.At(x, 3).R != '-' {
			t.Errorf("BoxStyled(ASCII) top edge at x=%d R = %v, want '-'", x, buf.At(x, 3).R)
		}

		if buf.At(x, 6).R != '-' {
			t.Errorf("BoxStyled(ASCII) bottom edge at x=%d R = %v, want '-'", x, buf.At(x, 6).R)
		}
	}

	// Left/right edges should use '|'
	for y := 4; y < 6; y++ {
		if buf.At(2, y).R != '|' {
			t.Errorf("BoxStyled(ASCII) left edge at y=%d R = %v, want '|'", y, buf.At(2, y).R)
		}

		if buf.At(7, y).R != '|' {
			t.Errorf("BoxStyled(ASCII) right edge at y=%d R = %v, want '|'", y, buf.At(7, y).R)
		}
	}
}

func TestPainter_BoxStyled_EdgesOnly(t *testing.T) {
	buf := NewBuffer(10, 10)
	clip := geom.Rect{X: 0, Y: 0, W: 10, H: 10}
	p := NewPainter(buf, clip, style.Style{})

	rect := geom.Rect{X: 2, Y: 3, W: 6, H: 1}
	p.BoxStyled(rect, BoxStyle{
		Glyphs: BoxGlyphsASCII,
		Edges:  BoxEdgeLeft | BoxEdgeRight,
		Style:  style.Style{FG: style.ColorGreen},
	})

	if buf.At(2, 3).R != '|' {
		t.Errorf("BoxStyled(edges) left bar R = %v, want '|'", buf.At(2, 3).R)
	}

	if buf.At(7, 3).R != '|' {
		t.Errorf("BoxStyled(edges) right bar R = %v, want '|'", buf.At(7, 3).R)
	}

	if buf.At(3, 3).R != 0 {
		t.Errorf("BoxStyled(edges) interior wrote R = %v, want empty", buf.At(3, 3).R)
	}
}

func TestPainter_Box_SingleCell(t *testing.T) {
	buf := NewBuffer(10, 10)
	clip := geom.Rect{X: 0, Y: 0, W: 10, H: 10}
	p := NewPainter(buf, clip, style.Style{})

	style := style.Style{FG: style.ColorRed}
	rect := geom.Rect{X: 5, Y: 5, W: 1, H: 1}
	p.Box(rect, style)

	// Single cell box: top-left and top-right are at same position
	// The last one written (top-right '┐') overwrites top-left
	cell := buf.At(5, 5)
	if cell.R != '┐' {
		t.Errorf("Box() single cell R = %v, want '┐'", cell.R)
	}
}

func TestPainter_Box_EmptyRect(t *testing.T) {
	buf := NewBuffer(10, 10)
	clip := geom.Rect{X: 0, Y: 0, W: 10, H: 10}
	p := NewPainter(buf, clip, style.Style{})

	style := style.Style{FG: style.ColorRed}

	// Empty rect should do nothing
	p.Box(geom.Rect{X: 5, Y: 5, W: 0, H: 5}, style)
	p.Box(geom.Rect{X: 5, Y: 5, W: 5, H: 0}, style)

	// Buffer should be empty
	for y := range buf.H {
		for x := range buf.W {
			if buf.At(x, y).R != 0 {
				t.Errorf("Box() with empty rect wrote at (%d,%d)", x, y)
			}
		}
	}
}

func TestPainter_Offset(t *testing.T) {
	buf := NewBuffer(10, 10)
	clip := geom.Rect{X: 0, Y: 0, W: 10, H: 10}
	p := NewPainter(buf, clip, style.Style{})

	// Default offset should be 0,0
	x, y := p.Offset()
	if x != 0 || y != 0 {
		t.Errorf("default offset = (%d,%d), want (0,0)", x, y)
	}

	// Set offset
	p.SetOffset(5, 3)

	x, y = p.Offset()
	if x != 5 || y != 3 {
		t.Errorf("after SetOffset(5,3) = (%d,%d), want (5,3)", x, y)
	}

	// SetCell should use offset
	st := style.Style{}
	p.SetCell(2, 1, 'X', st)

	// Should be written at (2+5, 1+3) = (7, 4)
	if buf.At(7, 4).R != 'X' {
		t.Errorf("SetCell with offset wrote at wrong position")
	}
	// Original position should be empty
	if buf.At(2, 1).R != 0 {
		t.Errorf("SetCell with offset wrote to original position")
	}
}

func TestPainter_Text_WithOffset(t *testing.T) {
	buf := NewBuffer(20, 20)
	clip := geom.Rect{X: 0, Y: 0, W: 20, H: 20}
	p := NewPainter(buf, clip, style.Style{})

	p.SetOffset(5, 3)
	p.Text(2, 1, "ABC", style.Style{})

	// Should be written at (7, 4), (8, 4), (9, 4) after offset
	if buf.At(7, 4).R != 'A' {
		t.Errorf("Text with offset: 'A' at (7,4), got %c", buf.At(7, 4).R)
	}

	if buf.At(8, 4).R != 'B' {
		t.Errorf("Text with offset: 'B' at (8,4), got %c", buf.At(8, 4).R)
	}

	if buf.At(9, 4).R != 'C' {
		t.Errorf("Text with offset: 'C' at (9,4), got %c", buf.At(9, 4).R)
	}
	// Original positions should be empty
	if buf.At(2, 1).R != 0 {
		t.Errorf("Text with offset wrote to original position")
	}
}

func TestPainter_Text_WithOffset_Clipping(t *testing.T) {
	buf := NewBuffer(10, 10)
	clip := geom.Rect{X: 0, Y: 0, W: 10, H: 10}
	p := NewPainter(buf, clip, style.Style{})

	// Offset to (8, 8), then draw at (5, 5) -> (13, 13) which is outside clip
	p.SetOffset(8, 8)
	p.Text(5, 5, "ABC", style.Style{})

	// Characters at (13,13), (14,13), (15,13) are all outside clip (0-9)
	// So nothing should be written (cells should remain as zeroCell which is ' ')
	// The zeroCell has R = ' ', so check that it wasn't overwritten with something else
	// by verifying a cell INSIDE the clip region was NOT affected
	if buf.At(1, 1).R != 0 {
		t.Error("Text incorrectly wrote to position (1,1) which is inside clip")
	}
}

func TestPainter_Fill_WithOffset(t *testing.T) {
	buf := NewBuffer(20, 20)
	clip := geom.Rect{X: 0, Y: 0, W: 20, H: 20}
	p := NewPainter(buf, clip, style.Style{})

	p.SetOffset(5, 3)
	p.Fill(geom.Rect{X: 2, Y: 1, W: 3, H: 2}, 'X', style.Style{})

	// Should fill at (7,4), (8,4), (9,4), (7,5), (8,5), (9,5)
	for dy := range 2 {
		for dx := range 3 {
			x := 5 + 2 + dx // 7, 8, 9

			y := 3 + 1 + dy // 4, 5
			if buf.At(x, y).R != 'X' {
				t.Errorf("Fill with offset: expected 'X' at (%d,%d)", x, y)
			}
		}
	}
}

func TestPainter_HLine_WithOffset(t *testing.T) {
	buf := NewBuffer(20, 20)
	clip := geom.Rect{X: 0, Y: 0, W: 20, H: 20}
	p := NewPainter(buf, clip, style.Style{})

	p.SetOffset(5, 3)
	p.HLine(2, 1, 4, '=', style.Style{})

	// Should draw at (7,4), (8,4), (9,4), (10,4)
	for dx := range 4 {
		x := 5 + 2 + dx
		if buf.At(x, 4).R != '=' {
			t.Errorf("HLine with offset: expected '=' at (%d,4)", x)
		}
	}
}

func TestPainter_VLine_WithOffset(t *testing.T) {
	buf := NewBuffer(20, 20)
	clip := geom.Rect{X: 0, Y: 0, W: 20, H: 20}
	p := NewPainter(buf, clip, style.Style{})

	p.SetOffset(5, 3)
	p.VLine(2, 1, 4, '|', style.Style{})

	// Should draw at (7,4), (7,5), (7,6), (7,7)
	for dy := range 4 {
		y := 3 + 1 + dy
		if buf.At(7, y).R != '|' {
			t.Errorf("VLine with offset: expected '|' at (7,%d)", y)
		}
	}
}

func TestPainter_Box_WithOffset(t *testing.T) {
	buf := NewBuffer(20, 20)
	clip := geom.Rect{X: 0, Y: 0, W: 20, H: 20}
	p := NewPainter(buf, clip, style.Style{})

	p.SetOffset(5, 3)
	p.Box(geom.Rect{X: 2, Y: 1, W: 5, H: 3}, style.Style{})

	// Top-left corner at (7, 4)
	if buf.At(7, 4).R != '┌' {
		t.Errorf("Box with offset: expected '┌' at (7,4), got %c", buf.At(7, 4).R)
	}
	// Bottom-right corner at (11, 6) = (5+2+5-1, 3+1+3-1) = (11, 6)
	if buf.At(11, 6).R != '┘' {
		t.Errorf("Box with offset: expected '┘' at (11,6), got %c", buf.At(11, 6).R)
	}
}

func TestPainter_Offset_Clipping(t *testing.T) {
	buf := NewBuffer(10, 10)
	clip := geom.Rect{X: 0, Y: 0, W: 5, H: 5} // Clip to top-left 5x5
	p := NewPainter(buf, clip, style.Style{})

	st := style.Style{}

	// With offset, should still clip correctly
	// Set offset to (3, 3), then set cell at (0, 0)
	// Should write at (3, 3) which is inside clip
	p.SetOffset(3, 3)
	p.SetCell(0, 0, 'A', st)

	if buf.At(3, 3).R != 'A' {
		t.Error("offset cell not written inside clip")
	}

	// Set cell at (3, 3) -> position (6, 6) which is outside clip
	p.SetCell(3, 3, 'B', st)

	if buf.At(6, 6).R != 0 {
		t.Error("offset cell should be clipped")
	}
}
