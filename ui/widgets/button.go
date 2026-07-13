package widgets

import (
	"github.com/losinggeneration/rovel"
	"github.com/losinggeneration/rovel/geom"
	"github.com/losinggeneration/rovel/style"
	"github.com/losinggeneration/rovel/text"
	"github.com/losinggeneration/rovel/ui"
)

// ButtonOpts holds options for creating a Button.
type ButtonOpts struct {
	ID rovel.ID

	Label string

	// Called when the button is "activated" (Enter or Space) while focused.
	OnPress func(ctx *rovel.Ctx)

	Disabled bool

	// Chrome controls the visual "shape" of the button label.
	// If zero, the button chooses based on the theme aesthetic.
	Chrome ButtonChrome

	// Optional style overrides. When non-nil, the style replaces the
	// palette-derived style for that state completely (no merging).
	// When nil, the widget falls back to the theme palette.
	StyleNormal   *style.Style
	StyleFocused  *style.Style
	StyleDisabled *style.Style
}

// Button is a clickable button widget.
type Button struct {
	id       rovel.ID
	rect     rovel.Rect
	label    string
	disabled bool
	onPress  func(ctx *rovel.Ctx)

	stNormal   *style.Style
	stFocused  *style.Style
	stDisabled *style.Style

	chrome ButtonChrome
}

// ButtonChrome controls the visual "chrome" of the button.
type ButtonChrome uint8

const (
	ButtonChromeAuto ButtonChrome = iota
	ButtonChromeBrackets
	ButtonChromeSolid
)

func NewButton(label string) *Button {
	return NewButtonOpts(ButtonOpts{Label: label})
}

func NewButtonOpts(opts ButtonOpts) *Button {
	id := opts.ID
	if id == 0 {
		id = rovel.NewID()
	}

	return &Button{
		id:         id,
		label:      opts.Label,
		disabled:   opts.Disabled,
		onPress:    opts.OnPress,
		stNormal:   opts.StyleNormal,
		stFocused:  opts.StyleFocused,
		stDisabled: opts.StyleDisabled,
		chrome:     opts.Chrome,
	}
}

func (b *Button) ID() rovel.ID { return b.id }

func (b *Button) Rect() rovel.Rect { return b.rect }

func (b *Button) Layout(r rovel.Rect) { b.rect = r }

// MinSize returns the minimum size needed for the button.
// MinSize reserves 4 extra columns for button chrome/padding.
// Classic bracket chrome is: "[ " + label + " ]".
// Note: rune-count is an approximation for wide chars; good enough for MVP.
func (b *Button) MinSize() geom.Size {
	w := 4 + text.Width(b.label)
	if w < 4 {
		w = 4
	}

	return geom.Size{W: w, H: 1}
}

func (b *Button) PreferredSize() geom.Size {
	return b.MinSize()
}

func (b *Button) SetLabel(ctx *rovel.Ctx, s string) {
	if b.label == s {
		return
	}

	b.label = s
	if ctx != nil {
		ctx.Invalidate(b.rect)
	}
}

func (b *Button) SetDisabled(ctx *rovel.Ctx, v bool) {
	if b.disabled == v {
		return
	}

	b.disabled = v
	if ctx != nil {
		ctx.Invalidate(b.rect)
	}
}

func (b *Button) SetOnPress(fn func(ctx *rovel.Ctx)) {
	b.onPress = fn
}

