package render

import (
	"testing"

	"github.com/losinggeneration/tui"
)

func TestNewBuffer(t *testing.T) {
	b := NewBuffer(10, 5)

	if b.W != 10 {
		t.Errorf("NewBuffer() W = %v, want 10", b.W)
	}
	if b.H != 5 {
		t.Errorf("NewBuffer() H = %v, want 5", b.H)
	}
	if len(b.cells) != 50 {
		t.Errorf("NewBuffer() cells length = %v, want 50", len(b.cells))
	}
}

func TestBuffer_At(t *testing.T) {
	b := NewBuffer(10, 5)

	// Test valid coordinates
	cell := b.At(5, 3)
	if cell == nil {
		t.Error("At() should not return nil for valid coordinates")
	}

	// Test out of bounds - should return zero cell
	cell = b.At(-1, 0)
	if cell == nil {
		t.Error("At() should not return nil for out of bounds")
	}
	if !cell.IsZero() {
		t.Error("At() should return zero cell for out of bounds")
	}

	cell = b.At(0, -1)
	if !cell.IsZero() {
		t.Error("At() should return zero cell for negative y")
	}

	cell = b.At(10, 0)
	if !cell.IsZero() {
		t.Error("At() should return zero cell for x >= W")
	}

	cell = b.At(0, 5)
	if !cell.IsZero() {
		t.Error("At() should return zero cell for y >= H")
	}

	// Test that returned cell is a pointer to actual data
	b.At(5, 3).R = 'X'
	if b.At(5, 3).R != 'X' {
		t.Error("At() should return pointer to actual cell")
	}
}

func TestBuffer_Resize(t *testing.T) {
	b := NewBuffer(10, 5)

	// Write some data
	for y := 0; y < 5; y++ {
		for x := 0; x < 10; x++ {
			cell := b.At(x, y)
			cell.R = rune('A' + x%26)
			cell.Style = tui.Style{FG: tui.Color(x % 16)}
		}
	}

	// Resize to larger buffer
	b.Resize(15, 8)

	if b.W != 15 {
		t.Errorf("Resize() W = %v, want 15", b.W)
	}
	if b.H != 8 {
		t.Errorf("Resize() H = %v, want 8", b.H)
	}

	// Verify preserved content
	for y := 0; y < 5; y++ {
		for x := 0; x < 10; x++ {
			cell := b.At(x, y)
			expectedR := rune('A' + x%26)
			if cell.R != expectedR {
				t.Errorf("Resize() preserved content at (%d,%d) = %v, want %v", x, y, cell.R, expectedR)
			}
		}
	}

	// Resize to smaller buffer
	b.Resize(5, 3)

	if b.W != 5 {
		t.Errorf("Resize() W = %v, want 5", b.W)
	}
	if b.H != 3 {
		t.Errorf("Resize() H = %v, want 3", b.H)
	}

	// Verify preserved content in overlap
	for y := 0; y < 3; y++ {
		for x := 0; x < 5; x++ {
			cell := b.At(x, y)
			expectedR := rune('A' + x%26)
			if cell.R != expectedR {
				t.Errorf("Resize() preserved content at (%d,%d) = %v, want %v", x, y, cell.R, expectedR)
			}
		}
	}
}

func TestBuffer_Resize_SameSize(t *testing.T) {
	b := NewBuffer(10, 5)
	cellsPtr := &b.cells[0]

	b.Resize(10, 5)

	if &b.cells[0] != cellsPtr {
		t.Error("Resize() should not reallocate when size is unchanged")
	}
}

func TestBuffer_Clear(t *testing.T) {
	b := NewBuffer(10, 5)

	// Write some data
	for y := 0; y < 5; y++ {
		for x := 0; x < 10; x++ {
			cell := b.At(x, y)
			cell.R = 'X'
			cell.Style = tui.Style{FG: tui.ColorRed}
		}
	}

	// Clear with specific style
	clearStyle := tui.Style{FG: tui.ColorBlue, BG: tui.ColorWhite}
	clearCell := Cell{R: ' ', Style: clearStyle}
	b.Clear(clearCell)

	// Verify all cells are cleared
	for y := 0; y < 5; y++ {
		for x := 0; x < 10; x++ {
			cell := b.At(x, y)
			if cell.R != ' ' {
				t.Errorf("Clear() cell at (%d,%d) R = %v, want ' '", x, y, cell.R)
			}
			if cell.Style != clearStyle {
				t.Errorf("Clear() cell at (%d,%d) Style = %+v, want %+v", x, y, cell.Style, clearStyle)
			}
		}
	}
}
