package widgets

import (
	"github.com/losinggeneration/tui"
	"github.com/losinggeneration/tui/event"
	"github.com/losinggeneration/tui/geom"
	"github.com/losinggeneration/tui/style"
	"github.com/losinggeneration/tui/text"
	"github.com/losinggeneration/tui/ui"
)

// TextInput is a single-line text input widget with cursor navigation.
type TextInput struct {
	id      tui.ID
	text    string
	cursor  int // UTF-8 byte offset in [0..len(text)], always at a cluster boundary
	scrollX int // horizontal scroll in cells
	rect    tui.Rect
}

// NewTextInput creates a new text input.
func NewTextInput() *TextInput {
	return &TextInput{
		id:      tui.NewID(),
		text:    "",
		cursor:  0,
		scrollX: 0,
	}
}

// SetText sets the input text and moves cursor to end.
func (t *TextInput) SetText(ctx *tui.Ctx, s string) {
	t.text = text.Sanitize(s)
	t.cursor = len(t.text)
	t.cursor = text.ClampCluster(t.text, t.cursor)
	t.updateScroll()
	if ctx != nil {
		ctx.Invalidate(t.rect)
	}
}

// Text returns the current text.
func (t *TextInput) Text() string {
	return t.text
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
	w := text.Width(t.text)
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

	startLeft := text.OffsetAtColumnBias(t.text, t.scrollX, text.BiasLeft)
	startRight := text.OffsetAtColumnBias(t.text, t.scrollX, text.BiasRight)
	startByte := startLeft
	if startRight != startLeft {
		// We're in the middle of a wide cluster; leave a blank cell.
		p.SetCell(x, y, ' ', ctx.Theme.Base)
		x++
		availableW--
		startByte = startRight
	}

	// Draw visible runes
	byteOff := startByte
	for byteOff < len(t.text) && availableW > 0 {
		next := text.NextCluster(t.text, byteOff)
		if next <= byteOff {
			break
		}

		// Check if this is the cursor position
		cursorHere := focused && byteOff == t.cursor

		var st style.Style
		if cursorHere {
			st = ctx.Theme.Palette.Focus
		} else {
			st = ctx.Theme.Base
		}

		// Paint the cluster rune-by-rune. For some clusters (e.g. flags), this
		// allows terminals to render ligatures across cells.
		cluster := t.text[byteOff:next]
		for _, r := range cluster {
			rw := tui.RuneWidth(r)
			if rw <= 0 {
				continue
			}
			if availableW < rw {
				availableW = 0
				break
			}
			p.SetCell(x, y, r, st)
			x += rw
			availableW -= rw
		}

		byteOff = next
	}

	// Draw cursor at end of text if positioned there
	if focused && t.cursor == len(t.text) && availableW > 0 {
		p.SetCell(x, y, ' ', ctx.Theme.Palette.Focus)
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

// Handle processes keyboard and paste events.
func (t *TextInput) Handle(e tui.Event, ctx *tui.Ctx) bool {
	// Handle paste events
	if pe, ok := e.(event.PasteEvent); ok {
		return t.handlePaste(pe, ctx)
	}

	ke, ok := e.(tui.KeyEvent)
	if !ok {
		return false
	}

	if ctx.FocusedID != t.id {
		return false
	}

	switch ke.Key {
	case tui.KeyRune:
		// Insert character at cursor.
		insert := text.Sanitize(string(ke.Rune))
		if insert == "" {
			return true
		}
		t.cursor = text.ClampCluster(t.text, t.cursor)
		t.text = t.text[:t.cursor] + insert + t.text[t.cursor:]
		t.cursor += len(insert)
		t.cursor = text.ClampCluster(t.text, t.cursor)
		t.updateScroll()
		ctx.Invalidate(t.rect)
		return true

	case tui.KeyLeft:
		t.cursor = text.PrevCluster(t.text, t.cursor)
		t.updateScroll()
		ctx.Invalidate(t.rect)
		return true

	case tui.KeyRight:
		t.cursor = text.NextCluster(t.text, t.cursor)
		t.updateScroll()
		ctx.Invalidate(t.rect)
		return true

	case tui.KeyBackspace:
		t.text, t.cursor = text.DeletePrevCluster(t.text, t.cursor)
		t.updateScroll()
		ctx.Invalidate(t.rect)
		return true

	case tui.KeyDelete:
		t.text, t.cursor = text.DeleteNextCluster(t.text, t.cursor)
		t.updateScroll()
		ctx.Invalidate(t.rect)
		return true
	}

	return false
}

// Handle also processes paste events.
func (t *TextInput) handlePaste(e event.PasteEvent, ctx *tui.Ctx) bool {
	if ctx.FocusedID != t.id {
		return false
	}
	insert := text.Sanitize(e.Text)
	if insert == "" {
		return true
	}
	t.cursor = text.ClampCluster(t.text, t.cursor)
	t.text = t.text[:t.cursor] + insert + t.text[t.cursor:]
	t.cursor += len(insert)
	t.cursor = text.ClampCluster(t.text, t.cursor)
	t.updateScroll()
	ctx.Invalidate(t.rect)
	return true
}

// HandleAction handles semantic actions.
func (t *TextInput) HandleAction(act int, ctx *tui.Ctx) bool {
	if ctx.FocusedID != t.id {
		return false
	}
	switch ui.Action(act) {
	case ui.ActionMoveLeft:
		t.cursor = text.PrevCluster(t.text, t.cursor)
		t.updateScroll()
		ctx.Invalidate(t.rect)
		return true
	case ui.ActionMoveRight:
		t.cursor = text.NextCluster(t.text, t.cursor)
		t.updateScroll()
		ctx.Invalidate(t.rect)
		return true
	case ui.ActionDeleteBackward:
		t.text, t.cursor = text.DeletePrevCluster(t.text, t.cursor)
		t.updateScroll()
		ctx.Invalidate(t.rect)
		return true
	case ui.ActionDeleteForward:
		t.text, t.cursor = text.DeleteNextCluster(t.text, t.cursor)
		t.updateScroll()
		ctx.Invalidate(t.rect)
		return true
	case ui.ActionHome:
		t.cursor = 0
		t.updateScroll()
		ctx.Invalidate(t.rect)
		return true
	case ui.ActionEnd:
		t.cursor = len(t.text)
		t.updateScroll()
		ctx.Invalidate(t.rect)
		return true
	}
	return false
}

// IsTextInputMode returns true, indicating the keymap should use text input context.
func (t *TextInput) IsTextInputMode() bool {
	return true
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
	t.cursor = text.ClampCluster(t.text, t.cursor)
	cursorX := text.ColumnOf(t.text, t.cursor)

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
