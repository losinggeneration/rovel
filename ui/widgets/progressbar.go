package widgets

import (
	"github.com/losinggeneration/rovel"
	"github.com/losinggeneration/rovel/geom"
	"github.com/losinggeneration/rovel/style"
)

// ProgressBarOpts holds options for creating a ProgressBar.
type ProgressBarOpts struct {
	ID    rovel.ID
	Value float64 // 0.0 to 1.0
	Width int     // preferred width; 0 defaults to 20

	// Optional style overrides. When non-nil, the style replaces the
	// palette-derived style for that state completely (no merging).
	StyleFilled *style.Style
	StyleEmpty  *style.Style
}

// ProgressBar is a display-only horizontal progress indicator.
// It does not receive focus.
type ProgressBar struct {
	id    rovel.ID
	rect  rovel.Rect
	value float64 // clamped to [0, 1]
	width int

	stFilled *style.Style
	stEmpty  *style.Style
}

func NewProgressBar() *ProgressBar {
	return NewProgressBarOpts(ProgressBarOpts{})
}

func NewProgressBarOpts(opts ProgressBarOpts) *ProgressBar {
	id := opts.ID
	if id == 0 {
		id = rovel.NewID()
	}

	w := opts.Width
	if w <= 0 {
		w = 20
	}

	return &ProgressBar{
		id:       id,
		value:    clampf(opts.Value),
		width:    w,
		stFilled: opts.StyleFilled,
		stEmpty:  opts.StyleEmpty,
	}
}

func (b *ProgressBar) ID() rovel.ID        { return b.id }
func (b *ProgressBar) Rect() rovel.Rect    { return b.rect }
func (b *ProgressBar) Layout(r rovel.Rect) { b.rect = r }
func (b *ProgressBar) Focusable() bool     { return false }

func (b *ProgressBar) Value() float64 { return b.value }

func (b *ProgressBar) SetValue(ctx *rovel.Ctx, v float64) {
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

func (b *ProgressBar) Paint(d rovel.Drawer, ctx *rovel.Ctx) {
	r := b.rect
	if r.W <= 0 || r.H <= 0 {
		return
	}

	cd, ok := rovel.CellDrawerOf(d)
	if !ok {
		return
	}

	filled := int(b.value * float64(r.W))
	if filled > r.W {
		filled = r.W
	}

	y := r.Y + r.H/2

	filledFallback := ctx.Theme.Palette.Accent
	if filledFallback == (style.Style{}) {
		filledFallback = ctx.Theme.Base.WithAttr(style.AttrReverse)
	}

	filledSt := resolveStyle(b.stFilled, filledFallback)

	emptyFallback := ctx.Theme.Palette.SurfaceMuted
	if emptyFallback == (style.Style{}) {
		emptyFallback = ctx.Theme.Base
	}

	emptySt := resolveStyle(b.stEmpty, emptyFallback)

	for x := r.X; x < r.X+filled; x++ {
		cd.SetCell(x, y, '█', filledSt)
	}

	for x := r.X + filled; x < r.X+r.W; x++ {
		cd.SetCell(x, y, '░', emptySt)
	}
}

func (b *ProgressBar) Handle(_ rovel.Event, _ *rovel.Ctx) bool {
	return false
}

func clampf(v float64) float64 {
	return max(0.0, min(1.0, v))
}
