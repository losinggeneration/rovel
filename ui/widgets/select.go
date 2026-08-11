package widgets

import (
	"github.com/losinggeneration/rovel"
	"github.com/losinggeneration/rovel/geom"
	"github.com/losinggeneration/rovel/style"
	"github.com/losinggeneration/rovel/text"
	"github.com/losinggeneration/rovel/ui"
	"github.com/losinggeneration/rovel/ui/overlay"
)

// SelectOpts holds options for creating a Select widget.
type SelectOpts struct {
	ID          rovel.ID
	Items       []string
	Selected    int    // initial selection, -1 for none
	Placeholder string // shown when nothing selected
	OnChange    func(index int, ctx *rovel.Ctx)

	// Optional style overrides. When non-nil, the style replaces the
	// palette-derived style for that state completely (no merging).
	StyleNormal  *style.Style
	StyleFocused *style.Style
}

// Select is a dropdown widget. It renders as a single-line trigger showing
// the current selection. On activation it opens an overlay list; selecting
// an item closes the overlay.
type Select struct {
	id          rovel.ID
	rect        rovel.Rect
	items       []string
	selected    int
	placeholder string
	onChange    func(index int, ctx *rovel.Ctx)
	overlayID   rovel.ID // non-zero when dropdown is open

	stNormal  *style.Style
	stFocused *style.Style
}

func NewSelect(items []string) *Select {
	return NewSelectOpts(SelectOpts{Items: items, Selected: -1})
}

func NewSelectOpts(opts SelectOpts) *Select {
	id := opts.ID
	if id == 0 {
		id = rovel.NewID()
	}

	sel := opts.Selected
	if sel < -1 || sel >= len(opts.Items) {
		sel = -1
	}

	ph := opts.Placeholder
	if ph == "" {
		ph = "-- select --"
	}

	return &Select{
		id:          id,
		items:       opts.Items,
		selected:    sel,
		placeholder: ph,
		onChange:    opts.OnChange,
		stNormal:    opts.StyleNormal,
		stFocused:   opts.StyleFocused,
	}
}

func (s *Select) ID() rovel.ID        { return s.id }
func (s *Select) Rect() rovel.Rect    { return s.rect }
func (s *Select) Layout(r rovel.Rect) { s.rect = r }
func (s *Select) Focusable() bool     { return true }

func (s *Select) Selected() int { return s.selected }

func (s *Select) SetSelected(ctx *rovel.Ctx, idx int) {
	if idx < -1 || idx >= len(s.items) || idx == s.selected {
		return
	}

	s.selected = idx
	if ctx != nil {
		ctx.Invalidate(s.rect)
	}
}

func (s *Select) SetOnChange(fn func(int, *rovel.Ctx)) { s.onChange = fn }

func (s *Select) MinSize() geom.Size {
	maxW := text.Width(s.placeholder)

	for _, item := range s.items {
		w := text.Width(item)
		if w > maxW {
			maxW = w
		}
	}

	return geom.Size{W: maxW + 4, H: 1} // "v " prefix + padding
}

func (s *Select) Paint(d rovel.Drawer, ctx *rovel.Ctx) {
	r := s.rect
	if r.W <= 0 || r.H <= 0 {
		return
	}

	focused := ctx != nil && ctx.FocusedID == s.id

	var st style.Style
	if focused {
		st = resolveStyle(s.stFocused, ctx.Theme.Palette.Focus)
	} else {
		st = resolveStyle(s.stNormal, ctx.Theme.Base)
	}

	d.FillRect(r, st)

	y := r.Y + r.H/2
	label := s.displayText()
	// Show dropdown indicator
	indicator := "v "
	d.DrawText(rovel.Point{X: r.X, Y: y}, indicator, st)
	lbl := text.Truncate(label, r.W-2, false)
	d.DrawText(rovel.Point{X: r.X + 2, Y: y}, lbl, st)
}

func (s *Select) Handle(e rovel.Event, ctx *rovel.Ctx) bool {
	if me, ok := e.(rovel.MouseEvent); ok {
		if me.Button == rovel.MouseButtonLeft && me.Action == rovel.MousePress {
			if ctx != nil && ctx.RequestFocus != nil {
				ctx.RequestFocus(s.id)
			}

			s.openDropdown(ctx)

			return true
		}

		return false
	}

	ke, ok := e.(rovel.KeyEvent)
	if !ok {
		return false
	}

	switch ke.Key {
	case rovel.KeyEnter:
		s.openDropdown(ctx)

		return true
	case rovel.KeyRune:
		if ke.Rune == ' ' {
			s.openDropdown(ctx)

			return true
		}
	default:
	}

	return false
}

// HandleAction handles semantic actions.
func (s *Select) HandleAction(act ui.Action, ctx *rovel.Ctx) bool {
	if act == ui.ActionActivate {
		s.openDropdown(ctx)

		return true
	}

	return false
}

func (s *Select) displayText() string {
	if s.selected >= 0 && s.selected < len(s.items) {
		return s.items[s.selected]
	}

	return s.placeholder
}