func (b *Button) Paint(d rovel.Drawer, ctx *rovel.Ctx) {
	r := b.rect
	if r.W <= 0 || r.H <= 0 {
		return
	}

	focused := ctx != nil && ctx.FocusedID == b.id

	chrome := b.chrome
	if chrome == ButtonChromeAuto && ctx != nil {
		switch ctx.Theme.EffectiveAesthetic() {
		case rovel.AestheticModern:
			chrome = ButtonChromeSolid
		case rovel.AestheticClassic:
		default:
			chrome = ButtonChromeBrackets
		}
	}

	if chrome == ButtonChromeAuto {
		chrome = ButtonChromeBrackets
	}

	var st style.Style

	switch {
	case b.disabled:
		fallback := ctx.Theme.Palette.Disabled
		if fallback == (style.Style{}) {
			fallback = ctx.Theme.Base
		}

		st = resolveStyle(b.stDisabled, fallback)

	case focused:
		fallback := ctx.Theme.Palette.Focus
		if fallback == (style.Style{}) {
			fallback = ctx.Theme.Base
		}

		st = resolveStyle(b.stFocused, fallback)

	default:
		fallback := ctx.Theme.Palette.Surface
		if fallback == (style.Style{}) {
			fallback = ctx.Theme.Base
		}

		st = resolveStyle(b.stNormal, fallback)
	}

	// Paint full rect (Paint Contract A already clears damaged spans, but this
	// keeps the widget visually self-contained when invalidated).
	d.FillRect(r, st)

	y := r.Y + r.H/2

	label := b.renderText(r.W, chrome)

	x := r.X + (r.W-text.Width(label))/2
	if x < r.X {
		x = r.X
	}

	d.DrawText(rovel.Point{X: x, Y: y}, label, st)
}

func (b *Button) Handle(e rovel.Event, ctx *rovel.Ctx) bool {
	if me, ok := e.(rovel.MouseEvent); ok {
		if me.Button == rovel.MouseButtonLeft && me.Action == rovel.MousePress && !b.disabled {
			if ctx != nil && ctx.RequestFocus != nil {
				ctx.RequestFocus(b.id)
			}

			if b.onPress != nil {
				b.onPress(ctx)
			}

			if ctx != nil {
				ctx.Invalidate(b.rect)
			}

			return true
		}

		return false
	}

	ke, ok := e.(rovel.KeyEvent)
	if !ok {
		return false
	}

	if b.disabled {
		return false
	}

	switch ke.Key {
	case rovel.KeyEnter:
		if b.onPress != nil {
			b.onPress(ctx)
		}

		if ctx != nil {
			ctx.Invalidate(b.rect)
		}

		return true

	case rovel.KeyRune:
		// Space activates buttons in many TUIs.
		if ke.Rune == ' ' {
			if b.onPress != nil {
				b.onPress(ctx)
			}

			if ctx != nil {
				ctx.Invalidate(b.rect)
			}

			return true
		}

		return false

	default:
		return false
	}
}

// HandleAction handles semantic actions.
func (b *Button) HandleAction(act ui.Action, ctx *rovel.Ctx) bool {
	if b.disabled {
		return false
	}

	if act == ui.ActionActivate {
		if b.onPress != nil {
			b.onPress(ctx)
		}

		if ctx != nil {
			ctx.Invalidate(b.rect)
		}

		return true
	}

	return false
}

func (b *Button) Focusable() bool {
	return true
}

// resolveStyle returns the override style if non-nil, otherwise the fallback.
func resolveStyle(override *style.Style, fallback style.Style) style.Style {
	if override != nil {
		return *override
	}

	return fallback
}

func (b *Button) renderText(maxW int, chrome ButtonChrome) string {
	if maxW <= 0 {
		return ""
	}

	switch chrome {
	case ButtonChromeSolid:
		// Prefer a little padding when there is room.
		if maxW >= 3 {
			lbl := text.Truncate(b.label, maxW-2, false)

			return " " + lbl + " "
		}

		return text.Truncate(b.label, maxW, false)

	default:
		// Desired: "[ " + label + " ]"
		// Layout for small widths:
		// 1: "["
		// 2: "[]"
		// 3: "[ ]"
		// 4+: "[ " + label(truncated) + " ]"
		if maxW == 1 {
			return "["
		}

		if maxW == 2 {
			return "[]"
		}

		if maxW == 3 {
			return "[ ]"
		}

		maxLabel := maxW - 4
		lbl := text.Truncate(b.label, maxLabel, false)

		return "[ " + lbl + " ]"
	}
}
