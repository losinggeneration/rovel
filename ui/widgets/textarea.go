package widgets

import (
	"strings"

	"github.com/losinggeneration/tui"
	"github.com/losinggeneration/tui/event"
	"github.com/losinggeneration/tui/geom"
	"github.com/losinggeneration/tui/style"
	"github.com/losinggeneration/tui/text"
	"github.com/losinggeneration/tui/ui"
)

// TextAreaOpts holds options for creating a TextArea.
type TextAreaOpts struct {
	ID tui.ID

	// Optional style overrides. When non-nil, the style replaces the
	// palette-derived style for that state completely (no merging).
	StyleNormal    *style.Style
	StyleFocused   *style.Style
	StyleSelection *style.Style
}

// TextArea is a multi-line text editing widget with cursor navigation.
type TextArea struct {
	id   tui.ID
	rect tui.Rect
	text string

	cursor     int // UTF-8 byte offset, always at a cluster boundary
	desiredCol int // sticky column for Up/Down; -1 = unset

	lines   []lineEntry
	scrollY int
	scrollX int

	wordWrap bool
	readOnly bool

	anchor int // -1 = no selection (used in Gap 4)

	stNormal    *style.Style
	stFocused   *style.Style
	stSelection *style.Style
}

// lineEntry tracks byte offsets for a single logical line.
type lineEntry struct {
	startByte int
	endByte   int // exclusive, before \n
}

func NewTextArea() *TextArea {
	return NewTextAreaOpts(TextAreaOpts{})
}

func NewTextAreaOpts(opts TextAreaOpts) *TextArea {
	id := opts.ID
	if id == 0 {
		id = tui.NewID()
	}

	ta := &TextArea{
		id:          id,
		desiredCol:  -1,
		anchor:      -1,
		stNormal:    opts.StyleNormal,
		stFocused:   opts.StyleFocused,
		stSelection: opts.StyleSelection,
	}
	ta.rebuildLineIndex()

	return ta
}

func (ta *TextArea) SetText(ctx *tui.Ctx, s string) {
	ta.text = text.Sanitize(s)
	ta.rebuildLineIndex()
	ta.cursor = len(ta.text)
	ta.cursor = text.ClampCluster(ta.text, ta.cursor)
	ta.desiredCol = -1
	ta.anchor = -1
	ta.scrollToCursor()

	if ctx != nil {
		ctx.Invalidate(ta.rect)
	}
}

func (ta *TextArea) Text() string {
	return ta.text
}

func (ta *TextArea) SetWordWrap(enabled bool) {
	ta.wordWrap = enabled
}

func (ta *TextArea) SetReadOnly(ro bool) {
	ta.readOnly = ro
}

func (ta *TextArea) ID() tui.ID         { return ta.id }
func (ta *TextArea) Rect() tui.Rect     { return ta.rect }
func (ta *TextArea) Layout(r tui.Rect)  { ta.rect = r }
func (ta *TextArea) MinSize() geom.Size { return geom.Size{W: 10, H: 3} }
func (ta *TextArea) Focusable() bool    { return true }

// IsTextInputMode returns true when the text area is not read-only.
func (ta *TextArea) IsTextInputMode() bool { return !ta.readOnly }

// rebuildLineIndex scans text for \n and builds the line index.
// An empty text produces one empty entry.
func (ta *TextArea) rebuildLineIndex() {
	ta.lines = ta.lines[:0]
	start := 0

	for i := range len(ta.text) {
		if ta.text[i] == '\n' {
			ta.lines = append(ta.lines, lineEntry{startByte: start, endByte: i})
			start = i + 1
		}
	}
	// Final line (or only line if no \n)
	ta.lines = append(ta.lines, lineEntry{startByte: start, endByte: len(ta.text)})
}

// cursorLine returns the line index containing the cursor.
func (ta *TextArea) cursorLine() int {
	for i, ln := range ta.lines {
		// Cursor is within or at the end of this line
		if ta.cursor >= ln.startByte && ta.cursor <= ln.endByte {
			return i
		}
	}

	return len(ta.lines) - 1
}

