package render

import (
	"testing"

	"github.com/losinggeneration/rovel/geom"
)

func TestNewDamage(t *testing.T) {
	d := NewDamage(10, 5)

	if d.W != 10 {
		t.Errorf("NewDamage() W = %v, want 10", d.W)
	}

	if d.H != 5 {
		t.Errorf("NewDamage() H = %v, want 5", d.H)
	}

	if len(d.Rows) != 5 {
		t.Errorf("NewDamage() Rows length = %v, want 5", len(d.Rows))
	}
}

func TestDamage_IsEmpty(t *testing.T) {
	d := NewDamage(10, 5)

	if !d.IsEmpty() {
		t.Error("IsEmpty() should return true for new damage")
	}

	d.AddSpan(0, 0, 5)

	if d.IsEmpty() {
		t.Error("IsEmpty() should return false after adding span")
	}
}

func TestDamage_Clear(t *testing.T) {
	d := NewDamage(10, 5)

	d.AddSpan(0, 0, 5)
	d.AddSpan(1, 2, 8)

	d.Clear()

	if !d.IsEmpty() {
		t.Error("IsEmpty() should return true after Clear()")
	}
}

func TestDamage_Reset(t *testing.T) {
	d := NewDamage(10, 5)

	d.AddSpan(0, 0, 5)
	d.AddSpan(1, 2, 8)

	d.Reset(15, 8)

	if d.W != 15 {
		t.Errorf("Reset() W = %v, want 15", d.W)
	}

	if d.H != 8 {
		t.Errorf("Reset() H = %v, want 8", d.H)
	}

	if !d.IsEmpty() {
		t.Error("IsEmpty() should return true after Reset()")
	}
}

func TestDamage_Reset_SameSize(t *testing.T) {
	d := NewDamage(10, 5)

	d.AddSpan(0, 0, 5)
	rowsCap := cap(d.Rows)

	d.Reset(10, 5)

	if d.W != 10 {
		t.Errorf("Reset() W = %v, want 10", d.W)
	}

	if d.H != 5 {
		t.Errorf("Reset() H = %v, want 5", d.H)
	}

	if !d.IsEmpty() {
		t.Error("IsEmpty() should return true after Reset()")
	}

	if cap(d.Rows) != rowsCap {
		t.Error("Reset() should reuse capacity when size is unchanged")
	}
}

func TestDamage_AddRect(t *testing.T) {
	tests := []struct {
		name     string
		w, h     int
		rect     geom.Rect
		expected []RowSpans
	}{
		{
			name: "simple rect",
			w:    10,
			h:    10,
			rect: geom.Rect{X: 2, Y: 3, W: 4, H: 2},
			expected: []RowSpans{
				nil, nil, nil,
				{{X0: 2, X1: 6}},
				{{X0: 2, X1: 6}},
				nil, nil, nil, nil, nil, nil,
			},
		},
		{
			name: "empty rect (zero width)",
			w:    10,
			h:    10,
			rect: geom.Rect{X: 2, Y: 3, W: 0, H: 2},
			expected: []RowSpans{
				nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
			},
		},
		{
			name: "rect clamped to bounds",
			w:    10,
			h:    10,
			rect: geom.Rect{X: 5, Y: 5, W: 10, H: 10},
			expected: []RowSpans{
				nil, nil, nil, nil, nil,
				{{X0: 5, X1: 10}},
				{{X0: 5, X1: 10}},
				{{X0: 5, X1: 10}},
				{{X0: 5, X1: 10}},
				{{X0: 5, X1: 10}},
			},
		},
		{
			name: "rect completely out of bounds",
			w:    10,
			h:    10,
			rect: geom.Rect{X: 20, Y: 20, W: 5, H: 5},
			expected: []RowSpans{
				nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDamage(tt.w, tt.h)
			d.AddRect(tt.rect)

			for y := range tt.h {
				expected := tt.expected[y]
				actual := d.Rows[y]

				if len(expected) != len(actual) {
					t.Errorf("Row %d: got %d spans, want %d", y, len(actual), len(expected))

					continue
				}

				for i := range expected {
					if actual[i].X0 != expected[i].X0 || actual[i].X1 != expected[i].X1 {
						t.Errorf("Row %d span %d: got %+v, want %+v", y, i, actual[i], expected[i])
					}
				}
			}
		})
	}
}

