package widgets

import (
	"github.com/losinggeneration/tui"
	"github.com/losinggeneration/tui/event"
	"github.com/losinggeneration/tui/geom"
	"github.com/losinggeneration/tui/ui"
)

var (
	_ ui.Focusable = (*ScrollView)(nil)
	_ ui.Composite = (*ScrollView)(nil)
)

// ScrollbarMode controls scrollbar visibility on a ScrollView.
type ScrollbarMode uint8

const (
	ScrollbarAuto   ScrollbarMode = iota // show when content overflows (default)
	ScrollbarAlways                      // always reserve space
	ScrollbarHidden                      // never show
)

type ScrollViewOpts struct {
	ID        tui.ID
	Child     tui.View
	Focusable bool
	Scrollbar ScrollbarMode
}

type ScrollView struct {
	id            tui.ID
	child         tui.View
	rect          geom.Rect
	scrollY       int
	focusable     bool
	scrollbarMode ScrollbarMode
	scrollbar     *Scrollbar
	showScrollbar bool
}

func NewScrollView(opts ScrollViewOpts) *ScrollView {
	id := opts.ID
	if isZeroID(id) {
		id = tui.NewID()
	}

	return &ScrollView{
		id:            id,
		child:         opts.Child,
		focusable:     opts.Focusable,
		scrollbarMode: opts.Scrollbar,
	}
}

func (s *ScrollView) ID() tui.ID {
	return s.id
}

func (s *ScrollView) Rect() geom.Rect {
	return s.rect
}

func (s *ScrollView) Layout(r geom.Rect) {
	s.rect = r

	// Resolve scrollbar visibility
	switch s.scrollbarMode {
	case ScrollbarHidden:
		s.showScrollbar = false
	case ScrollbarAlways:
		s.showScrollbar = true
	default: // ScrollbarAuto
		s.showScrollbar = s.child != nil && s.child.MinSize().H > r.H
	}

	// Layout child — reduce width if scrollbar is visible
	if s.child != nil {
		childRect := r
		if s.showScrollbar && r.W > 1 {
			childRect.W = r.W - 1
		}
		s.child.Layout(childRect)
	}

	s.clampScroll()

	// Layout scrollbar
	if s.showScrollbar {
		if s.scrollbar == nil {
			s.scrollbar = NewScrollbar(ScrollbarOpts{
				OnScroll: func(pos int, ctx *tui.Ctx) {
					s.ScrollTo(ctx, pos)
				},
			})
		}
		s.scrollbar.Layout(geom.Rect{
			X: r.X + r.W - 1,
			Y: r.Y,
			W: 1,
			H: r.H,
		})
		s.scrollbar.SetState(s.contentHeight(), r.H, s.scrollY)
	}
}

func (s *ScrollView) MinSize() geom.Size {
	if s.child != nil {
		ms := s.child.MinSize()
		if s.scrollbarMode == ScrollbarAlways {
			ms.W++
		}
		return ms
	}
	return geom.Size{W: 1, H: 1}
}

func (s *ScrollView) Focusable() bool {
	return s.focusable
}

func (s *ScrollView) Children() []tui.View {
	if s.child == nil {
		return nil
	}
	return []tui.View{s.child}
}

func (s *ScrollView) Paint(p *tui.Painter, ctx *tui.Ctx) {
	if s.child == nil || s.rect.W <= 0 || s.rect.H <= 0 {
		return
	}

	// Clip region for child excludes scrollbar column
	clipRect := s.rect
	if s.showScrollbar && clipRect.W > 1 {
		clipRect.W--
	}

	p.WithClip(clipRect, func(cp *tui.Painter) {
		cp.WithOffset(0, -s.scrollY, func(op *tui.Painter) {
			s.child.Paint(op, ctx)
		})
	})

	// Paint scrollbar
	if s.showScrollbar && s.scrollbar != nil {
		s.scrollbar.SetState(s.contentHeight(), s.rect.H, s.scrollY)
		s.scrollbar.Paint(p, ctx)
	}
}

