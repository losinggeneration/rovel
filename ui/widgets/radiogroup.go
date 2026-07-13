package widgets

import (
	"github.com/losinggeneration/rovel"
	"github.com/losinggeneration/rovel/geom"
	"github.com/losinggeneration/rovel/style"
	"github.com/losinggeneration/rovel/text"
	"github.com/losinggeneration/rovel/ui"
)

// RadioGroupOpts holds options for creating a RadioGroup.
type RadioGroupOpts struct {
	ID       rovel.ID
	Items    []string
	Selected int // index of initially selected item, -1 for none
	Disabled bool
	OnChange func(index int, ctx *rovel.Ctx)

	// Optional style overrides. When non-nil, the style replaces the
	// palette-derived style for that state completely (no merging).
	StyleNormal   *style.Style
	StyleFocused  *style.Style
	StyleDisabled *style.Style
}

// RadioGroup is a group of mutually exclusive options.
// Renders vertically, one option per line: (o) label / ( ) label.
type RadioGroup struct {
	id       rovel.ID
	rect     rovel.Rect
	items    []string
	selected int
	focused  int // which item has internal focus (for arrow key nav)
	disabled bool
	onChange func(index int, ctx *rovel.Ctx)

	stNormal   *style.Style
	stFocused  *style.Style
	stDisabled *style.Style
}

func NewRadioGroup(items []string) *RadioGroup {
	return NewRadioGroupOpts(RadioGroupOpts{Items: items, Selected: -1})
}

func NewRadioGroupOpts(opts RadioGroupOpts) *RadioGroup {
	id := opts.ID
	if id == 0 {
		id = rovel.NewID()
	}

	sel := opts.Selected
	if sel < -1 || sel >= len(opts.Items) {
		sel = -1
	}

	return &RadioGroup{
		id:         id,
		items:      opts.Items,
		selected:   sel,
		focused:    max(sel, 0),
		disabled:   opts.Disabled,
		onChange:   opts.OnChange,
		stNormal:   opts.StyleNormal,
		stFocused:  opts.StyleFocused,
		stDisabled: opts.StyleDisabled,
	}
}

func (r *RadioGroup) ID() rovel.ID         { return r.id }
func (r *RadioGroup) Rect() rovel.Rect     { return r.rect }
func (r *RadioGroup) Layout(rr rovel.Rect) { r.rect = rr }
func (r *RadioGroup) Focusable() bool    { return !r.disabled && len(r.items) > 0 }

func (r *RadioGroup) Selected() int { return r.selected }

func (r *RadioGroup) SetSelected(ctx *rovel.Ctx, idx int) {
	if idx < -1 || idx >= len(r.items) {
		return
	}

	if r.selected == idx {
		return
	}

	r.selected = idx
	if ctx != nil {
		ctx.Invalidate(r.rect)
	}
}

func (r *RadioGroup) SetOnChange(fn func(int, *rovel.Ctx)) { r.onChange = fn }

func (r *RadioGroup) MinSize() geom.Size {
	maxW := 0

	for _, item := range r.items {
		w := 4 + text.Width(item) // "(o) " prefix
		if w > maxW {
			maxW = w
		}
	}

	return geom.Size{W: maxW, H: len(r.items)}
}

func (r *RadioGroup) Paint(d rovel.Drawer, ctx *rovel.Ctx) {
	rect := r.rect
	if rect.W <= 0 || rect.H <= 0 || len(r.items) == 0 {
		return
	}

	isFocused := ctx != nil && ctx.FocusedID == r.id

	for i, item := range r.items {
		if i >= rect.H {
			break
		}

		y := rect.Y + i

		isItemFocused := isFocused && i == r.focused
		st := r.itemStyle(ctx, isItemFocused)

		d.FillRect(geom.Rect{X: rect.X, Y: y, W: rect.W, H: 1}, st)

		indicator := "( ) "
		if i == r.selected {
			indicator = "(o) "
		}

		d.DrawText(rovel.Point{X: rect.X, Y: y}, indicator, st)

		if rect.W > 4 {
			lbl := text.Truncate(item, rect.W-4, false)
			d.DrawText(rovel.Point{X: rect.X + 4, Y: y}, lbl, st)
		}
	}
}

func (r *RadioGroup) Handle(e rovel.Event, ctx *rovel.Ctx) bool {
	if me, ok := e.(rovel.MouseEvent); ok {
		if me.Button == rovel.MouseButtonLeft && me.Action == rovel.MousePress && !r.disabled && len(r.items) > 0 {
			if ctx != nil && ctx.RequestFocus != nil {
				ctx.RequestFocus(r.id)
			}
			// Determine which item was clicked based on Y offset
			idx := me.Y - r.rect.Y
			if idx >= 0 && idx < len(r.items) {
				r.focused = idx
				r.selectFocused(ctx)
			}

			return true
		}

		return false
	}

	ke, ok := e.(rovel.KeyEvent)
	if !ok || r.disabled || len(r.items) == 0 {
		return false
	}

	switch ke.Key {
	case rovel.KeyUp:
		if r.focused > 0 {
			r.focused--
			ctx.Invalidate(r.rect)
		}

		return true
	case rovel.KeyDown:
		if r.focused < len(r.items)-1 {
			r.focused++
			ctx.Invalidate(r.rect)
		}

		return true
	case rovel.KeyEnter:
		r.selectFocused(ctx)

		return true
	case rovel.KeyRune:
		if ke.Rune == ' ' {
			r.selectFocused(ctx)

			return true
		}
	default:
	}

	return false
}

// HandleAction handles semantic actions.
func (r *RadioGroup) HandleAction(act ui.Action, ctx *rovel.Ctx) bool {
	if r.disabled || len(r.items) == 0 {
		return false
	}

	switch act {
	case ui.ActionActivate:
		r.selectFocused(ctx)

		return true
	case ui.ActionMoveUp:
		if r.focused > 0 {
			r.focused--
			ctx.Invalidate(r.rect)
		}

		return true
	case ui.ActionMoveDown:
		if r.focused < len(r.items)-1 {
			r.focused++
			ctx.Invalidate(r.rect)
		}

		return true
	default:
	}

	return false
}

func (r *RadioGroup) selectFocused(ctx *rovel.Ctx) {
	if r.focused < 0 || r.focused >= len(r.items) {
		return
	}

	old := r.selected

	r.selected = r.focused
	if ctx != nil {
		ctx.Invalidate(r.rect)
	}

	if r.selected != old && r.onChange != nil {
		r.onChange(r.selected, ctx)
	}
}

func (r *RadioGroup) itemStyle(ctx *rovel.Ctx, focused bool) style.Style {
	if ctx == nil {
		return style.Style{}
	}

	if r.disabled {
		return resolveStyle(r.stDisabled, ctx.Theme.Palette.Disabled)
	}

	if focused {
		return resolveStyle(r.stFocused, ctx.Theme.Palette.Focus)
	}

	return resolveStyle(r.stNormal, ctx.Theme.Base)
}
