package widgets

import (
	"github.com/losinggeneration/tui"
	"github.com/losinggeneration/tui/geom"
	"github.com/losinggeneration/tui/style"
	"github.com/losinggeneration/tui/text"
	"github.com/losinggeneration/tui/ui"
)

// ButtonOpts holds options for creating a Button.
type ButtonOpts struct {
	ID tui.ID

	Label string

	// Called when the button is "activated" (Enter or Space) while focused.
	OnPress func(ctx *tui.Ctx)

	Disabled bool

	// Chrome controls the visual "shape" of the button label.
	// If zero, the button chooses based on the theme aesthetic.
	Chrome ButtonChrome

	// Optional styles. If left as zero values, these defaults are used:
	// - Normal:   default fg/bg, no attrs
	// - Focused:  Normal + AttrReverse
	// - Disabled: Normal + AttrUnderline
	StyleNormal   style.Style
	StyleFocused  style.Style
	StyleDisabled style.Style
}

// Button is a clickable button widget.
type Button struct {
	id       tui.ID
	rect     tui.Rect
	label    string
	disabled bool
	onPress  func(ctx *tui.Ctx)

	stNormal   style.Style
	stFocused  style.Style
	stDisabled style.Style
	useTheme   bool // Use theme-based styles

	chrome ButtonChrome
}

// ButtonChrome controls the visual "chrome" of the button.
type ButtonChrome uint8

const (
	ButtonChromeAuto ButtonChrome = iota
	ButtonChromeBrackets
	ButtonChromeSolid
)

// NewButton creates a new button with the given label.
// For more control, use NewButtonOpts.
func NewButton(label string) *Button {
	return NewButtonOpts(ButtonOpts{Label: label})
}

// NewButtonOpts creates a new button with options.
func NewButtonOpts(opts ButtonOpts) *Button {
	id := opts.ID
	if id == (tui.ID(0)) {
		id = tui.NewID()
	}

	normal := opts.StyleNormal
	focused := opts.StyleFocused
	disabled := opts.StyleDisabled

	// If styles are zero, use theme-based derivation
	useTheme := normal == (style.Style{}) && focused == (style.Style{}) && disabled == (style.Style{})

	// Set default styles if not using theme
	if !useTheme {
		if normal == (style.Style{}) {
			normal = style.Style{
				FG:   style.ColorDefault,
				BG:   style.ColorDefault,
				Attr: 0,
			}
		}

		if focused == (style.Style{}) {
			focused = normal
			focused.Attr |= style.AttrReverse
		}

		if disabled == (style.Style{}) {
			disabled = normal
			disabled.Attr |= style.AttrUnderline
		}
	}

	return &Button{
		id:         id,
		label:      opts.Label,
		disabled:   opts.Disabled,
		onPress:    opts.OnPress,
		stNormal:   normal,
		stFocused:  focused,
		stDisabled: disabled,
		useTheme:   useTheme,
		chrome:     opts.Chrome,
	}
}

func (b *Button) ID() tui.ID { return b.id }

func (b *Button) Rect() tui.Rect { return b.rect }

func (b *Button) Layout(r tui.Rect) { b.rect = r }

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

// SetLabel sets the button's label and invalidates the rect.
func (b *Button) SetLabel(ctx *tui.Ctx, s string) {
	if b.label == s {
		return
	}
	b.label = s
	if ctx != nil {
		ctx.Invalidate(b.rect)
	}
}

// SetDisabled sets the disabled state and invalidates the rect.
func (b *Button) SetDisabled(ctx *tui.Ctx, v bool) {
	if b.disabled == v {
		return
	}
	b.disabled = v
	if ctx != nil {
		ctx.Invalidate(b.rect)
	}
}

// SetOnPress sets the callback function for when the button is pressed.
func (b *Button) SetOnPress(fn func(ctx *tui.Ctx)) {
	b.onPress = fn
}

func (b *Button) Paint(p *tui.Painter, ctx *tui.Ctx) {
	r := b.rect
	if r.W <= 0 || r.H <= 0 {
		return
	}

	focused := ctx != nil && ctx.FocusedID == b.id
	chrome := b.chrome
	if chrome == ButtonChromeAuto && ctx != nil {
		switch ctx.Theme.EffectiveAesthetic() {
		case tui.AestheticModern:
			chrome = ButtonChromeSolid
		default:
			chrome = ButtonChromeBrackets
		}
	}
	if chrome == ButtonChromeAuto {
		chrome = ButtonChromeBrackets
	}

	var st style.Style
	if b.useTheme {
		// Derive from theme roles
		if b.disabled {
			st = ctx.Theme.Palette.Disabled
		} else if focused {
			st = ctx.Theme.Palette.Focus
		} else {
			st = ctx.Theme.Palette.Surface
		}
		// Ensure non-zero style
		if st == (style.Style{}) {
			st = ctx.Theme.Base
		}
	} else {
		// Use explicit styles
		st = b.stNormal
		if b.disabled {
			st = b.stDisabled
		} else if focused {
			st = b.stFocused
		}
	}

	// Paint full rect (Paint Contract A already clears damaged spans, but this
	// keeps the widget visually self-contained when invalidated).
	p.Fill(r, ' ', st)

	y := r.Y + r.H/2

	text := b.renderText(r.W, chrome)
	x := r.X + (r.W-approxWidth(text))/2
	if x < r.X {
		x = r.X
	}

	p.Text(x, y, text, st)
}

func (b *Button) Handle(e tui.Event, ctx *tui.Ctx) bool {
	ke, ok := e.(tui.KeyEvent)
	if !ok {
		return false
	}

	if b.disabled {
		return false
	}

	switch ke.Key {
	case tui.KeyEnter:
		if b.onPress != nil {
			b.onPress(ctx)
		}
		if ctx != nil {
			ctx.Invalidate(b.rect)
		}
		return true

	case tui.KeyRune:
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
func (b *Button) HandleAction(act int, ctx *tui.Ctx) bool {
	if b.disabled {
		return false
	}
	switch ui.Action(act) {
	case ui.ActionActivate:
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

// Focusable returns true - buttons can receive focus.
func (b *Button) Focusable() bool {
	return true
}

func (b *Button) renderText(maxW int, chrome ButtonChrome) string {
	if maxW <= 0 {
		return ""
	}

	switch chrome {
	case ButtonChromeSolid:
		// Prefer a little padding when there is room.
		if maxW >= 3 {
			lbl := truncateRunes(b.label, maxW-2)
			return " " + lbl + " "
		}
		return truncateRunes(b.label, maxW)

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
		lbl := truncateRunes(b.label, maxLabel)
		return "[ " + lbl + " ]"
	}
}

func truncateRunes(s string, max int) string {
	if max <= 0 || s == "" {
		return ""
	}
	return text.Truncate(s, max, false)
}

func approxWidth(s string) int {
	return text.Width(s)
}
