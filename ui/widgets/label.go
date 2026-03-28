package widgets

import (
	"strings"

	"github.com/losinggeneration/tui"
	"github.com/losinggeneration/tui/geom"
	"github.com/losinggeneration/tui/style"
	"github.com/losinggeneration/tui/text"
)

// Label is a static text display widget.
type Label struct {
	id   tui.ID
	text string
	rect tui.Rect
	st   style.Style
}

func NewLabel(text string) *Label {
	return &Label{
		id:   tui.NewID(),
		text: text,
	}
}

func (l *Label) SetText(ctx *tui.Ctx, text string) {
	if l.text == text {
		return
	}

	l.text = text
	if ctx != nil {
		ctx.Invalidate(l.rect)
	}
}

func (l *Label) SetStyle(ctx *tui.Ctx, st style.Style) {
	if l.st == st {
		return
	}

	l.st = st
	if ctx != nil {
		ctx.Invalidate(l.rect)
	}
}

func (l *Label) ID() tui.ID {
	return l.id
}

func (l *Label) Rect() tui.Rect {
	return l.rect
}

func (l *Label) Layout(r tui.Rect) {
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

func (l *Label) Paint(d tui.Drawer, ctx *tui.Ctx) {
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

		d.DrawText(tui.Point{X: x, Y: y}, string(r), st)
		x += tui.RuneWidth(r)
	}
}

func (l *Label) Handle(e tui.Event, ctx *tui.Ctx) bool {
	return false
}

func (l *Label) Focusable() bool {
	return false
}
