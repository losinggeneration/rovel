package render

import (
	"testing"

	"github.com/losinggeneration/tui"
)

func TestNewPainter(t *testing.T) {
	buf := NewBuffer(10, 5)
	damage := NewDamage(10, 5)
	clip := tui.Rect{X: 0, Y: 0, W: 10, H: 5}

	p := NewPainter(buf, clip, damage)

	if p.buf != buf {
		t.Error("NewPainter() buf not set correctly")
	}
	if p.clip != clip {
		t.Error("NewPainter() clip not set correctly")
	}
	if p.damage != damage {
		t.Error("NewPainter() damage not set correctly")
	}
}

func TestPainter_SetCell(t *testing.T) {
	buf := NewBuffer(10, 5)
	damage := NewDamage(10, 5)
	clip := tui.Rect{X: 0, Y: 0, W: 10, H: 5}
	p := NewPainter(buf, clip, damage)

	style := tui.Style{FG: tui.ColorRed}
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

	// Check damage
	if len(damage.Rows[3]) != 1 {
		t.Errorf("SetCell() damage rows = %d, want 1", len(damage.Rows[3]))
	}
}

func TestPainter_SetCell_Clipped(t *testing.T) {
	buf := NewBuffer(10, 5)
	damage := NewDamage(10, 5)
	clip := tui.Rect{X: 2, Y: 2, W: 5, H: 3}
	p := NewPainter(buf, clip, damage)

	style := tui.Style{FG: tui.ColorRed}

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
	damage := NewDamage(10, 5)
	clip := tui.Rect{X: 0, Y: 0, W: 10, H: 5}
	p := NewPainter(buf, clip, damage)

	style := tui.Style{FG: tui.ColorRed}

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
	damage := NewDamage(10, 5)
	clip := tui.Rect{X: 0, Y: 0, W: 10, H: 5}
	p := NewPainter(buf, clip, damage)

	style := tui.Style{FG: tui.ColorRed}

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
	damage := NewDamage(10, 5)
	clip := tui.Rect{X: 0, Y: 0, W: 10, H: 5}
	p := NewPainter(buf, clip, damage)

	style := tui.Style{FG: tui.ColorRed}

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
	damage := NewDamage(10, 5)
	clip := tui.Rect{X: 0, Y: 0, W: 10, H: 5}
	p := NewPainter(buf, clip, damage)

	style := tui.Style{FG: tui.ColorRed}

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
	damage := NewDamage(10, 5)
	clip := tui.Rect{X: 0, Y: 0, W: 10, H: 5}
	p := NewPainter(buf, clip, damage)

	style := tui.Style{FG: tui.ColorRed}
	p.Text(2, 3, "Hello", style)

	// Check each character
	expected := "Hello"
	for i, ch := range expected {
		cell := buf.At(2+i, 3)
		if cell.R != ch {
			t.Errorf("Text() at x=%d R = %v, want %v", 2+i, cell.R, ch)
		}
	}

	// Check damage spans
	if len(damage.Rows[3]) != 1 {
		t.Errorf("Text() should create one damage span, got %d", len(damage.Rows[3]))
	}
}

func TestPainter_Text_Clipped(t *testing.T) {
	buf := NewBuffer(10, 5)
	damage := NewDamage(10, 5)
	clip := tui.Rect{X: 2, Y: 2, W: 5, H: 3}
	p := NewPainter(buf, clip, damage)

	style := tui.Style{FG: tui.ColorRed}

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
	damage := NewDamage(10, 5)
	clip := tui.Rect{X: 0, Y: 0, W: 10, H: 5}
	p := NewPainter(buf, clip, damage)

	style := tui.Style{FG: tui.ColorRed}
	rect := tui.Rect{X: 2, Y: 1, W: 5, H: 3}
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
	damage := NewDamage(10, 5)
	clip := tui.Rect{X: 2, Y: 2, W: 5, H: 3}
	p := NewPainter(buf, clip, damage)

	style := tui.Style{FG: tui.ColorRed}

	// Fill that extends beyond clip region
	rect := tui.Rect{X: 0, Y: 0, W: 10, H: 5}
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
	damage := NewDamage(10, 5)
	clip := tui.Rect{X: 0, Y: 0, W: 10, H: 5}
	p := NewPainter(buf, clip, damage)

	style := tui.Style{FG: tui.ColorRed}
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
	damage := NewDamage(10, 10)
	clip := tui.Rect{X: 0, Y: 0, W: 10, H: 10}
	p := NewPainter(buf, clip, damage)

	style := tui.Style{FG: tui.ColorRed}
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
	damage := NewDamage(10, 10)
	clip := tui.Rect{X: 0, Y: 0, W: 10, H: 10}
	p := NewPainter(buf, clip, damage)

	style := tui.Style{FG: tui.ColorRed}
	rect := tui.Rect{X: 2, Y: 3, W: 6, H: 4}
	p.Box(rect, style)

	// Check corners
	corners := []struct {
		x, y int
		ch   rune
	}{
		{2, 3, '┌'},  // Top-left
		{7, 3, '┐'},  // Top-right
		{2, 6, '└'},  // Bottom-left
		{7, 6, '┘'},  // Bottom-right
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

func TestPainter_Box_SingleCell(t *testing.T) {
	buf := NewBuffer(10, 10)
	damage := NewDamage(10, 10)
	clip := tui.Rect{X: 0, Y: 0, W: 10, H: 10}
	p := NewPainter(buf, clip, damage)

	style := tui.Style{FG: tui.ColorRed}
	rect := tui.Rect{X: 5, Y: 5, W: 1, H: 1}
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
	damage := NewDamage(10, 10)
	clip := tui.Rect{X: 0, Y: 0, W: 10, H: 10}
	p := NewPainter(buf, clip, damage)

	style := tui.Style{FG: tui.ColorRed}

	// Empty rect should do nothing
	p.Box(tui.Rect{X: 5, Y: 5, W: 0, H: 5}, style)
	p.Box(tui.Rect{X: 5, Y: 5, W: 5, H: 0}, style)

	// Buffer should be empty
	for y := 0; y < buf.H; y++ {
		for x := 0; x < buf.W; x++ {
			if buf.At(x, y).R != 0 {
				t.Errorf("Box() with empty rect wrote at (%d,%d)", x, y)
			}
		}
	}
}
