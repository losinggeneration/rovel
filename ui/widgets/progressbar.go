package widgets

import (
	"github.com/losinggeneration/tui"
	"github.com/losinggeneration/tui/geom"
	"github.com/losinggeneration/tui/style"
)

// ProgressBarOpts holds options for creating a ProgressBar.
type ProgressBarOpts struct {
	ID    tui.ID
	Value float64 // 0.0 to 1.0
	Width int     // preferred width; 0 defaults to 20
}

// ProgressBar is a display-only horizontal progress indicator.
// It does not receive focus.
type ProgressBar struct {
	id    tui.ID
	rect  tui.Rect
	value float64 // clamped to [0, 1]
	width int
}

// NewProgressBar creates a new progress bar.
func NewProgressBar() *ProgressBar {
	return NewProgressBarOpts(ProgressBarOpts{})
}

// NewProgressBarOpts creates a new progress bar with options.
func NewProgressBarOpts(opts ProgressBarOpts) *ProgressBar {
	id := opts.ID
	if isZeroID(id) {
		id = tui.NewID()
	}
	w := opts.Width
	if w <= 0 {
		w = 20
	}
	return &ProgressBar{
		id:    id,
		value: clampf(opts.Value),
		width: w,
	}
}

func (b *ProgressBar) ID() tui.ID        { return b.id }
func (b *ProgressBar) Rect() tui.Rect    { return b.rect }
func (b *ProgressBar) Layout(r tui.Rect) { b.rect = r }
func (b *ProgressBar) Focusable() bool   { return false }

// Value returns the current progress (0.0 to 1.0).
func (b *ProgressBar) Value() float64 { return b.value }

// SetValue sets the progress value and invalidates.
func (b *ProgressBar) SetValue(ctx *tui.Ctx, v float64) {
	v = clampf(v)
	if b.value == v {
		return
	}
	b.value = v
	if ctx != nil {
		ctx.Invalidate(b.rect)
	}
}

func (b *ProgressBar) MinSize() geom.Size {
	return geom.Size{W: b.width, H: 1}
}

func (b *ProgressBar) Paint(p *tui.Painter, ctx *tui.Ctx) {
	r := b.rect
	if r.W <= 0 || r.H <= 0 {
		return
	}

	filled := int(b.value * float64(r.W))
	if filled > r.W {
		filled = r.W
	}

	y := r.Y + r.H/2

	filledSt := ctx.Theme.Palette.Accent
	if filledSt == (style.Style{}) {
		filledSt = ctx.Theme.Base.WithAttr(style.AttrReverse)
	}
	emptySt := ctx.Theme.Palette.SurfaceMuted
	if emptySt == (style.Style{}) {
		emptySt = ctx.Theme.Base
	}

	for x := r.X; x < r.X+filled; x++ {
		p.SetCell(x, y, '█', filledSt)
	}
	for x := r.X + filled; x < r.X+r.W; x++ {
		p.SetCell(x, y, '░', emptySt)
	}
}

func (b *ProgressBar) Handle(_ tui.Event, _ *tui.Ctx) bool {
	return false
}

func clampf(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