// cursorCol returns the display column of the cursor within its line.
func (ta *TextArea) cursorCol() int {
	line := ta.cursorLine()
	ln := ta.lines[line]

	return text.ColumnOf(ta.text[ln.startByte:ln.endByte], ta.cursor-ln.startByte)
}

// ScrollY returns the current vertical scroll offset (line index).
func (ta *TextArea) ScrollY() int { return ta.scrollY }

// LineCount returns the number of lines in the text.
func (ta *TextArea) LineCount() int { return len(ta.lines) }

func (ta *TextArea) SetScrollY(ctx *tui.Ctx, y int) {
	old := ta.scrollY
	ta.scrollY = y
	ta.clampScrollY()

	if ta.scrollY != old && ctx != nil {
		ctx.Invalidate(ta.rect)
	}
}

// scrollBy adjusts scrollY by delta, clamping to valid range.
func (ta *TextArea) scrollBy(delta int) {
	ta.scrollY += delta
	ta.clampScrollY()
}

// clampScrollY keeps scrollY within valid bounds.
func (ta *TextArea) clampScrollY() {
	maxScroll := len(ta.lines) - ta.rect.H
	if maxScroll < 0 {
		maxScroll = 0
	}

	if ta.scrollY < 0 {
		ta.scrollY = 0
	} else if ta.scrollY > maxScroll {
		ta.scrollY = maxScroll
	}
}

// scrollToCursor ensures the cursor line is visible.
func (ta *TextArea) scrollToCursor() {
	if ta.rect.H <= 0 {
		return
	}

	line := ta.cursorLine()
	if line < ta.scrollY {
		ta.scrollY = line
	}

	if line >= ta.scrollY+ta.rect.H {
		ta.scrollY = line - ta.rect.H + 1
	}
}

// Paint renders the text area.
func (ta *TextArea) Paint(p *tui.Painter, ctx *tui.Ctx) {
	ta.PaintDrawer(tui.NewDrawer(p), ctx)
}

func (ta *TextArea) PaintDrawer(d tui.Drawer, ctx *tui.Ctx) {
	if ta.rect.W <= 0 || ta.rect.H <= 0 {
		return
	}

	cd, ok := tui.CellDrawerOf(d)
	if !ok {
		return
	}

	focused := ctx.FocusedID == ta.id
	curLine := ta.cursorLine()

	for row := range ta.rect.H {
		lineIdx := ta.scrollY + row
		y := ta.rect.Y + row
		x := ta.rect.X
		availW := ta.rect.W

		if lineIdx >= len(ta.lines) {
			// Fill empty rows
			normalSt := resolveStyle(ta.stNormal, ctx.Theme.Base)
			for availW > 0 {
				cd.SetCell(x, y, ' ', normalSt)

				x++
				availW--
			}

			continue
		}

		ln := ta.lines[lineIdx]
		lineText := ta.text[ln.startByte:ln.endByte]

		// Precompute selection range for this line.
		var selStart, selEnd int

		hasSel := focused && ta.hasSelection()
		if hasSel {
			sr := ta.selectionRange()
			selStart = sr.Start
			selEnd = sr.End
		}

		// Render line content
		byteOff := 0
		for byteOff < len(lineText) && availW > 0 {
			next := text.NextCluster(lineText, byteOff)
			if next <= byteOff {
				break
			}

			absByte := ln.startByte + byteOff
			cursorHere := focused && !hasSel && lineIdx == curLine && absByte == ta.cursor
			inSel := hasSel && absByte >= selStart && absByte < selEnd

			var st style.Style
			if cursorHere {
				st = resolveStyle(ta.stFocused, ctx.Theme.Palette.Focus)
			} else if inSel {
				st = resolveStyle(ta.stSelection, ctx.Theme.Palette.Selection)
			} else {
				st = resolveStyle(ta.stNormal, ctx.Theme.Base)
			}

			cluster := lineText[byteOff:next]
			for _, r := range cluster {
				rw := tui.RuneWidth(r)
				if rw <= 0 {
					continue
				}

				if availW < rw {
					availW = 0

					break
				}

				cd.SetCell(x, y, r, st)
				x += rw
				availW -= rw
			}

			byteOff = next
		}

		// Draw cursor at end of line if positioned there (not during selection)
		if focused && !hasSel && lineIdx == curLine && ta.cursor == ln.endByte && availW > 0 {
			cd.SetCell(x, y, ' ', resolveStyle(ta.stFocused, ctx.Theme.Palette.Focus))

			x++
			availW--
		}

		// Fill remaining space
		normalSt := resolveStyle(ta.stNormal, ctx.Theme.Base)
		for availW > 0 {
			cd.SetCell(x, y, ' ', normalSt)

			x++
			availW--
		}
	}
}