// HandleAction handles semantic actions.
func (s *ScrollView) HandleAction(act int, ctx *tui.Ctx) bool {
	switch ui.Action(act) {
	case ui.ActionMoveUp:
		s.ScrollBy(ctx, -1)
		return true
	case ui.ActionMoveDown:
		s.ScrollBy(ctx, 1)
		return true
	case ui.ActionPageUp:
		s.ScrollBy(ctx, -(s.rect.H - 1))
		return true
	case ui.ActionPageDown:
		s.ScrollBy(ctx, s.rect.H-1)
		return true
	case ui.ActionHome:
		s.ScrollTo(ctx, 0)
		return true
	case ui.ActionEnd:
		s.ScrollTo(ctx, s.maxScrollY())
		return true
	}
	return false
}

// MouseOpaque marks ScrollView as opaque to hit-testing. The app dispatches
// all mouse events for its rect here; ScrollView intercepts wheel events and
// delegates other mouse events to its child with scroll-adjusted coordinates.
func (s *ScrollView) MouseOpaque() {}

func (s *ScrollView) Handle(e tui.Event, ctx *tui.Ctx) bool {
	// Mouse handling — ScrollView is mouseOpaque, so it receives all mouse
	// events for its rect and must delegate non-wheel events to its child.
	if me, ok := e.(tui.MouseEvent); ok {
		switch me.Button {
		case tui.MouseButtonWheelUp:
			s.ScrollBy(ctx, -3)
			return true
		case tui.MouseButtonWheelDown:
			s.ScrollBy(ctx, 3)
			return true
		default:
			// Delegate to scrollbar if the click is in the scrollbar column
			if s.showScrollbar && s.scrollbar != nil {
				sbRect := s.scrollbar.Rect()
				if me.X >= sbRect.X && me.X < sbRect.X+sbRect.W {
					return s.scrollbar.Handle(me, ctx)
				}
				// Dragging: if scrollbar is in drag mode, delegate regardless of X
				if s.scrollbar.dragging {
					return s.scrollbar.Handle(me, ctx)
				}
			}

			if s.child != nil {
				adjusted := me
				adjusted.Y += s.scrollY
				if s.child.Handle(adjusted, ctx) {
					return true
				}
			}
			if me.Button == tui.MouseButtonLeft && me.Action == tui.MousePress && s.focusable {
				if ctx != nil && ctx.RequestFocus != nil {
					ctx.RequestFocus(s.id)
				}
				return true
			}
			return false
		}
	}

	ke, ok := e.(event.KeyEvent)
	if !ok {
		return false
	}

	switch ke.Key {
	case event.KeyUp:
		s.ScrollBy(ctx, -1)
		return true
	case event.KeyDown:
		s.ScrollBy(ctx, 1)
		return true
	case event.KeyPageUp:
		s.ScrollBy(ctx, -(s.rect.H - 1))
		return true
	case event.KeyPageDown:
		s.ScrollBy(ctx, s.rect.H-1)
		return true
	case event.KeyHome:
		s.ScrollTo(ctx, 0)
		return true
	case event.KeyEnd:
		s.ScrollTo(ctx, s.maxScrollY())
		return true
	}

	return false
}

func (s *ScrollView) ScrollPos() int {
	return s.scrollY
}

func (s *ScrollView) ScrollTo(ctx *tui.Ctx, y int) {
	old := s.scrollY
	s.scrollY = y
	s.clampScroll()
	if s.scrollY != old {
		ctx.Invalidate(s.rect)
	}
}

func (s *ScrollView) ScrollBy(ctx *tui.Ctx, delta int) {
	s.ScrollTo(ctx, s.scrollY+delta)
}

func (s *ScrollView) ScrollTop(ctx *tui.Ctx) {
	s.ScrollTo(ctx, 0)
}

func (s *ScrollView) ScrollBottom(ctx *tui.Ctx) {
	s.ScrollTo(ctx, s.maxScrollY())
}

func (s *ScrollView) contentHeight() int {
	if s.child == nil {
		return 0
	}
	return s.child.MinSize().H
}

func (s *ScrollView) maxScrollY() int {
	contentH := s.contentHeight()
	viewH := s.rect.H
	maxScroll := contentH - viewH
	if maxScroll < 0 {
		return 0
	}
	return maxScroll
}

func (s *ScrollView) clampScroll() {
	maxScroll := s.maxScrollY()
	if s.scrollY < 0 {
		s.scrollY = 0
	} else if s.scrollY > maxScroll {
		s.scrollY = maxScroll
	}
}
