package layout

import (
	"github.com/losinggeneration/tui"
	"github.com/losinggeneration/tui/event"
	"github.com/losinggeneration/tui/ui"
)

type VStack struct {
	id       tui.ID
	children []Child
	rect     tui.Rect
	gap      int
}

func NewVStack() *VStack {
	return &VStack{
		id: tui.NewID(),
	}
}

func NewVStackWithChildren(children []Child) *VStack {
	s := NewVStack()
	s.children = children

	return s
}

func NewVStackWithGap(children []Child, gap int) *VStack {
	s := NewVStackWithChildren(children)
	s.gap = gap

	return s
}

func (s *VStack) Add(v tui.View) {
	s.children = append(s.children, NewChild(v))
}

func (s *VStack) AddChild(c Child) {
	s.children = append(s.children, c)
}

func (s *VStack) SetGap(gap int) {
	s.gap = gap
}

func (s *VStack) ID() tui.ID {
	return s.id
}

func (s *VStack) Rect() tui.Rect {
	return s.rect
}

func (s *VStack) Layout(r tui.Rect) {
	s.rect = r

	if len(s.children) == 0 {
		return
	}

	n := len(s.children)

	totalGap := s.gap * (n - 1)
	if totalGap < 0 {
		totalGap = 0
	}

	type childInfo struct {
		view      tui.View
		opts      SizePolicy
		minH      int
		prefH     int
		allocated int
	}

	infos := make([]childInfo, n)
	totalMinH := 0
	totalPrefH := 0
	maxW := 0
	totalStretch := 0

	for i, c := range s.children {
		info := &infos[i]
		info.view = c.View
		info.opts = c.Opts

		minSz := c.View.MinSize()

		info.minH = minSz.H

		if minSz.W > maxW {
			maxW = minSz.W
		}

		if ps, ok := c.View.(ui.PreferredSizer); ok {
			info.prefH = ps.PreferredSize().H
		} else {
			info.prefH = minSz.H
		}

		totalMinH += info.minH
		totalPrefH += info.prefH

		if c.Opts.GrowY && c.Opts.StretchY > 0 {
			totalStretch += c.Opts.StretchY
		}
	}

	availableH := r.H - totalGap
	if availableH < 0 {
		availableH = 0
	}

	for i := range infos {
		infos[i].allocated = infos[i].minH
	}

	remaining := availableH - totalMinH
	if remaining > 0 && totalStretch > 0 {
		for i := range infos {
			if infos[i].opts.GrowY && infos[i].opts.StretchY > 0 {
				extra := (remaining * infos[i].opts.StretchY) / totalStretch
				infos[i].allocated += extra
			}
		}
	} else if remaining > 0 && totalStretch == 0 {
		infos[n-1].allocated += remaining
	}

	y := r.Y

	for i := range infos {
		childH := infos[i].allocated
		if childH < 0 {
			childH = 0
		}

		if y+childH > r.Y+r.H {
			childH = r.Y + r.H - y
		}

		if childH < 0 {
			childH = 0
		}

		childW := r.W

		alignX := infos[i].opts.AlignX
		if alignX == AlignStretch || childW <= maxW {
			alignX = AlignStretch
		}

		childX := r.X
		if alignX == AlignCenter && childW > maxW {
			childX = r.X + (childW-maxW)/2
			childW = maxW
		} else if alignX == AlignEnd && childW > maxW {
			childX = r.X + childW - maxW
			childW = maxW
		}

		childRect := tui.Rect{
			X: childX,
			Y: y,
			W: childW,
			H: childH,
		}
		infos[i].view.Layout(childRect)

		y += childH + s.gap
	}
}

func (s *VStack) MinSize() tui.Size {
	maxW := 0
	totalH := 0

	n := len(s.children)

	for _, c := range s.children {
		sz := c.View.MinSize()
		if sz.W > maxW {
			maxW = sz.W
		}

		totalH += sz.H
	}

	if n > 1 {
		totalH += s.gap * (n - 1)
	}

	return tui.Size{W: maxW, H: totalH}
}

func (s *VStack) Paint(p *tui.Painter, ctx *tui.Ctx) {
	p.WithClip(s.rect, func(p *tui.Painter) {
		for _, c := range s.children {
			p.WithClip(c.View.Rect(), func(p *tui.Painter) {
				tui.PaintView(c.View, p, ctx)
			})
		}
	})
}

