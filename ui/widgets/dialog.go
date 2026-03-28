package widgets

import (
	"github.com/losinggeneration/tui"
	"github.com/losinggeneration/tui/geom"
	"github.com/losinggeneration/tui/style"
	"github.com/losinggeneration/tui/text"
	"github.com/losinggeneration/tui/ui"
)

// DialogButton describes a button in a dialog's button row.
type DialogButton struct {
	Label   string
	OnPress func(ctx *tui.Ctx)
}

// DialogOpts holds options for creating a Dialog.
type DialogOpts struct {
	ID      tui.ID
	Title   string
	Message string
	Buttons []DialogButton
	Width   int // preferred width; 0 defaults to fit content

	// Optional style overrides. When non-nil, the style replaces the
	// palette-derived style for that state completely (no merging).
	StyleSurface *style.Style
	StyleBorder  *style.Style
}

// Dialog is a modal overlay panel with a title, message, and button row.
// It is intended to be shown via ctx.ShowOverlay with a placement strategy.
//
// Dialog implements tui.View and can serve as an overlay root. It manages
// its own internal focus for the button row.
type Dialog struct {
	id      tui.ID
	rect    tui.Rect
	title   string
	message string
	buttons []DialogButton
	focused int // which button has focus
	prefW   int

	stSurface *style.Style
	stBorder  *style.Style
}

func NewDialog(opts DialogOpts) *Dialog {
	id := opts.ID
	if id == 0 {
		id = tui.NewID()
	}

	w := opts.Width
	if w <= 0 {
		// Auto-size: max of title, message lines, and button row
		w = text.Width(opts.Title) + 4

		for _, line := range splitLines(opts.Message) {
			lw := text.Width(line) + 4
			if lw > w {
				w = lw
			}
		}

		btnW := 2 // left/right padding
		for _, b := range opts.Buttons {
			btnW += text.Width(b.Label) + 4 + 1 // "[ label ] "
		}

		if btnW > w {
			w = btnW
		}

		if w < 20 {
			w = 20
		}
	}

	return &Dialog{
		id:        id,
		title:     opts.Title,
		message:   opts.Message,
		buttons:   opts.Buttons,
		prefW:     w,
		stSurface: opts.StyleSurface,
		stBorder:  opts.StyleBorder,
	}
}

func (d *Dialog) ID() tui.ID        { return d.id }
func (d *Dialog) Rect() tui.Rect    { return d.rect }
func (d *Dialog) Layout(r tui.Rect) { d.rect = r }
func (d *Dialog) Focusable() bool   { return len(d.buttons) > 0 }

func (d *Dialog) MinSize() geom.Size {
	lines := splitLines(d.message)
	// title line + blank + message lines + blank + button row + border
	h := 1 + 1 + len(lines) + 1 + 1 + 2

	return geom.Size{W: d.prefW, H: h}
}

func (d *Dialog) PreferredSize() geom.Size {
	return d.MinSize()
}

func (d *Dialog) Paint(p *tui.Painter, ctx *tui.Ctx) {
	d.PaintDrawer(tui.NewDrawer(p), ctx)
}

func (d *Dialog) PaintDrawer(dr tui.Drawer, ctx *tui.Ctx) {
	r := d.rect
	if r.W <= 0 || r.H <= 0 {
		return
	}

	surfFallback := ctx.Theme.Palette.Surface
	if surfFallback == (style.Style{}) {
		surfFallback = ctx.Theme.Base
	}

	surfSt := resolveStyle(d.stSurface, surfFallback)

	borderFallback := ctx.Theme.Palette.Border
	if borderFallback == (style.Style{}) {
		borderFallback = surfSt
	}

	borderSt := resolveStyle(d.stBorder, borderFallback)

	// Clear and draw border
	dr.FillRect(r, surfSt)
	dr.DrawBorder(r, tui.BoxStyle{
		Glyphs: tui.BoxGlyphsLight,
		Edges:  tui.BoxEdgesAll,
		Style:  borderSt,
	})

	inner := geom.Rect{X: r.X + 1, Y: r.Y + 1, W: r.W - 2, H: r.H - 2}
	if inner.W <= 0 || inner.H <= 0 {
		return
	}

	// Title (centered, bold)
	if d.title != "" {
		titleSt := ctx.Theme.Palette.Text
		if titleSt == (style.Style{}) {
			titleSt = surfSt.WithAttr(style.AttrBold)
		}

		title := text.Truncate(d.title, inner.W, false)
		tx := inner.X + (inner.W-text.Width(title))/2
		dr.DrawText(tui.Point{X: tx, Y: inner.Y}, title, titleSt)
	}

	// Message
	lines := splitLines(d.message)

	msgY := inner.Y + 2
	for i, line := range lines {
		if msgY+i >= inner.Y+inner.H-1 {
			break
		}

		l := text.Truncate(line, inner.W, false)
		dr.DrawText(tui.Point{X: inner.X, Y: msgY + i}, l, surfSt)
	}

	// Button row at bottom of inner area
	btnY := inner.Y + inner.H - 1
	if btnY <= inner.Y {
		return
	}

	btnX := inner.X + 1

	for i, btn := range d.buttons {
		label := "[ " + btn.Label + " ]"

		w := text.Width(label)
		if btnX+w > inner.X+inner.W {
			break
		}

		var st style.Style
		if ctx.FocusedID == d.id && i == d.focused {
			st = ctx.Theme.Palette.Focus
		} else {
			st = surfSt
		}

		dr.DrawText(tui.Point{X: btnX, Y: btnY}, label, st)
		btnX += w + 1
	}
}

