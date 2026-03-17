package render

import "github.com/losinggeneration/tui/geom"

type Span struct {
	X0 int // inclusive
	X1 int // exclusive
}

type RowSpans []Span

type Damage struct {
	W, H int
	Rows []RowSpans // len == H
}

func NewDamage(w, h int) *Damage {
	return &Damage{
		W:    w,
		H:    h,
		Rows: make([]RowSpans, h),
	}
}

func (d *Damage) Reset(w, h int) {
	d.W = w
	d.H = h
	if cap(d.Rows) >= h {
		d.Rows = d.Rows[:h]
		for i := range d.Rows {
			d.Rows[i] = d.Rows[i][:0]
		}
		return
	}
	d.Rows = make([]RowSpans, h)
}

func (d *Damage) Clear() {
	for i := range d.Rows {
		d.Rows[i] = d.Rows[i][:0]
	}
}

func (d *Damage) IsEmpty() bool {
	for i := range d.Rows {
		if len(d.Rows[i]) != 0 {
			return false
		}
	}
	return true
}

func (d *Damage) AddRect(r geom.Rect) {
	// Clamp to [0..d.W), [0..d.H) and expand to row spans.
	if r.W <= 0 || r.H <= 0 {
		return
	}

	x0 := clamp(r.X, 0, d.W)
	x1 := clamp(r.X+r.W, 0, d.W)
	y0 := clamp(r.Y, 0, d.H)
	y1 := clamp(r.Y+r.H, 0, d.H)

	if x0 >= x1 || y0 >= y1 {
		return
	}

	for y := y0; y < y1; y++ {
		d.AddSpan(y, x0, x1)
	}
}

func (d *Damage) AddSpan(y, x0, x1 int) {
	// Maintains row spans in sorted, merged (normalized) form.
	// Merges both overlaps and adjacent spans (recommended for fewer fragments).

	if y < 0 || y >= d.H {
		return
	}

	x0 = clamp(x0, 0, d.W)
	x1 = clamp(x1, 0, d.W)
	if x0 >= x1 {
		return
	}

	row := d.Rows[y]
	if len(row) == 0 {
		d.Rows[y] = append(row[:0], Span{X0: x0, X1: x1})
		return
	}

	newS := Span{X0: x0, X1: x1}

	// Fast path: append if after last and not adjacent/overlapping.
	last := row[len(row)-1]
	if last.X1 < newS.X0 {
		d.Rows[y] = append(row, newS)
		return
	}

	// General path: single-pass merge into a fresh slice.
	// We must not use row[:0] as the output because appending to the
	// shared backing array corrupts values that range is still reading.
	out := make([]Span, 0, len(row)+1)
	inserted := false

	for _, s := range row {
		// If current span is completely before new span.
		if s.X1 < newS.X0 {
			out = append(out, s)
			continue
		}

		// If new span is completely before current span (no overlap/adjacency).
		if newS.X1 < s.X0-1 {
			if !inserted {
				out = append(out, newS)
				inserted = true
			}
			out = append(out, s)
			continue
		}

		// Overlap or adjacency: merge.
		if s.X0 < newS.X0 {
			newS.X0 = s.X0
		}
		if s.X1 > newS.X1 {
			newS.X1 = s.X1
		}
	}

	if !inserted {
		out = append(out, newS)
	}

	d.Rows[y] = out
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