func (s *VStack) PaintDrawer(d tui.Drawer, ctx *tui.Ctx) {
	d.WithClip(s.rect, func(d tui.Drawer) {
		for _, c := range s.children {
			d.WithClip(c.View.Rect(), func(d tui.Drawer) {
				tui.PaintViewDrawer(c.View, d, ctx)
			})
		}
	})
}

// HandleAction handles semantic actions for focus navigation.
func (s *VStack) HandleAction(act int, ctx *tui.Ctx) bool {
	switch ui.Action(act) {
	case ui.ActionFocusNext:
		focusables := s.collectFocusable()

		next, atBoundary := s.findNextFocusable(focusables, ctx.FocusedID)
		if atBoundary {
			return false
		}

		if next != nil {
			ctx.RequestFocus(next.ID())
			ctx.Invalidate(s.Rect())

			return true
		}

		return false

	case ui.ActionFocusPrev:
		focusables := s.collectFocusable()

		prev, atBoundary := s.findPrevFocusable(focusables, ctx.FocusedID)
		if atBoundary {
			return false
		}

		if prev != nil {
			ctx.RequestFocus(prev.ID())
			ctx.Invalidate(s.Rect())

			return true
		}

		return false
	}

	return false
}

func (s *VStack) Handle(e tui.Event, ctx *tui.Ctx) bool {
	ke, ok := e.(event.KeyEvent)
	if !ok {
		return false
	}

	focusedDescendant := s.findFocusedDescendant(ctx.FocusedID)
	if focusedDescendant != nil && focusedDescendant.Handle(ke, ctx) {
		return true
	}

	focusables := s.collectFocusable()

	switch ke.Key {
	case event.KeyTab:
		if focusedDescendant == nil {
			if len(focusables) > 0 {
				ctx.RequestFocus(focusables[0].ID())
				ctx.Invalidate(s.Rect())

				return true
			}

			return false
		}

		next, atBoundary := s.findNextFocusable(focusables, ctx.FocusedID)
		if atBoundary {
			return false
		}

		if next != nil {
			ctx.RequestFocus(next.ID())
			ctx.Invalidate(s.Rect())

			return true
		}

		return false

	case event.KeyShiftTab:
		if focusedDescendant == nil {
			if len(focusables) > 0 {
				ctx.RequestFocus(focusables[len(focusables)-1].ID())
				ctx.Invalidate(s.Rect())

				return true
			}

			return false
		}

		prev, atBoundary := s.findPrevFocusable(focusables, ctx.FocusedID)
		if atBoundary {
			return false
		}

		if prev != nil {
			ctx.RequestFocus(prev.ID())
			ctx.Invalidate(s.Rect())

			return true
		}

		return false
	}

	return false
}

func (s *VStack) Focusable() bool {
	return false
}

func (s *VStack) Children() []tui.View {
	views := make([]tui.View, len(s.children))
	for i, c := range s.children {
		views[i] = c.View
	}

	return views
}

func (s *VStack) findFocusedDescendant(id tui.ID) tui.View {
	return FindByID(s, id)
}

func (s *VStack) findNextFocusable(focusables []tui.View, currentFocusID tui.ID) (tui.View, bool) {
	if len(focusables) == 0 {
		return nil, false
	}

	currentIdx := -1

	for i, v := range focusables {
		if v.ID() == currentFocusID {
			currentIdx = i

			break
		}
	}

	if currentIdx < 0 {
		return focusables[0], false
	}

	if currentIdx >= len(focusables)-1 {
		return nil, true
	}

	return focusables[currentIdx+1], false
}

func (s *VStack) collectFocusable() []tui.View {
	return CollectFocusable(s)
}

func (s *VStack) findPrevFocusable(focusables []tui.View, currentFocusID tui.ID) (tui.View, bool) {
	if len(focusables) == 0 {
		return nil, false
	}

	currentIdx := -1

	for i, v := range focusables {
		if v.ID() == currentFocusID {
			currentIdx = i

			break
		}
	}

	if currentIdx < 0 {
		return focusables[len(focusables)-1], false
	}

	if currentIdx <= 0 {
		return nil, true
	}

	return focusables[currentIdx-1], false
}
