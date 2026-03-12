package widgets

import (
	"github.com/losinggeneration/tui"
	"github.com/losinggeneration/tui/geom"
	"github.com/losinggeneration/tui/style"
	"github.com/losinggeneration/tui/text"
)

// TextInput is a single-line text input widget with cursor navigation.
type TextInput struct {
	id      tui.ID
	text    []rune
	cursor  int // position in [0..len(text)]
	scrollX int // horizontal scroll in cells
	rect    tui.Rect
}

// NewTextInput creates a new text input.
func NewTextInput() *TextInput {
	return &TextInput{
		id:      tui.NewID(),
		text:    make([]rune, 0),
		cursor:  0,
		scrollX: 0,
	}
}

// SetText sets the input text and moves cursor to end.
func (t *TextInput) SetText(ctx *tui.Ctx, s string) {
	t.text = []rune(s)
	t.cursor = len(t.text)
	t.updateScroll()
	if ctx != nil {
		ctx.Invalidate(t.rect)
	}
}

// Text returns the current text.
func (t *TextInput) Text() string {
	return string(t.text)
}

// ID returns the text input's unique ID.
func (t *TextInput) ID() tui.ID {
	return t.id
}

// Rect returns the text input's current rect.
func (t *TextInput) Rect() tui.Rect {
	return t.rect
}

// Layout positions the text input within the given rect.
func (t *TextInput) Layout(r tui.Rect) {
	t.rect = r
}

// MinSize returns the minimum size needed for the text input.
func (t *TextInput) MinSize() geom.Size {
	return geom.Size{W: 10, H: 1}
}

func (t *TextInput) PreferredSize() geom.Size {
	w := text.Width(string(t.text))
	if w < 10 {
		w = 10
	}
	return geom.Size{W: w, H: 1}
}

// Paint renders the text input.
func (t *TextInput) Paint(p *tui.Painter, ctx *tui.Ctx) {
	if t.rect.W <= 0 {
		return
	}

	focused := ctx.FocusedID == t.id
	x := t.rect.X
	y := t.rect.Y

	// Calculate visible window
	visibleW := t.rect.W
	availableW := visibleW

	// Skip runes that are scrolled out of view
	textIdx := 0
	skipped := 0
	for i := 0; i < len(t.text); i++ {
		rw := text.WidthRune(t.text[i])
		if skipped+rw > t.scrollX {
			// This rune is (partially) visible
			textIdx = i
			break
		}
		skipped += rw
	}
	if skipped > t.scrollX {
		// We're in the middle of a wide glyph - leave a blank cell
		p.SetCell(x, y, ' ', ctx.Theme.Base)
		x++
		availableW--
		textIdx++
	}

	// Draw visible runes
	for i := textIdx; i < len(t.text) && availableW > 0; i++ {
		r := t.text[i]
		rw := text.WidthRune(r)

		// Check if this is the cursor position
		cursorHere := focused && i == t.cursor

		var st style.Style
		if cursorHere {
			st = ctx.Theme.Focus
		} else {
			st = ctx.Theme.Base
		}

		// Don't paint continuation cells
		if rw > 0 {
			p.SetCell(x, y, r, st)
			x += rw
			availableW -= rw
		}

		// Skip continuation cell for wide chars
		if rw == 2 && availableW > 0 {
			availableW--
		}
	}

	// Draw cursor at end of text if positioned there
	if focused && t.cursor == len(t.text) && availableW > 0 {
		p.SetCell(x, y, ' ', ctx.Theme.Focus)
		x++
		availableW--
	}

	// Fill remaining space with base style
	for availableW > 0 {
		p.SetCell(x, y, ' ', ctx.Theme.Base)
		x++
		availableW--
	}
}

// Handle processes keyboard events.
func (t *TextInput) Handle(e tui.Event, ctx *tui.Ctx) bool {
	ke, ok := e.(tui.KeyEvent)
	if !ok {
		return false
	}

	if ctx.FocusedID != t.id {
		return false
	}

	switch ke.Key {
	case tui.KeyRune:
		// Insert character at cursor
		t.text = append(t.text[:t.cursor], append([]rune{ke.Rune}, t.text[t.cursor:]...)...)
		t.cursor++
		t.updateScroll()
		ctx.Invalidate(t.rect)
		return true

	case tui.KeyLeft:
		if t.cursor > 0 {
			t.cursor--
			t.updateScroll()
			ctx.Invalidate(t.rect)
		}
		return true

	case tui.KeyRight:
		if t.cursor < len(t.text) {
			t.cursor++
			t.updateScroll()
			ctx.Invalidate(t.rect)
		}
		return true

	case tui.KeyBackspace:
		if t.cursor > 0 {
			t.text = append(t.text[:t.cursor-1], t.text[t.cursor:]...)
			t.cursor--
			t.updateScroll()
			ctx.Invalidate(t.rect)
		}
		return true
	}

	return false
}

// Focusable returns true - text inputs can receive focus.
func (t *TextInput) Focusable() bool {
	return true
}

// updateScroll adjusts scrollX to keep cursor visible.
func (t *TextInput) updateScroll() {
	if t.rect.W <= 0 {
		t.scrollX = 0
		return
	}

	// Calculate cursor position in cells
	cursorX := 0
	for i := 0; i < t.cursor; i++ {
		cursorX += text.WidthRune(t.text[i])
	}

	// Keep cursor within visible bounds
	if cursorX < t.scrollX {
		t.scrollX = cursorX
	}
	if cursorX >= t.scrollX+t.rect.W {
		t.scrollX = cursorX - t.rect.W + 1
	}

	// Ensure scrollX is non-negative
	if t.scrollX < 0 {
		t.scrollX = 0
	}
}