// Handle processes keyboard, mouse, and paste events.
func (ta *TextArea) Handle(e tui.Event, ctx *tui.Ctx) bool {
	// Mouse handling: press, drag, double-click, wheel
	if me, ok := e.(tui.MouseEvent); ok {
		switch {
		case me.Button == tui.MouseButtonWheelUp:
			ta.scrollBy(-3)
			ctx.Invalidate(ta.rect)

			return true
		case me.Button == tui.MouseButtonWheelDown:
			ta.scrollBy(3)
			ctx.Invalidate(ta.rect)

			return true
		case me.Button == tui.MouseButtonLeft && me.Action == tui.MousePress:
			if ctx != nil && ctx.RequestFocus != nil {
				ctx.RequestFocus(ta.id)
			}

			ta.positionCursorFromClick(me.X, me.Y)
			ta.anchor = ta.cursor // Set anchor for potential drag

			ta.desiredCol = -1
			if me.ClickCount >= 2 {
				ta.selectWordAtCursor()
			}

			ta.scrollToCursor()
			ctx.Invalidate(ta.rect)

			return true
		case me.Action == tui.MouseDrag:
			ta.positionCursorFromClick(me.X, me.Y)
			// anchor stays fixed from press
			ta.desiredCol = -1
			ta.scrollToCursor()
			ctx.Invalidate(ta.rect)

			return true
		}

		return false
	}

	// Handle paste events
	if pe, ok := e.(event.PasteEvent); ok {
		return ta.handlePaste(pe, ctx)
	}

	ke, ok := e.(tui.KeyEvent)
	if !ok {
		return false
	}

	if ctx.FocusedID != ta.id {
		return false
	}

	if ke.Key == tui.KeyRune {
		if ta.readOnly {
			return false
		}

		insert := text.Sanitize(string(ke.Rune))
		if insert == "" {
			return true
		}

		if ta.hasSelection() {
			ta.deleteSelection()
		}

		ta.insertText(insert)
		ta.anchor = -1
		ta.desiredCol = -1
		ta.scrollToCursor()
		ctx.Invalidate(ta.rect)

		return true
	}

	return false
}

// handlePaste inserts pasted text.
func (ta *TextArea) handlePaste(e event.PasteEvent, ctx *tui.Ctx) bool {
	if ctx.FocusedID != ta.id || ta.readOnly {
		return false
	}

	insert := text.Sanitize(strings.ReplaceAll(e.Text, "\r\n", "\n"))
	if insert == "" {
		return true
	}

	ta.insertText(insert)
	ta.desiredCol = -1
	ta.scrollToCursor()
	ctx.Invalidate(ta.rect)

	return true
}

