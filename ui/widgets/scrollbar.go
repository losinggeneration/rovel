package widgets

import (
	"github.com/losinggeneration/rovel"
	"github.com/losinggeneration/rovel/geom"
	"github.com/losinggeneration/rovel/style"
)

// ScrollbarOpts holds options for creating a Scrollbar.
type ScrollbarOpts struct {
	ID          rovel.ID
	ContentSize int
	ViewSize    int
	Position    int
	OnScroll    func(pos int, ctx *rovel.Ctx)

	// Optional style overrides. When non-nil, the style replaces the
	// palette-derived style for that state completely (no merging).
	StyleThumb *style.Style
	StyleTrack *style.Style
}

// Scrollbar is a standalone vertical scroll indicator. It can be used
// independently or composed into containers like ScrollView.
type Scrollbar struct {
	id           rovel.ID
	rect         geom.Rect
	contentSize  int
	viewSize     int
	position     int
	onScroll     func(pos int, ctx *rovel.Ctx)
	dragging     bool
	dragStartY   int
	dragStartPos int

	stThumb *style.Style
	stTrack *style.Style
}

func NewScrollbar(opts ScrollbarOpts) *Scrollbar {
	id := opts.ID
	if id == 0 {
		id = rovel.NewID()
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

func (s *Scrollbar) ID() rovel.ID    { return s.id }
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

func (s *Scrollbar) Paint(d rovel.Drawer, ctx *rovel.Ctx) {
	r := s.rect
	if r.W <= 0 || r.H <= 0 || s.contentSize <= s.viewSize {
		return
	}

	cd, ok := rovel.CellDrawerOf(d)
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

func (s *Scrollbar) Handle(e rovel.Event, ctx *rovel.Ctx) bool {
	me, ok := e.(rovel.MouseEvent)
	if !ok {
		return false
	}

	if s.contentSize <= s.viewSize {
		return false
	}

	switch me.Action {
	case rovel.MousePress:
		if me.Button != rovel.MouseButtonLeft {
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

	case rovel.MouseDrag:
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

	case rovel.MouseRelease:
		if s.dragging {
			s.dragging = false

			return true
		}

		return false
	case rovel.MouseMove:
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
