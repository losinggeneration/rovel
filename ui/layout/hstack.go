package layout

import (
	"github.com/losinggeneration/tui"
	"github.com/losinggeneration/tui/event"
	"github.com/losinggeneration/tui/ui"
)

type HStack struct {
	id       tui.ID
	children []Child
	rect     tui.Rect
	gap      int
}

func NewHStack() *HStack {
	return &HStack{
		id: tui.NewID(),
	}
}

func NewHStackWithChildren(children []Child) *HStack {
	s := NewHStack()
	s.children = children

	return s
}

func NewHStackWithGap(children []Child, gap int) *HStack {
	s := NewHStackWithChildren(children)
	s.gap = gap

	return s
}

func (s *HStack) Add(v tui.View) {
	s.children = append(s.children, NewChild(v))
}

func (s *HStack) AddChild(c Child) {
	s.children = append(s.children, c)
}

func (s *HStack) SetGap(gap int) {
	s.gap = gap
}

func (s *HStack) ID() tui.ID {
	return s.id
}

func (s *HStack) Rect() tui.Rect {
	return s.rect
}

func (s *HStack) Layout(r tui.Rect) {
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
		minW      int
		prefW     int
		allocated int
	}

	infos := make([]childInfo, n)
	totalMinW := 0
	totalPrefW := 0
	maxH := 0
	totalStretch := 0

	for i, c := range s.children {
		info := &infos[i]
		info.view = c.View
		info.opts = c.Opts

		minSz := c.View.MinSize()

		info.minW = minSz.W

		if minSz.H > maxH {
			maxH = minSz.H
		}

		if ps, ok := c.View.(ui.PreferredSizer); ok {
			info.prefW = ps.PreferredSize().W
		} else {
			info.prefW = minSz.W
		}

		totalMinW += info.minW
		totalPrefW += info.prefW

		if c.Opts.GrowX && c.Opts.StretchX > 0 {
			totalStretch += c.Opts.StretchX
		}
	}

	availableW := r.W - totalGap
	if availableW < 0 {
		availableW = 0
	}

	for i := range infos {
		infos[i].allocated = infos[i].minW
	}

	remaining := availableW - totalMinW
	if remaining > 0 && totalStretch > 0 {
		for i := range infos {
			if infos[i].opts.GrowX && infos[i].opts.StretchX > 0 {
				extra := (remaining * infos[i].opts.StretchX) / totalStretch
				infos[i].allocated += extra
			}
		}
	} else if remaining > 0 && totalStretch == 0 {
		infos[n-1].allocated += remaining
	}

	x := r.X

	for i := range infos {
		childW := infos[i].allocated
		if childW < 0 {
			childW = 0
		}

		if x+childW > r.X+r.W {
			childW = r.X + r.W - x
		}

		if childW < 0 {
			childW = 0
		}

		childH := r.H

		alignY := infos[i].opts.AlignY
		if alignY == AlignStretch || childH <= maxH {
			alignY = AlignStretch
		}

		childY := r.Y
		if alignY == AlignCenter && childH > maxH {
			childY = r.Y + (childH-maxH)/2
			childH = maxH
		} else if alignY == AlignEnd && childH > maxH {
			childY = r.Y + childH - maxH
			childH = maxH
		}

		childRect := tui.Rect{
			X: x,
			Y: childY,
			W: childW,
			H: childH,
		}
		infos[i].view.Layout(childRect)

		x += childW + s.gap
	}
}

func (s *HStack) MinSize() tui.Size {
	totalW := 0
	maxH := 0

	n := len(s.children)

	for _, c := range s.children {
		sz := c.View.MinSize()

		totalW += sz.W

		if sz.H > maxH {
			maxH = sz.H
		}
	}

	if n > 1 {
		totalW += s.gap * (n - 1)
	}

	return tui.Size{W: totalW, H: maxH}
}

func (s *HStack) Paint(p *tui.Painter, ctx *tui.Ctx) {
	p.WithClip(s.rect, func(p *tui.Painter) {
		for _, c := range s.children {
			p.WithClip(c.View.Rect(), func(p *tui.Painter) {
				tui.PaintView(c.View, p, ctx)
			})
		}
	})
}

func (s *HStack) PaintDrawer(d tui.Drawer, ctx *tui.Ctx) {
	d.WithClip(s.rect, func(d tui.Drawer) {
		for _, c := range s.children {
			d.WithClip(c.View.Rect(), func(d tui.Drawer) {
				tui.PaintViewDrawer(c.View, d, ctx)
			})
		}
	})
}

// HandleAction handles semantic actions for focus navigation.
func (s *HStack) HandleAction(act int, ctx *tui.Ctx) bool {
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
	default:
	}

	return false
}

func (s *HStack) Handle(e tui.Event, ctx *tui.Ctx) bool {
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
	default:
	}

	return false
}

func (s *HStack) Focusable() bool {
	return false
}

func (s *HStack) Children() []tui.View {
	views := make([]tui.View, len(s.children))
	for i, c := range s.children {
		views[i] = c.View
	}

	return views
}

func (s *HStack) findFocusedDescendant(id tui.ID) tui.View {
	return FindByID(s, id)
}

func (s *HStack) findNextFocusable(focusables []tui.View, currentFocusID tui.ID) (tui.View, bool) {
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

func (s *HStack) collectFocusable() []tui.View {
	return CollectFocusable(s)
}

func (s *HStack) findPrevFocusable(focusables []tui.View, currentFocusID tui.ID) (tui.View, bool) {
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
