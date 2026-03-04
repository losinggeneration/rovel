package tui

import (
	"sync/atomic"
)

// ID is a unique identifier.
type ID uint64

// idCounter is the global counter for generating unique IDs.
var idCounter uint64

// NewID generates a new unique ID.
func NewID() ID {
	return ID(atomic.AddUint64(&idCounter, 1))
}

// Point represents a 2D coordinate.
type Point struct {
	X, Y int
}

// Size represents a 2D size.
type Size struct {
	W, H int
}

// Rect represents a 2D rectangle.
type Rect struct {
	X, Y, W, H int
}

// Contains returns true if the point is inside the rectangle.
func (r Rect) Contains(p Point) bool {
	return p.X >= r.X && p.X < r.X+r.W &&
		p.Y >= r.Y && p.Y < r.Y+r.H
}

// Intersect returns the intersection of two rectangles.
// The result may be an empty rectangle.
func (r Rect) Intersect(other Rect) Rect {
	x0 := r.X
	if other.X > x0 {
		x0 = other.X
	}
	x1 := r.X + r.W
	if other.X+other.W < x1 {
		x1 = other.X + other.W
	}

	y0 := r.Y
	if other.Y > y0 {
		y0 = other.Y
	}
	y1 := r.Y + r.H
	if other.Y+other.H < y1 {
		y1 = other.Y + other.H
	}

	if x0 > x1 {
		x0 = x1
	}
	if y0 > y1 {
		y0 = y1
	}

	return Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}

// Empty returns true if the rectangle has zero or negative area.
func (r Rect) Empty() bool {
	return r.W <= 0 || r.H <= 0
}
