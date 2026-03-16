package widgets

import (
	"github.com/losinggeneration/tui"
	"github.com/losinggeneration/tui/geom"
	"github.com/losinggeneration/tui/style"
	"github.com/losinggeneration/tui/text"
	"github.com/losinggeneration/tui/ui"
	"github.com/losinggeneration/tui/ui/overlay"
)

// SelectOpts holds options for creating a Select widget.
type SelectOpts struct {
	ID          tui.ID
	Items       []string
	Selected    int    // initial selection, -1 for none
	Placeholder string // shown when nothing selected
	OnChange    func(index int, ctx *tui.Ctx)
}

// Select is a dropdown widget. It renders as a single-line trigger showing
// the current selection. On activation it opens an overlay list; selecting
// an item closes the overlay.
type Select struct {
	id          tui.ID
	rect        tui.Rect
	items       []string
	selected    int
	placeholder string
	onChange    func(index int, ctx *tui.Ctx)
	overlayID   tui.ID // non-zero when dropdown is open
}

// NewSelect creates a new Select widget.
func NewSelect(items []string) *Select {
	return NewSelectOpts(SelectOpts{Items: items, Selected: -1})
}

// NewSelectOpts creates a new Select widget with options.
func NewSelectOpts(opts SelectOpts) *Select {
	id := opts.ID
	if isZeroID(id) {
		id = tui.NewID()
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
	}
}

func (s *Select) ID() tui.ID      { return s.id }
func (s *Select) Rect() tui.Rect  { return s.rect }
func (s *Select) Layout(r tui.Rect) { s.rect = r }
func (s *Select) Focusable() bool  { return true }

// Selected returns the current selection index, or -1.
func (s *Select) Selected() int { return s.selected }

// SetSelected changes the selection.
func (s *Select) SetSelected(ctx *tui.Ctx, idx int) {
	if idx < -1 || idx >= len(s.items) || idx == s.selected {
		return
	}
	s.selected = idx
	if ctx != nil {
		ctx.Invalidate(s.rect)
	}
}

// SetOnChange sets the change callback.
func (s *Select) SetOnChange(fn func(int, *tui.Ctx)) { s.onChange = fn }

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

func (s *Select) Paint(p *tui.Painter, ctx *tui.Ctx) {
	r := s.rect
	if r.W <= 0 || r.H <= 0 {
		return
	}

	focused := ctx != nil && ctx.FocusedID == s.id
	st := ctx.Theme.Base
	if focused {
		st = ctx.Theme.Palette.Focus
	}

	p.Fill(r, ' ', st)

	y := r.Y + r.H/2
	label := s.displayText()
	// Show dropdown indicator
	indicator := "v "
	p.Text(r.X, y, indicator, st)
	lbl := text.Truncate(label, r.W-2, false)
	p.Text(r.X+2, y, lbl, st)
}

func (s *Select) displayText() string {
	if s.selected >= 0 && s.selected < len(s.items) {
		return s.items[s.selected]
	}
	return s.placeholder
}

func (s *Select) Handle(e tui.Event, ctx *tui.Ctx) bool {
	if me, ok := e.(tui.MouseEvent); ok {
		if me.Button == tui.MouseButtonLeft && me.Action == tui.MousePress {
			if ctx != nil && ctx.RequestFocus != nil {
				ctx.RequestFocus(s.id)
			}
			s.openDropdown(ctx)
			return true
		}
		return false
	}

	ke, ok := e.(tui.KeyEvent)
	if !ok {
		return false
	}

	switch ke.Key {
	case tui.KeyEnter:
		s.openDropdown(ctx)
		return true
	case tui.KeyRune:
		if ke.Rune == ' ' {
			s.openDropdown(ctx)
			return true
		}
	}
	return false
}

// HandleAction handles semantic actions.
func (s *Select) HandleAction(act int, ctx *tui.Ctx) bool {
	if ui.Action(act) == ui.ActionActivate {
		s.openDropdown(ctx)
		return true
	}
	return false
}

func (s *Select) openDropdown(ctx *tui.Ctx) {
	if ctx == nil || ctx.ShowOverlay == nil || len(s.items) == 0 {
		return
	}

	list := newSelectList(s.items, s.selected, func(idx int, lctx *tui.Ctx) {
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

	o := ctx.ShowOverlay(tui.OverlayOpts{
		Root:  list,
		Modal: true,
		Place: overlay.Anchored{Anchor: s.rect},
	})
	s.overlayID = o.ID()
}

// selectList is the overlay content for Select — a simple focusable list.
type selectList struct {
	id       tui.ID
	rect     tui.Rect
	items    []string
	focused  int
	onSelect func(int, *tui.Ctx)
}

func newSelectList(items []string, initial int, onSelect func(int, *tui.Ctx)) *selectList {
	focused := initial
	if focused < 0 {
		focused = 0
	}
	return &selectList{
		id:       tui.NewID(),
		items:    items,
		focused:  focused,
		onSelect: onSelect,
	}
}

func (l *selectList) ID() tui.ID      { return l.id }
func (l *selectList) Rect() tui.Rect  { return l.rect }
func (l *selectList) Layout(r tui.Rect) { l.rect = r }
func (l *selectList) Focusable() bool  { return true }

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

func (l *selectList) Paint(p *tui.Painter, ctx *tui.Ctx) {
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

		p.Fill(geom.Rect{X: r.X, Y: y, W: r.W, H: 1}, ' ', st)
		lbl := text.Truncate(" "+item, r.W, false)
		p.Text(r.X, y, lbl, st)
	}
}

func (l *selectList) Handle(e tui.Event, ctx *tui.Ctx) bool {
	if me, ok := e.(tui.MouseEvent); ok {
		if me.Button == tui.MouseButtonLeft && me.Action == tui.MousePress {
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

	ke, ok := e.(tui.KeyEvent)
	if !ok {
		return false
	}

	switch ke.Key {
	case tui.KeyUp:
		if l.focused > 0 {
			l.focused--
			ctx.Invalidate(l.rect)
		}
		return true
	case tui.KeyDown:
		if l.focused < len(l.items)-1 {
			l.focused++
			ctx.Invalidate(l.rect)
		}
		return true
	case tui.KeyEnter:
		if l.onSelect != nil {
			l.onSelect(l.focused, ctx)
		}
		return true
	case tui.KeyRune:
		if ke.Rune == ' ' {
			if l.onSelect != nil {
				l.onSelect(l.focused, ctx)
			}
			return true
		}
	}
	return false
}

// HandleAction handles semantic actions for the select list.
func (l *selectList) HandleAction(act int, ctx *tui.Ctx) bool {
	switch ui.Action(act) {
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
	}
	return false
}
