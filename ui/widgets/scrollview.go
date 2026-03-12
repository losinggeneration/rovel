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

type ScrollViewOpts struct {
	ID        tui.ID
	Child     tui.View
	Focusable bool
}

type ScrollView struct {
	id        tui.ID
	child     tui.View
	rect      geom.Rect
	scrollY   int
	focusable bool
}

func NewScrollView(opts ScrollViewOpts) *ScrollView {
	id := opts.ID
	if isZeroID(id) {
		id = tui.NewID()
	}

	return &ScrollView{
		id:        id,
		child:     opts.Child,
		focusable: opts.Focusable,
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
	if s.child != nil {
		s.child.Layout(r)
	}
	s.clampScroll()
}

func (s *ScrollView) MinSize() geom.Size {
	if s.child != nil {
		return s.child.MinSize()
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

	p.WithClip(s.rect, func(cp *tui.Painter) {
		cp.WithOffset(0, -s.scrollY, func(op *tui.Painter) {
			s.child.Paint(op, ctx)
		})
	})
}

func (s *ScrollView) Handle(e tui.Event, ctx *tui.Ctx) bool {
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
	maxScroll := contentH - s.rect.H
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
