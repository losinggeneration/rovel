package widgets

import (
	"github.com/losinggeneration/tui"
	"github.com/losinggeneration/tui/event"
	"github.com/losinggeneration/tui/geom"
	"github.com/losinggeneration/tui/style"
	"github.com/losinggeneration/tui/text"
	"github.com/losinggeneration/tui/ui"
)

// TextInputOpts holds options for creating a TextInput.
type TextInputOpts struct {
	ID tui.ID

	// Optional style overrides. When non-nil, the style replaces the
	// palette-derived style for that state completely (no merging).
	StyleNormal    *style.Style
	StyleFocused   *style.Style
	StyleSelection *style.Style
}

// TextInput is a single-line text input widget with cursor navigation.
type TextInput struct {
	id      tui.ID
	text    string
	cursor  int // UTF-8 byte offset in [0..len(text)], always at a cluster boundary
	scrollX int // horizontal scroll in cells
	rect    tui.Rect
	anchor  int // selection anchor; -1 = no selection

	stNormal    *style.Style
	stFocused   *style.Style
	stSelection *style.Style
}

func NewTextInput() *TextInput {
	return NewTextInputOpts(TextInputOpts{})
}

func NewTextInputOpts(opts TextInputOpts) *TextInput {
	id := opts.ID
	if id == 0 {
		id = tui.NewID()
	}

	return &TextInput{
		id:          id,
		anchor:      -1,
		stNormal:    opts.StyleNormal,
		stFocused:   opts.StyleFocused,
		stSelection: opts.StyleSelection,
	}
}

func (t *TextInput) SetText(ctx *tui.Ctx, s string) {
	t.text = text.Sanitize(s)
	t.cursor = len(t.text)
	t.cursor = text.ClampCluster(t.text, t.cursor)
	t.updateScroll()

	if ctx != nil {
		ctx.Invalidate(t.rect)
	}
}

func (t *TextInput) Text() string {
	return t.text
}

func (t *TextInput) ID() tui.ID {
	return t.id
}

func (t *TextInput) Rect() tui.Rect {
	return t.rect
}

func (t *TextInput) Layout(r tui.Rect) {
	t.rect = r
}

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
	t.PaintDrawer(tui.NewDrawer(p), ctx)
}

func (t *TextInput) PaintDrawer(d tui.Drawer, ctx *tui.Ctx) {
	if t.rect.W <= 0 {
		return
	}

	cd, ok := tui.CellDrawerOf(d)
	if !ok {
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
		cd.SetCell(x, y, ' ', resolveStyle(t.stNormal, ctx.Theme.Base))

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

		// Check if this is the cursor position or in selection
		hasSel := t.hasSelection()
		cursorHere := focused && !hasSel && byteOff == t.cursor
		inSelection := focused && hasSel && byteOff >= t.selectionRange().Start && byteOff < t.selectionRange().End

		var st style.Style

		switch {
		case cursorHere:
			st = resolveStyle(t.stFocused, ctx.Theme.Palette.Focus)
		case inSelection:
			st = resolveStyle(t.stSelection, ctx.Theme.Palette.Selection)
		default:
			st = resolveStyle(t.stNormal, ctx.Theme.Base)
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

			cd.SetCell(x, y, r, st)
			x += rw
			availableW -= rw
		}

		byteOff = next
	}

	// Draw cursor at end of text if positioned there (not during selection)
	if focused && !t.hasSelection() && t.cursor == len(t.text) && availableW > 0 {
		cd.SetCell(x, y, ' ', resolveStyle(t.stFocused, ctx.Theme.Palette.Focus))

		x++
		availableW--
	}

	// Fill remaining space with base style
	normalSt := resolveStyle(t.stNormal, ctx.Theme.Base)
	for availableW > 0 {
		cd.SetCell(x, y, ' ', normalSt)

		x++
		availableW--
	}
}