func (s *Select) openDropdown(ctx *rovel.Ctx) {
	if ctx == nil || ctx.ShowOverlay == nil || len(s.items) == 0 {
		return
	}

	list := newSelectList(s.items, s.selected, func(idx int, lctx *rovel.Ctx) {
		old := s.selected
		s.selected = idx

		if lctx != nil {
			// Dismiss overlay
			if lctx.DismissOverlay != nil {
				lctx.DismissOverlay()
			}

			lctx.Invalidate(s.rect)
		}

		if s.selected != old && s.onChange != nil {
			s.onChange(s.selected, lctx)
		}
	})

	o := ctx.ShowOverlay(rovel.OverlayOpts{
		Root:  list,
		Modal: true,
		Place: overlay.Anchored{Anchor: s.rect},
	})
	s.overlayID = o.ID()
}

// selectList is the overlay content for Select — a simple focusable list.
type selectList struct {
	id       rovel.ID
	rect     rovel.Rect
	items    []string
	focused  int
	onSelect func(int, *rovel.Ctx)
}

func newSelectList(items []string, initial int, onSelect func(int, *rovel.Ctx)) *selectList {
	focused := initial
	if focused < 0 {
		focused = 0
	}

	return &selectList{
		id:       rovel.NewID(),
		items:    items,
		focused:  focused,
		onSelect: onSelect,
	}
}

func (l *selectList) ID() rovel.ID        { return l.id }
func (l *selectList) Rect() rovel.Rect    { return l.rect }
func (l *selectList) Layout(r rovel.Rect) { l.rect = r }
func (l *selectList) Focusable() bool     { return true }

func (l *selectList) MinSize() geom.Size {
	maxW := 0

	for _, item := range l.items {
		w := text.Width(item) + 2
		if w > maxW {
			maxW = w
		}
	}

	return geom.Size{W: maxW, H: len(l.items)}
}

func (l *selectList) PreferredSize() geom.Size {
	return l.MinSize()
}

func (l *selectList) Paint(d rovel.Drawer, ctx *rovel.Ctx) {
	r := l.rect
	if r.W <= 0 || r.H <= 0 {
		return
	}

	surfSt := ctx.Theme.Palette.Surface
	if surfSt == (style.Style{}) {
		surfSt = ctx.Theme.Base
	}

	for i, item := range l.items {
		if i >= r.H {
			break
		}

		y := r.Y + i

		var st style.Style
		if i == l.focused {
			st = ctx.Theme.Palette.Focus
		} else {
			st = surfSt
		}

		d.FillRect(geom.Rect{X: r.X, Y: y, W: r.W, H: 1}, st)
		lbl := text.Truncate(" "+item, r.W, false)
		d.DrawText(rovel.Point{X: r.X, Y: y}, lbl, st)
	}
}

func (l *selectList) Handle(e rovel.Event, ctx *rovel.Ctx) bool {
	if me, ok := e.(rovel.MouseEvent); ok {
		if me.Button == rovel.MouseButtonLeft && me.Action == rovel.MousePress {
			idx := me.Y - l.rect.Y
			if idx >= 0 && idx < len(l.items) {
				l.focused = idx
				if l.onSelect != nil {
					l.onSelect(idx, ctx)
				}
			}

			return true
		}

		return false
	}

	ke, ok := e.(rovel.KeyEvent)
	if !ok {
		return false
	}

	switch ke.Key {
	case rovel.KeyEsc:
		// The dropdown is a modal overlay, so without this the list would be
		// a keyboard trap in apps that don't configure ResolveAction (whose
		// Cancel handling otherwise dismisses the overlay at the app level).
		if ctx != nil && ctx.DismissOverlay != nil {
			ctx.DismissOverlay()

			return true
		}

		return false
	case rovel.KeyUp:
		if l.focused > 0 {
			l.focused--
			ctx.Invalidate(l.rect)
		}

		return true
	case rovel.KeyDown:
		if l.focused < len(l.items)-1 {
			l.focused++
			ctx.Invalidate(l.rect)
		}

		return true
	case rovel.KeyEnter:
		if l.onSelect != nil {
			l.onSelect(l.focused, ctx)
		}

		return true
	case rovel.KeyRune:
		if ke.Rune == ' ' {
			if l.onSelect != nil {
				l.onSelect(l.focused, ctx)
			}

			return true
		}
	default:
	}

	return false
}

// HandleAction handles semantic actions for the select list.
func (l *selectList) HandleAction(act ui.Action, ctx *rovel.Ctx) bool {
	switch act {
	case ui.ActionActivate:
		if l.onSelect != nil {
			l.onSelect(l.focused, ctx)
		}

		return true
	case ui.ActionMoveUp:
		if l.focused > 0 {
			l.focused--
			ctx.Invalidate(l.rect)
		}

		return true
	case ui.ActionMoveDown:
		if l.focused < len(l.items)-1 {
			l.focused++
			ctx.Invalidate(l.rect)
		}

		return true
	default:
	}

	return false
}
