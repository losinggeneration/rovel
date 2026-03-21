// Package geom provides basic geometric types for the tui library.
//
// These types are shared across render, backend, and the root tui package
// to avoid import cycles.
//
// This package is unstable before v0.1.0.
package geom

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
	x0 := max(r.X, other.X)
	x1 := min(r.X+r.W, other.X+other.W)
	y0 := max(r.Y, other.Y)
	y1 := min(r.Y+r.H, other.Y+other.H)

	return Rect{X: x0, Y: y0, W: max(0, x1-x0), H: max(0, y1-y0)}
}

// Empty returns true if the rectangle has zero or negative area.
func (r Rect) Empty() bool {
	return r.W <= 0 || r.H <= 0
}
