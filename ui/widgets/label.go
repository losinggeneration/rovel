package widgets

import (
	"strings"

	"github.com/losinggeneration/rovel"
	"github.com/losinggeneration/rovel/geom"
	"github.com/losinggeneration/rovel/style"
	"github.com/losinggeneration/rovel/text"
)

// Label is a static text display widget.
type Label struct {
	id   rovel.ID
	text string
	rect rovel.Rect
	st   style.Style
}

func NewLabel(text string) *Label {
	return &Label{
		id:   rovel.NewID(),
		text: text,
	}
}

func (l *Label) SetText(ctx *rovel.Ctx, text string) {
	if l.text == text {
		return
	}

	l.text = text
	if ctx != nil {
		ctx.Invalidate(l.rect)
	}
}

func (l *Label) SetStyle(ctx *rovel.Ctx, st style.Style) {
	if l.st == st {
		return
	}

	l.st = st
	if ctx != nil {
		ctx.Invalidate(l.rect)
	}
}

func (l *Label) ID() rovel.ID {
	return l.id
}

func (l *Label) Rect() rovel.Rect {
	return l.rect
}

func (l *Label) Layout(r rovel.Rect) {
	l.rect = r
}

func (l *Label) MinSize() geom.Size {
	maxW := 0

	lines := strings.Split(l.text, "\n")
	for _, line := range lines {
		w := text.Width(line)
		if w > maxW {
			maxW = w
		}
	}

	return geom.Size{W: maxW, H: len(lines)}
}

func (l *Label) PreferredSize() geom.Size {
	return l.MinSize()
}

func (l *Label) Paint(d rovel.Drawer, ctx *rovel.Ctx) {
	x := l.rect.X
	y := l.rect.Y

	st := l.st
	if st == (style.Style{}) {
		st = ctx.Theme.Base
	}

	for _, r := range l.text {
		if r == '\n' {
			x = l.rect.X
			y++

			continue
		}

		d.DrawText(rovel.Point{X: x, Y: y}, string(r), st)
		x += rovel.RuneWidth(r)
	}
}

func (l *Label) Handle(e rovel.Event, ctx *rovel.Ctx) bool {
	return false
}

func (l *Label) Focusable() bool {
	return false
}
