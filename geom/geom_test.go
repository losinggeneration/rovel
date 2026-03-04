package geom

import "testing"

func TestRect_Contains(t *testing.T) {
	tests := []struct {
		name     string
		r        Rect
		p        Point
		expected bool
	}{
		{
			name:     "point inside",
			r:        Rect{X: 10, Y: 20, W: 100, H: 50},
			p:        Point{X: 50, Y: 40},
			expected: true,
		},
		{
			name:     "point at left edge",
			r:        Rect{X: 10, Y: 20, W: 100, H: 50},
			p:        Point{X: 10, Y: 40},
			expected: true,
		},
		{
			name:     "point at right edge (outside)",
			r:        Rect{X: 10, Y: 20, W: 100, H: 50},
			p:        Point{X: 110, Y: 40},
			expected: false,
		},
		{
			name:     "point above",
			r:        Rect{X: 10, Y: 20, W: 100, H: 50},
			p:        Point{X: 50, Y: 10},
			expected: false,
		},
		{
			name:     "point below",
			r:        Rect{X: 10, Y: 20, W: 100, H: 50},
			p:        Point{X: 50, Y: 80},
			expected: false,
		},
		{
			name:     "empty rect",
			r:        Rect{X: 10, Y: 20, W: 0, H: 50},
			p:        Point{X: 10, Y: 20},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.Contains(tt.p); got != tt.expected {
				t.Errorf("Rect.Contains() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRect_Intersect(t *testing.T) {
	tests := []struct {
		name     string
		r        Rect
		other    Rect
		expected Rect
	}{
		{
			name:     "overlapping rectangles",
			r:        Rect{X: 0, Y: 0, W: 10, H: 10},
			other:    Rect{X: 5, Y: 5, W: 10, H: 10},
			expected: Rect{X: 5, Y: 5, W: 5, H: 5},
		},
		{
			name:     "non-overlapping (other to right)",
			r:        Rect{X: 0, Y: 0, W: 10, H: 10},
			other:    Rect{X: 15, Y: 0, W: 10, H: 10},
			expected: Rect{X: 10, Y: 0, W: 0, H: 10},
		},
		{
			name:     "non-overlapping (other below)",
			r:        Rect{X: 0, Y: 0, W: 10, H: 10},
			other:    Rect{X: 0, Y: 15, W: 10, H: 10},
			expected: Rect{X: 0, Y: 10, W: 10, H: 0},
		},
		{
			name:     "one inside another",
			r:        Rect{X: 0, Y: 0, W: 20, H: 20},
			other:    Rect{X: 5, Y: 5, W: 5, H: 5},
			expected: Rect{X: 5, Y: 5, W: 5, H: 5},
		},
		{
			name:     "identical rectangles",
			r:        Rect{X: 10, Y: 10, W: 10, H: 10},
			other:    Rect{X: 10, Y: 10, W: 10, H: 10},
			expected: Rect{X: 10, Y: 10, W: 10, H: 10},
		},
		{
			name:     "empty first rect",
			r:        Rect{X: 0, Y: 0, W: 0, H: 10},
			other:    Rect{X: 5, Y: 5, W: 10, H: 10},
			expected: Rect{X: 0, Y: 5, W: 0, H: 5},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.Intersect(tt.other); got != tt.expected {
				t.Errorf("Rect.Intersect() = %+v, want %+v", got, tt.expected)
			}
		})
	}
}

func TestRect_Empty(t *testing.T) {
	tests := []struct {
		name     string
		r        Rect
		expected bool
	}{
		{
			name:     "zero width",
			r:        Rect{X: 0, Y: 0, W: 0, H: 10},
			expected: true,
		},
		{
			name:     "zero height",
			r:        Rect{X: 0, Y: 0, W: 10, H: 0},
			expected: true,
		},
		{
			name:     "negative width",
			r:        Rect{X: 0, Y: 0, W: -5, H: 10},
			expected: true,
		},
		{
			name:     "negative height",
			r:        Rect{X: 0, Y: 0, W: 10, H: -5},
			expected: true,
		},
		{
			name:     "valid rect",
			r:        Rect{X: 0, Y: 0, W: 10, H: 10},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.Empty(); got != tt.expected {
				t.Errorf("Rect.Empty() = %v, want %v", got, tt.expected)
			}
		})
	}
}
