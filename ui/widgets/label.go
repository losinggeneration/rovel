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

// NewLabel creates a new label with the given text.
func NewLabel(text string) *Label {
	return &Label{
		id:   tui.NewID(),
		text: text,
		st:   style.Style{},
	}
}

// SetText sets the label's text and invalidates the rect.
func (l *Label) SetText(ctx *tui.Ctx, text string) {
	if l.text == text {
		return
	}

	l.text = text
	if ctx != nil {
		ctx.Invalidate(l.rect)
	}
}

// SetStyle sets the label's style and invalidates the rect.
func (l *Label) SetStyle(ctx *tui.Ctx, st style.Style) {
	if l.st == st {
		return
	}

	l.st = st
	if ctx != nil {
		ctx.Invalidate(l.rect)
	}
}

// ID returns the label's unique ID.
func (l *Label) ID() tui.ID {
	return l.id
}

// Rect returns the label's current rect.
func (l *Label) Rect() tui.Rect {
	return l.rect
}

// Layout positions the label within the given rect.
func (l *Label) Layout(r tui.Rect) {
	l.rect = r
}

// MinSize returns the minimum size needed for the label.
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

// Paint renders the label.
func (l *Label) Paint(p *tui.Painter, ctx *tui.Ctx) {
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

		p.Text(x, y, string(r), st)
		x += tui.RuneWidth(r)
	}
}

// Handle processes events - label doesn't handle any.
func (l *Label) Handle(e tui.Event, ctx *tui.Ctx) bool {
	return false
}

// Focusable returns false - labels don't receive focus.
func (l *Label) Focusable() bool {
	return false
}
