package widgets

import (
	"github.com/losinggeneration/tui"
	"github.com/losinggeneration/tui/geom"
	"github.com/losinggeneration/tui/style"
)

// ScrollbarOpts holds options for creating a Scrollbar.
type ScrollbarOpts struct {
	ID          tui.ID
	ContentSize int
	ViewSize    int
	Position    int
	OnScroll    func(pos int, ctx *tui.Ctx)

	// Optional style overrides. When non-nil, the style replaces the
	// palette-derived style for that state completely (no merging).
	StyleThumb *style.Style
	StyleTrack *style.Style
}

// Scrollbar is a standalone vertical scroll indicator. It can be used
// independently or composed into containers like ScrollView.
type Scrollbar struct {
	id           tui.ID
	rect         geom.Rect
	contentSize  int
	viewSize     int
	position     int
	onScroll     func(pos int, ctx *tui.Ctx)
	dragging     bool
	dragStartY   int
	dragStartPos int

	stThumb *style.Style
	stTrack *style.Style
}

func NewScrollbar(opts ScrollbarOpts) *Scrollbar {
	id := opts.ID
	if id == 0 {
		id = tui.NewID()
	}

	return &Scrollbar{
		id:          id,
		contentSize: opts.ContentSize,
		viewSize:    opts.ViewSize,
		position:    opts.Position,
		onScroll:    opts.OnScroll,
		stThumb:     opts.StyleThumb,
		stTrack:     opts.StyleTrack,
	}
}

func (s *Scrollbar) ID() tui.ID      { return s.id }
func (s *Scrollbar) Rect() geom.Rect { return s.rect }
func (s *Scrollbar) Focusable() bool { return false }

func (s *Scrollbar) Layout(r geom.Rect) {
	s.rect = r
}

func (s *Scrollbar) MinSize() geom.Size {
	return geom.Size{W: 1, H: 1}
}

// SetState updates the scrollbar's content size, view size, and position.
func (s *Scrollbar) SetState(contentSize, viewSize, position int) {
	s.contentSize = contentSize
	s.viewSize = viewSize
	s.position = position
}

func (s *Scrollbar) Position() int {
	return s.position
}

// Dragging reports whether the scrollbar thumb is being dragged.
func (s *Scrollbar) Dragging() bool {
	return s.dragging
}

func (s *Scrollbar) Paint(p *tui.Painter, ctx *tui.Ctx) {
	s.PaintDrawer(tui.NewDrawer(p), ctx)
}

func (s *Scrollbar) PaintDrawer(d tui.Drawer, ctx *tui.Ctx) {
	r := s.rect
	if r.W <= 0 || r.H <= 0 || s.contentSize <= s.viewSize {
		return
	}

	cd, ok := tui.CellDrawerOf(d)
	if !ok {
		return
	}

	trackSt := resolveStyle(s.stTrack, ctx.Theme.Palette.BorderMuted)
	thumbSt := resolveStyle(s.stThumb, ctx.Theme.Palette.Border)

	thumbH, thumbY := s.thumbGeometry()

	for y := range r.H {
		if y >= thumbY && y < thumbY+thumbH {
			cd.SetCell(r.X, r.Y+y, '█', thumbSt)
		} else {
			cd.SetCell(r.X, r.Y+y, '░', trackSt)
		}
	}
}

// MouseOpaque marks the scrollbar as opaque to hit-testing.
func (s *Scrollbar) MouseOpaque() {}

func (s *Scrollbar) Handle(e tui.Event, ctx *tui.Ctx) bool {
	me, ok := e.(tui.MouseEvent)
	if !ok {
		return false
	}

	if s.contentSize <= s.viewSize {
		return false
	}

	switch me.Action {
	case tui.MousePress:
		if me.Button != tui.MouseButtonLeft {
			return false
		}

		clickY := me.Y - s.rect.Y
		thumbH, thumbY := s.thumbGeometry()

		if clickY >= thumbY && clickY < thumbY+thumbH {
			s.dragging = true
			s.dragStartY = me.Y
			s.dragStartPos = s.position
		} else {
			maxS := s.maxScroll()
			newPos := clickY * maxS / (s.rect.H - 1)
			newPos = max(0, min(newPos, maxS))

			s.position = newPos
			if s.onScroll != nil {
				s.onScroll(newPos, ctx)
			}
		}

		return true

	case tui.MouseDrag:
		if !s.dragging {
			return false
		}

		deltaY := me.Y - s.dragStartY
		thumbH, _ := s.thumbGeometry()

		trackSpace := s.rect.H - thumbH
		if trackSpace <= 0 {
			return true
		}

		maxS := s.maxScroll()
		newPos := s.dragStartPos + deltaY*maxS/trackSpace
		newPos = max(0, min(newPos, maxS))

		s.position = newPos
		if s.onScroll != nil {
			s.onScroll(newPos, ctx)
		}

		return true

	case tui.MouseRelease:
		if s.dragging {
			s.dragging = false

			return true
		}

		return false
	}

	return false
}

func (s *Scrollbar) maxScroll() int {
	m := s.contentSize - s.viewSize
	if m < 0 {
		return 0
	}

	return m
}

func (s *Scrollbar) thumbGeometry() (thumbH, thumbY int) {
	h := s.rect.H
	if h <= 0 || s.contentSize <= s.viewSize {
		return 0, 0
	}

	thumbH = max(1, s.viewSize*h/s.contentSize)

	maxS := s.maxScroll()
	if maxS <= 0 {
		thumbY = 0
	} else {
		thumbY = s.position * (h - thumbH) / maxS
	}

	if thumbY+thumbH > h {
		thumbY = h - thumbH
	}

	if thumbY < 0 {
		thumbY = 0
	}

	return thumbH, thumbY
}