// HandleAction handles semantic actions.
func (ta *TextArea) HandleAction(act int, ctx *tui.Ctx) bool {
	if ctx.FocusedID != ta.id {
		return false
	}

	switch ui.Action(act) {
	case ui.ActionMoveLeft:
		if isShift(ctx) {
			ta.ensureAnchor()
		} else {
			ta.anchor = -1
		}

		ta.cursor = text.PrevCluster(ta.text, ta.cursor)
		ta.desiredCol = -1
		ta.scrollToCursor()
		ctx.Invalidate(ta.rect)

		return true
	case ui.ActionMoveRight:
		if isShift(ctx) {
			ta.ensureAnchor()
		} else {
			ta.anchor = -1
		}

		ta.cursor = text.NextCluster(ta.text, ta.cursor)
		ta.desiredCol = -1
		ta.scrollToCursor()
		ctx.Invalidate(ta.rect)

		return true
	case ui.ActionMoveUp:
		if isShift(ctx) {
			ta.ensureAnchor()
		} else {
			ta.anchor = -1
		}

		ta.moveCursorVertical(-1)
		ta.scrollToCursor()
		ctx.Invalidate(ta.rect)

		return true
	case ui.ActionMoveDown:
		if isShift(ctx) {
			ta.ensureAnchor()
		} else {
			ta.anchor = -1
		}

		ta.moveCursorVertical(1)
		ta.scrollToCursor()
		ctx.Invalidate(ta.rect)

		return true
	case ui.ActionHome:
		if isShift(ctx) {
			ta.ensureAnchor()
		} else {
			ta.anchor = -1
		}

		line := ta.cursorLine()
		ta.cursor = ta.lines[line].startByte
		ta.desiredCol = -1
		ta.scrollToCursor()
		ctx.Invalidate(ta.rect)

		return true
	case ui.ActionEnd:
		if isShift(ctx) {
			ta.ensureAnchor()
		} else {
			ta.anchor = -1
		}

		line := ta.cursorLine()
		ta.cursor = ta.lines[line].endByte
		ta.desiredCol = -1
		ta.scrollToCursor()
		ctx.Invalidate(ta.rect)

		return true
	case ui.ActionDeleteBackward:
		if ta.readOnly {
			return false
		}

		if ta.hasSelection() {
			ta.deleteSelection()
		} else {
			ta.text, ta.cursor = text.DeletePrevCluster(ta.text, ta.cursor)
			ta.rebuildLineIndex()
		}

		ta.anchor = -1
		ta.desiredCol = -1
		ta.scrollToCursor()
		ctx.Invalidate(ta.rect)

		return true
	case ui.ActionDeleteForward:
		if ta.readOnly {
			return false
		}

		if ta.hasSelection() {
			ta.deleteSelection()
		} else {
			ta.text, ta.cursor = text.DeleteNextCluster(ta.text, ta.cursor)
			ta.rebuildLineIndex()
		}

		ta.anchor = -1
		ta.desiredCol = -1
		ta.scrollToCursor()
		ctx.Invalidate(ta.rect)

		return true
	case ui.ActionSelectAll:
		ta.anchor = 0
		ta.cursor = len(ta.text)
		ta.desiredCol = -1
		ta.scrollToCursor()
		ctx.Invalidate(ta.rect)

		return true
	case ui.ActionCopy:
		if ta.hasSelection() && ctx.ClipboardWrite != nil {
			ctx.ClipboardWrite(ta.selectedText())
		}

		return true
	case ui.ActionCut:
		if ta.readOnly {
			return false
		}

		if ta.hasSelection() {
			if ctx.ClipboardWrite != nil {
				ctx.ClipboardWrite(ta.selectedText())
			}

			ta.deleteSelection()
			ta.desiredCol = -1
			ta.scrollToCursor()
			ctx.Invalidate(ta.rect)
		}

		return true
	case ui.ActionSubmit:
		// Enter inserts a newline in TextArea
		if ta.readOnly {
			return false
		}

		ta.insertText("\n")
		ta.desiredCol = -1
		ta.scrollToCursor()
		ctx.Invalidate(ta.rect)

		return true
	case ui.ActionPageUp:
		h := ta.rect.H
		if h <= 0 {
			h = 1
		}

		for range h {
			ta.moveCursorVertical(-1)
		}

		ta.scrollToCursor()
		ctx.Invalidate(ta.rect)

		return true
	case ui.ActionPageDown:
		h := ta.rect.H
		if h <= 0 {
			h = 1
		}

		for range h {
			ta.moveCursorVertical(1)
		}

		ta.scrollToCursor()
		ctx.Invalidate(ta.rect)

		return true
	}

	return false
}

