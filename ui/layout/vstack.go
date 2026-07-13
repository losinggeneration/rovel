package layout

import (
	"github.com/losinggeneration/rovel"
	"github.com/losinggeneration/rovel/event"
	"github.com/losinggeneration/rovel/ui"
)

type VStack struct {
	id       rovel.ID
	children []Child
	rect     rovel.Rect
	gap      int
}

func NewVStack() *VStack {
	return &VStack{
		id: rovel.NewID(),
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

func (s *VStack) Add(v rovel.View) {
	s.children = append(s.children, NewChild(v))
}

func (s *VStack) AddChild(c Child) {
	s.children = append(s.children, c)
}

func (s *VStack) SetGap(gap int) {
	s.gap = gap
}

func (s *VStack) ID() rovel.ID {
	return s.id
}

func (s *VStack) Rect() rovel.Rect {
	return s.rect
}

func (s *VStack) Layout(r rovel.Rect) {
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
		view      rovel.View
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

		childRect := rovel.Rect{
			X: childX,
			Y: y,
			W: childW,
			H: childH,
		}
		infos[i].view.Layout(childRect)

		y += childH + s.gap
	}
}

func (s *VStack) MinSize() rovel.Size {
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

	return rovel.Size{W: maxW, H: totalH}
}

func (s *VStack) Paint(d rovel.Drawer, ctx *rovel.Ctx) {
	d.WithClip(s.rect, func(d rovel.Drawer) {
		for _, c := range s.children {
			d.WithClip(c.View.Rect(), func(d rovel.Drawer) {
				c.View.Paint(d, ctx)
			})
		}
	})
}

// HandleAction handles semantic actions for focus navigation.
func (s *VStack) HandleAction(act ui.Action, ctx *rovel.Ctx) bool {
	switch act {
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

func (s *VStack) Handle(e rovel.Event, ctx *rovel.Ctx) bool {
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

func (s *VStack) Focusable() bool {
	return false
}

func (s *VStack) Children() []rovel.View {
	views := make([]rovel.View, len(s.children))
	for i, c := range s.children {
		views[i] = c.View
	}

	return views
}

func (s *VStack) findFocusedDescendant(id rovel.ID) rovel.View {
	return FindByID(s, id)
}

func (s *VStack) findNextFocusable(focusables []rovel.View, currentFocusID rovel.ID) (rovel.View, bool) {
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

func (s *VStack) collectFocusable() []rovel.View {
	return CollectFocusable(s)
}

func (s *VStack) findPrevFocusable(focusables []rovel.View, currentFocusID rovel.ID) (rovel.View, bool) {
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
