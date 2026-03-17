package widgets

import (
	"github.com/losinggeneration/tui"
	"github.com/losinggeneration/tui/geom"
	"github.com/losinggeneration/tui/style"
	"github.com/losinggeneration/tui/text"
	"github.com/losinggeneration/tui/ui"
)

// RadioGroupOpts holds options for creating a RadioGroup.
type RadioGroupOpts struct {
	ID       tui.ID
	Items    []string
	Selected int // index of initially selected item, -1 for none
	Disabled bool
	OnChange func(index int, ctx *tui.Ctx)
}

// RadioGroup is a group of mutually exclusive options.
// Renders vertically, one option per line: (o) label / ( ) label.
type RadioGroup struct {
	id       tui.ID
	rect     tui.Rect
	items    []string
	selected int
	focused  int // which item has internal focus (for arrow key nav)
	disabled bool
	onChange func(index int, ctx *tui.Ctx)
}

// NewRadioGroup creates a new radio group with the given items.
func NewRadioGroup(items []string) *RadioGroup {
	return NewRadioGroupOpts(RadioGroupOpts{Items: items, Selected: -1})
}

// NewRadioGroupOpts creates a new radio group with options.
func NewRadioGroupOpts(opts RadioGroupOpts) *RadioGroup {
	id := opts.ID
	if isZeroID(id) {
		id = tui.NewID()
	}
	sel := opts.Selected
	if sel < -1 || sel >= len(opts.Items) {
		sel = -1
	}
	return &RadioGroup{
		id:       id,
		items:    opts.Items,
		selected: sel,
		focused:  max(sel, 0),
		disabled: opts.Disabled,
		onChange: opts.OnChange,
	}
}

func (r *RadioGroup) ID() tui.ID         { return r.id }
func (r *RadioGroup) Rect() tui.Rect     { return r.rect }
func (r *RadioGroup) Layout(rr tui.Rect) { r.rect = rr }
func (r *RadioGroup) Focusable() bool    { return !r.disabled && len(r.items) > 0 }

// Selected returns the index of the selected item, or -1.
func (r *RadioGroup) Selected() int { return r.selected }

// SetSelected sets the selected index.
func (r *RadioGroup) SetSelected(ctx *tui.Ctx, idx int) {
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

// SetOnChange sets the change callback.
func (r *RadioGroup) SetOnChange(fn func(int, *tui.Ctx)) { r.onChange = fn }

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

func (r *RadioGroup) Paint(p *tui.Painter, ctx *tui.Ctx) {
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

		p.Fill(geom.Rect{X: rect.X, Y: y, W: rect.W, H: 1}, ' ', st)

		indicator := "( ) "
		if i == r.selected {
			indicator = "(o) "
		}
		p.Text(rect.X, y, indicator, st)

		if rect.W > 4 {
			lbl := text.Truncate(item, rect.W-4, false)
			p.Text(rect.X+4, y, lbl, st)
		}
	}
}

func (r *RadioGroup) Handle(e tui.Event, ctx *tui.Ctx) bool {
	if me, ok := e.(tui.MouseEvent); ok {
		if me.Button == tui.MouseButtonLeft && me.Action == tui.MousePress && !r.disabled && len(r.items) > 0 {
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

	ke, ok := e.(tui.KeyEvent)
	if !ok || r.disabled || len(r.items) == 0 {
		return false
	}

	switch ke.Key {
	case tui.KeyUp:
		if r.focused > 0 {
			r.focused--
			ctx.Invalidate(r.rect)
		}
		return true
	case tui.KeyDown:
		if r.focused < len(r.items)-1 {
			r.focused++
			ctx.Invalidate(r.rect)
		}
		return true
	case tui.KeyEnter:
		r.selectFocused(ctx)
		return true
	case tui.KeyRune:
		if ke.Rune == ' ' {
			r.selectFocused(ctx)
			return true
		}
	}
	return false
}

// HandleAction handles semantic actions.
func (r *RadioGroup) HandleAction(act int, ctx *tui.Ctx) bool {
	if r.disabled || len(r.items) == 0 {
		return false
	}
	switch ui.Action(act) {
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
	}
	return false
}

func (r *RadioGroup) selectFocused(ctx *tui.Ctx) {
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

func (r *RadioGroup) itemStyle(ctx *tui.Ctx, focused bool) style.Style {
	if ctx == nil {
		return style.Style{}
	}
	if r.disabled {
		return ctx.Theme.Palette.Disabled
	}
	if focused {
		return ctx.Theme.Palette.Focus
	}
	return ctx.Theme.Base
}