// moveCursorVertical moves the cursor up (dir=-1) or down (dir=1),
// preserving the desiredCol for column stickiness.
func (ta *TextArea) moveCursorVertical(dir int) {
	curLine := ta.cursorLine()

	targetLine := curLine + dir
	if targetLine < 0 || targetLine >= len(ta.lines) {
		return
	}

	if ta.desiredCol == -1 {
		ta.desiredCol = ta.cursorCol()
	}

	ln := ta.lines[targetLine]
	lineText := ta.text[ln.startByte:ln.endByte]
	byteInLine := text.OffsetAtColumnBias(lineText, ta.desiredCol, text.BiasLeft)
	byteInLine = text.ClampCluster(lineText, byteInLine)
	ta.cursor = ln.startByte + byteInLine
}

// insertText inserts s at cursor position, handling newlines.
func (ta *TextArea) insertText(s string) {
	ta.cursor = text.ClampCluster(ta.text, ta.cursor)
	ta.text = ta.text[:ta.cursor] + s + ta.text[ta.cursor:]
	ta.cursor += len(s)
	ta.cursor = text.ClampCluster(ta.text, ta.cursor)
	ta.rebuildLineIndex()
}

// hasSelection reports whether a selection is active.
func (ta *TextArea) hasSelection() bool {
	return ta.anchor >= 0 && ta.anchor != ta.cursor
}

// selectionRange returns the normalized selection range.
func (ta *TextArea) selectionRange() text.Range {
	if ta.anchor < 0 {
		return text.Range{Start: ta.cursor, End: ta.cursor}
	}

	return text.Range{Start: ta.anchor, End: ta.cursor}.Normalized()
}

// selectedText returns the currently selected text.
func (ta *TextArea) selectedText() string {
	if !ta.hasSelection() {
		return ""
	}

	r := ta.selectionRange()

	return ta.text[r.Start:r.End]
}

// deleteSelection removes the selected text and clears the anchor.
func (ta *TextArea) deleteSelection() {
	if !ta.hasSelection() {
		return
	}

	r := ta.selectionRange()
	ta.text, _ = text.DeleteRange(ta.text, r)
	ta.cursor = r.Start
	ta.anchor = -1
	ta.rebuildLineIndex()
}

// ensureAnchor sets the anchor to cursor if not already set.
func (ta *TextArea) ensureAnchor() {
	if ta.anchor < 0 {
		ta.anchor = ta.cursor
	}
}

// isShift checks if shift is held in the context.
func isShift(ctx *tui.Ctx) bool {
	return ctx.Mod&tui.ModShift != 0
}

// selectWordAtCursor selects the word at the current cursor position.
func (ta *TextArea) selectWordAtCursor() {
	if len(ta.text) == 0 {
		return
	}
	// Find word boundaries (simple: non-space characters).
	start := ta.cursor
	for start > 0 {
		prev := text.PrevCluster(ta.text, start)
		if prev >= start {
			break
		}

		ch := ta.text[prev]
		if ch == ' ' || ch == '\t' || ch == '\n' {
			break
		}

		start = prev
	}

	end := ta.cursor
	for end < len(ta.text) {
		ch := ta.text[end]
		if ch == ' ' || ch == '\t' || ch == '\n' {
			break
		}

		next := text.NextCluster(ta.text, end)
		if next <= end {
			break
		}

		end = next
	}

	ta.anchor = start
	ta.cursor = end
}

// positionCursorFromClick sets cursor from screen coordinates.
func (ta *TextArea) positionCursorFromClick(clickX, clickY int) {
	lineIdx := ta.scrollY + (clickY - ta.rect.Y)
	if lineIdx < 0 {
		lineIdx = 0
	}

	if lineIdx >= len(ta.lines) {
		lineIdx = len(ta.lines) - 1
	}

	col := clickX - ta.rect.X + ta.scrollX
	if col < 0 {
		col = 0
	}

	ln := ta.lines[lineIdx]
	lineText := ta.text[ln.startByte:ln.endByte]
	byteInLine := text.OffsetAtColumnBias(lineText, col, text.BiasLeft)
	byteInLine = text.ClampCluster(lineText, byteInLine)
	ta.cursor = ln.startByte + byteInLine
}
