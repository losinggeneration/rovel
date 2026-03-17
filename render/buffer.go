package render

// Buffer represents a grid of cells.
type Buffer struct {
	W, H  int
	cells []Cell // row-major: index = y*W + x
}

// NewBuffer creates a new buffer with the given dimensions.
func NewBuffer(w, h int) *Buffer {
	return &Buffer{
		W:     w,
		H:     h,
		cells: make([]Cell, w*h),
	}
}

// At returns a pointer to the cell at (x, y).
// If coordinates are out of bounds, returns a pointer to a zero cell.
func (b *Buffer) At(x, y int) *Cell {
	if x < 0 || x >= b.W || y < 0 || y >= b.H {
		return &zeroCell
	}

	return &b.cells[y*b.W+x]
}

// Resize resizes the buffer, preserving content in the overlapping region.
func (b *Buffer) Resize(w, h int) {
	if w == b.W && h == b.H {
		return
	}

	newCells := make([]Cell, w*h)

	// Copy overlapping content
	copyW := w
	if copyW > b.W {
		copyW = b.W
	}

	copyH := h
	if copyH > b.H {
		copyH = b.H
	}

	for y := 0; y < copyH; y++ {
		src := b.cells[y*b.W : y*b.W+copyW]
		dst := newCells[y*w : y*w+copyW]
		copy(dst, src)
	}

	b.W = w
	b.H = h
	b.cells = newCells
}

// Clear sets all cells in the buffer to the given cell.
func (b *Buffer) Clear(cell Cell) {
	for i := range b.cells {
		b.cells[i] = cell
	}
}