func TestDamage_AddSpan(t *testing.T) {
	d := NewDamage(10, 5)

	// Add first span
	d.AddSpan(2, 3, 7)

	if len(d.Rows[2]) != 1 {
		t.Errorf("AddSpan() row 2 length = %v, want 1", len(d.Rows[2]))
	}

	if d.Rows[2][0].X0 != 3 || d.Rows[2][0].X1 != 7 {
		t.Errorf("AddSpan() row 2 span = %+v, want {X0:3, X1:7}", d.Rows[2][0])
	}

	// Add overlapping span
	d.AddSpan(2, 5, 9)

	if len(d.Rows[2]) != 1 {
		t.Errorf("AddSpan() overlapping should merge, got %d spans", len(d.Rows[2]))
	}

	if d.Rows[2][0].X0 != 3 || d.Rows[2][0].X1 != 9 {
		t.Errorf("AddSpan() merged span = %+v, want {X0:3, X1:9}", d.Rows[2][0])
	}

	// Add adjacent span (should merge)
	d.AddSpan(2, 9, 12)

	if len(d.Rows[2]) != 1 {
		t.Errorf("AddSpan() adjacent should merge, got %d spans", len(d.Rows[2]))
	}

	if d.Rows[2][0].X0 != 3 || d.Rows[2][0].X1 != 10 {
		t.Errorf("AddSpan() merged span = %+v, want {X0:3, X1:10}", d.Rows[2][0])
	}

	// Add non-adjacent span (before the existing span)
	// Span (0,1) and (3,10) are not adjacent (1 < 3-1=2)
	d.AddSpan(2, 0, 1)

	if len(d.Rows[2]) != 2 {
		t.Errorf("AddSpan() non-adjacent should not merge, got %d spans", len(d.Rows[2]))
	}
	// Spans should be sorted: first (0,1), then (3,10)
	if d.Rows[2][0].X0 != 0 || d.Rows[2][0].X1 != 1 {
		t.Errorf("AddSpan() first span = %+v, want {X0:0, X1:1}", d.Rows[2][0])
	}

	if d.Rows[2][1].X0 != 3 || d.Rows[2][1].X1 != 10 {
		t.Errorf("AddSpan() second span = %+v, want {X0:3, X1:10}", d.Rows[2][1])
	}
}

func TestDamage_AddSpan_OutOfBounds(t *testing.T) {
	d := NewDamage(10, 5)

	// Negative y
	d.AddSpan(-1, 0, 5)

	if !d.IsEmpty() {
		t.Error("AddSpan() with negative y should be ignored")
	}

	// Y >= H
	d.AddSpan(5, 0, 5)

	if !d.IsEmpty() {
		t.Error("AddSpan() with y >= H should be ignored")
	}

	// Add valid span then test x clamping
	d.AddSpan(2, -5, 15)

	if len(d.Rows[2]) != 1 {
		t.Errorf("AddSpan() with x out of bounds should be clamped, got %d spans", len(d.Rows[2]))
	}

	if d.Rows[2][0].X0 != 0 || d.Rows[2][0].X1 != 10 {
		t.Errorf("AddSpan() clamped span = %+v, want {X0:0, X1:10}", d.Rows[2][0])
	}
}

func TestClamp(t *testing.T) {
	tests := []struct {
		v, lo, hi, expected int
	}{
		{5, 0, 10, 5},
		{-5, 0, 10, 0},
		{15, 0, 10, 10},
		{0, 0, 10, 0},
		{10, 0, 10, 10},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			if got := clamp(tt.v, tt.lo, tt.hi); got != tt.expected {
				t.Errorf("clamp(%d, %d, %d) = %d, want %d", tt.v, tt.lo, tt.hi, got, tt.expected)
			}
		})
	}
}