func (d *Dialog) Handle(e tui.Event, ctx *tui.Ctx) bool {
	if me, ok := e.(tui.MouseEvent); ok {
		if me.Button == tui.MouseButtonLeft && me.Action == tui.MousePress && len(d.buttons) > 0 {
			// Check if click is on the button row
			inner := geom.Rect{X: d.rect.X + 1, Y: d.rect.Y + 1, W: d.rect.W - 2, H: d.rect.H - 2}

			btnY := inner.Y + inner.H - 1
			if me.Y == btnY {
				// Walk button positions to find which was clicked
				btnX := inner.X + 1

				for i, btn := range d.buttons {
					label := "[ " + btn.Label + " ]"

					w := text.Width(label)
					if me.X >= btnX && me.X < btnX+w {
						d.focused = i
						d.pressButton(ctx)

						return true
					}

					btnX += w + 1
				}
			}
		}

		return true // modal dialog consumes all mouse events
	}

	ke, ok := e.(tui.KeyEvent)
	if !ok || len(d.buttons) == 0 {
		return false
	}

	switch ke.Key {
	case tui.KeyLeft:
		if d.focused > 0 {
			d.focused--
			ctx.Invalidate(d.rect)
		}

		return true
	case tui.KeyRight:
		if d.focused < len(d.buttons)-1 {
			d.focused++
			ctx.Invalidate(d.rect)
		}

		return true
	case tui.KeyTab:
		if ke.Mod == 0 {
			d.focused = (d.focused + 1) % len(d.buttons)
			ctx.Invalidate(d.rect)

			return true
		}
	case tui.KeyShiftTab:
		d.focused = (d.focused - 1 + len(d.buttons)) % len(d.buttons)
		ctx.Invalidate(d.rect)

		return true
	case tui.KeyEnter:
		d.pressButton(ctx)

		return true
	case tui.KeyRune:
		if ke.Rune == ' ' {
			d.pressButton(ctx)

			return true
		}
	default:
	}

	return false
}

// HandleAction handles semantic actions.
func (d *Dialog) HandleAction(act int, ctx *tui.Ctx) bool {
	if len(d.buttons) == 0 {
		return false
	}

	switch ui.Action(act) {
	case ui.ActionActivate:
		d.pressButton(ctx)

		return true
	case ui.ActionMoveLeft:
		if d.focused > 0 {
			d.focused--
			ctx.Invalidate(d.rect)
		}

		return true
	case ui.ActionMoveRight:
		if d.focused < len(d.buttons)-1 {
			d.focused++
			ctx.Invalidate(d.rect)
		}

		return true
	case ui.ActionFocusNext:
		d.focused = (d.focused + 1) % len(d.buttons)
		ctx.Invalidate(d.rect)

		return true
	case ui.ActionFocusPrev:
		d.focused = (d.focused - 1 + len(d.buttons)) % len(d.buttons)
		ctx.Invalidate(d.rect)

		return true
	default:
	}

	return false
}

func (d *Dialog) pressButton(ctx *tui.Ctx) {
	if d.focused < 0 || d.focused >= len(d.buttons) {
		return
	}

	if fn := d.buttons[d.focused].OnPress; fn != nil {
		fn(ctx)
	}
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}

	var lines []string

	start := 0

	for i := range len(s) {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}

	if start < len(s) {
		lines = append(lines, s[start:])
	}

	return lines
}