// Handle processes keyboard, mouse, and paste events.
func (t *TextInput) Handle(e tui.Event, ctx *tui.Ctx) bool {
	// Mouse click positions the cursor
	if me, ok := e.(tui.MouseEvent); ok {
		if me.Button == tui.MouseButtonLeft && me.Action == tui.MousePress {
			if ctx != nil && ctx.RequestFocus != nil {
				ctx.RequestFocus(t.id)
			}
			// Convert click X to cursor position
			col := me.X - t.rect.X + t.scrollX
			t.cursor = text.OffsetAtColumnBias(t.text, col, text.BiasLeft)
			t.cursor = text.ClampCluster(t.text, t.cursor)
			t.updateScroll()
			ctx.Invalidate(t.rect)

			return true
		}

		return false
	}

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
		// Insert character at cursor (replacing selection if any).
		insert := text.Sanitize(string(ke.Rune))
		if insert == "" {
			return true
		}

		if t.hasSelection() {
			t.deleteSelection()
		}

		t.cursor = text.ClampCluster(t.text, t.cursor)
		t.text = t.text[:t.cursor] + insert + t.text[t.cursor:]
		t.cursor += len(insert)
		t.cursor = text.ClampCluster(t.text, t.cursor)
		t.anchor = -1
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

// HandleAction handles semantic actions.
func (t *TextInput) HandleAction(act int, ctx *tui.Ctx) bool {
	if ctx.FocusedID != t.id {
		return false
	}

	// Handle shift+move for selection extension.
	isShift := ctx.Mod&tui.ModShift != 0

	switch ui.Action(act) {
	case ui.ActionMoveLeft:
		if isShift {
			if t.anchor < 0 {
				t.anchor = t.cursor
			}
		} else {
			t.anchor = -1
		}

		t.cursor = text.PrevCluster(t.text, t.cursor)
		t.updateScroll()
		ctx.Invalidate(t.rect)

		return true
	case ui.ActionMoveRight:
		if isShift {
			if t.anchor < 0 {
				t.anchor = t.cursor
			}
		} else {
			t.anchor = -1
		}

		t.cursor = text.NextCluster(t.text, t.cursor)
		t.updateScroll()
		ctx.Invalidate(t.rect)

		return true
	case ui.ActionDeleteBackward:
		if t.hasSelection() {
			t.deleteSelection()
		} else {
			t.text, t.cursor = text.DeletePrevCluster(t.text, t.cursor)
		}

		t.anchor = -1
		t.updateScroll()
		ctx.Invalidate(t.rect)

		return true
	case ui.ActionDeleteForward:
		if t.hasSelection() {
			t.deleteSelection()
		} else {
			t.text, t.cursor = text.DeleteNextCluster(t.text, t.cursor)
		}

		t.anchor = -1
		t.updateScroll()
		ctx.Invalidate(t.rect)

		return true
	case ui.ActionHome:
		if isShift {
			if t.anchor < 0 {
				t.anchor = t.cursor
			}
		} else {
			t.anchor = -1
		}

		t.cursor = 0
		t.updateScroll()
		ctx.Invalidate(t.rect)

		return true
	case ui.ActionEnd:
		if isShift {
			if t.anchor < 0 {
				t.anchor = t.cursor
			}
		} else {
			t.anchor = -1
		}

		t.cursor = len(t.text)
		t.updateScroll()
		ctx.Invalidate(t.rect)

		return true
	case ui.ActionSelectAll:
		t.anchor = 0
		t.cursor = len(t.text)
		t.updateScroll()
		ctx.Invalidate(t.rect)

		return true
	case ui.ActionCopy:
		if t.hasSelection() && ctx.ClipboardWrite != nil {
			ctx.ClipboardWrite(t.selectedText())
		}

		return true
	case ui.ActionCut:
		if t.hasSelection() {
			if ctx.ClipboardWrite != nil {
				ctx.ClipboardWrite(t.selectedText())
			}

			t.deleteSelection()
			t.updateScroll()
			ctx.Invalidate(t.rect)
		}

		return true
	}

	return false
}

// IsTextInputMode returns true, indicating the keymap should use text input context.
func (t *TextInput) IsTextInputMode() bool {
	return true
}

func (t *TextInput) Focusable() bool {
	return true
}

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

func (t *TextInput) hasSelection() bool {
	return t.anchor >= 0 && t.anchor != t.cursor
}

func (t *TextInput) selectionRange() text.Range {
	if t.anchor < 0 {
		return text.Range{Start: t.cursor, End: t.cursor}
	}

	return text.Range{Start: t.anchor, End: t.cursor}.Normalized()
}

func (t *TextInput) selectedText() string {
	if !t.hasSelection() {
		return ""
	}

	r := t.selectionRange()

	return t.text[r.Start:r.End]
}

func (t *TextInput) deleteSelection() {
	if !t.hasSelection() {
		return
	}

	r := t.selectionRange()
	t.text, _ = text.DeleteRange(t.text, r)
	t.cursor = r.Start
	t.anchor = -1
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
