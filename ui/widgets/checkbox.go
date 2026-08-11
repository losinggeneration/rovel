package widgets

import (
	"github.com/losinggeneration/rovel"
	"github.com/losinggeneration/rovel/geom"
	"github.com/losinggeneration/rovel/style"
	"github.com/losinggeneration/rovel/text"
	"github.com/losinggeneration/rovel/ui"
)

// CheckboxOpts holds options for creating a Checkbox.
type CheckboxOpts struct {
	ID       rovel.ID
	Label    string
	Checked  bool
	Disabled bool
	OnChange func(checked bool, ctx *rovel.Ctx)

	// Optional style overrides. When non-nil, the style replaces the
	// palette-derived style for that state completely (no merging).
	StyleNormal   *style.Style
	StyleFocused  *style.Style
	StyleDisabled *style.Style
}

// Checkbox is a toggle widget that shows [x] or [ ] with a label.
type Checkbox struct {
	id       rovel.ID
	rect     rovel.Rect
	label    string
	checked  bool
	disabled bool
	onChange func(checked bool, ctx *rovel.Ctx)

	stNormal   *style.Style
	stFocused  *style.Style
	stDisabled *style.Style
}

func NewCheckbox(label string) *Checkbox {
	return NewCheckboxOpts(CheckboxOpts{Label: label})
}

func NewCheckboxOpts(opts CheckboxOpts) *Checkbox {
	id := opts.ID
	if id == 0 {
		id = rovel.NewID()
	}

	return &Checkbox{
		id:         id,
		label:      opts.Label,
		checked:    opts.Checked,
		disabled:   opts.Disabled,
		onChange:   opts.OnChange,
		stNormal:   opts.StyleNormal,
		stFocused:  opts.StyleFocused,
		stDisabled: opts.StyleDisabled,
	}
}

func (c *Checkbox) ID() rovel.ID        { return c.id }
func (c *Checkbox) Rect() rovel.Rect    { return c.rect }
func (c *Checkbox) Layout(r rovel.Rect) { c.rect = r }
func (c *Checkbox) Focusable() bool     { return !c.disabled }

func (c *Checkbox) Checked() bool { return c.checked }

func (c *Checkbox) SetChecked(ctx *rovel.Ctx, v bool) {
	if c.checked == v {
		return
	}

	c.checked = v
	if ctx != nil {
		ctx.Invalidate(c.rect)
	}
}

func (c *Checkbox) SetLabel(ctx *rovel.Ctx, s string) {
	if c.label == s {
		return
	}

	c.label = s
	if ctx != nil {
		ctx.Invalidate(c.rect)
	}
}

func (c *Checkbox) SetOnChange(fn func(bool, *rovel.Ctx)) { c.onChange = fn }

func (c *Checkbox) MinSize() geom.Size {
	// "[x] " + label
	return geom.Size{W: 4 + text.Width(c.label), H: 1}
}

func (c *Checkbox) Paint(d rovel.Drawer, ctx *rovel.Ctx) {
	r := c.rect
	if r.W <= 0 || r.H <= 0 {
		return
	}

	focused := ctx != nil && ctx.FocusedID == c.id
	st := c.style(ctx, focused)

	d.FillRect(r, st)

	indicator := "[ ] "
	if c.checked {
		indicator = "[x] "
	}

	y := r.Y + r.H/2
	x := r.X
	d.DrawText(rovel.Point{X: x, Y: y}, indicator, st)
	x += text.Width(indicator)

	if x < r.X+r.W {
		lbl := text.Truncate(c.label, r.W-4, false)
		d.DrawText(rovel.Point{X: x, Y: y}, lbl, st)
	}
}

func (c *Checkbox) Handle(e rovel.Event, ctx *rovel.Ctx) bool {
	if me, ok := e.(rovel.MouseEvent); ok {
		if me.Button == rovel.MouseButtonLeft && me.Action == rovel.MousePress && !c.disabled {
			if ctx != nil && ctx.RequestFocus != nil {
				ctx.RequestFocus(c.id)
			}

			c.toggle(ctx)

			return true
		}

		return false
	}

	ke, ok := e.(rovel.KeyEvent)
	if !ok {
		return false
	}

	if c.disabled {
		return false
	}

	if ke.Key == rovel.KeyEnter || (ke.Key == rovel.KeyRune && ke.Rune == ' ') {
		c.toggle(ctx)

		return true
	}

	return false
}

// HandleAction handles semantic actions.
func (c *Checkbox) HandleAction(act ui.Action, ctx *rovel.Ctx) bool {
	if c.disabled {
		return false
	}

	if act == ui.ActionActivate {
		c.toggle(ctx)

		return true
	}

	return false
}

func (c *Checkbox) toggle(ctx *rovel.Ctx) {
	c.checked = !c.checked
	if ctx != nil {
		ctx.Invalidate(c.rect)
	}

	if c.onChange != nil {
		c.onChange(c.checked, ctx)
	}
}

func (c *Checkbox) style(ctx *rovel.Ctx, focused bool) style.Style {
	if ctx == nil {
		return style.Style{}
	}

	if c.disabled {
		return resolveStyle(c.stDisabled, ctx.Theme.Palette.Disabled)
	}

	if focused {
		return resolveStyle(c.stFocused, ctx.Theme.Palette.Focus)
	}

	return resolveStyle(c.stNormal, ctx.Theme.Base)
}
