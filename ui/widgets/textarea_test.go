package widgets

import (
	"testing"

	"github.com/losinggeneration/tui"
	"github.com/losinggeneration/tui/geom"
	"github.com/losinggeneration/tui/ui"
)

func mkTextAreaCtx(ta *TextArea) *tui.Ctx {
	return &tui.Ctx{
		FocusedID:        ta.ID(),
		Invalidate:       func(r geom.Rect) {},
		InvalidateAll:    func() {},
		InvalidateLayout: func(id tui.ID) {},
		RequestFocus:     func(id tui.ID) {},
		Quit:             func() {},
	}
}

func TestTextArea_LineIndex_Empty(t *testing.T) {
	ta := NewTextArea()
	if len(ta.lines) != 1 {
		t.Fatalf("empty text: want 1 line, got %d", len(ta.lines))
	}

	ln := ta.lines[0]
	if ln.startByte != 0 || ln.endByte != 0 {
		t.Errorf("empty line: got start=%d end=%d, want 0,0", ln.startByte, ln.endByte)
	}
}

func TestTextArea_LineIndex_SingleLine(t *testing.T) {
	ta := NewTextArea()
	ctx := mkTextAreaCtx(ta)
	ta.SetText(ctx, "hello")

	if len(ta.lines) != 1 {
		t.Fatalf("single line: want 1, got %d", len(ta.lines))
	}

	if ta.lines[0].startByte != 0 || ta.lines[0].endByte != 5 {
		t.Errorf("got start=%d end=%d", ta.lines[0].startByte, ta.lines[0].endByte)
	}
}

func TestTextArea_LineIndex_MultiLine(t *testing.T) {
	ta := NewTextArea()
	ctx := mkTextAreaCtx(ta)
	ta.SetText(ctx, "abc\ndef\nghi")

	if len(ta.lines) != 3 {
		t.Fatalf("want 3 lines, got %d", len(ta.lines))
	}

	want := []lineEntry{
		{0, 3},
		{4, 7},
		{8, 11},
	}
	for i, w := range want {
		got := ta.lines[i]
		if got.startByte != w.startByte || got.endByte != w.endByte {
			t.Errorf("line %d: got {%d,%d} want {%d,%d}", i, got.startByte, got.endByte, w.startByte, w.endByte)
		}
	}
}

func TestTextArea_LineIndex_TrailingNewline(t *testing.T) {
	ta := NewTextArea()
	ctx := mkTextAreaCtx(ta)
	ta.SetText(ctx, "abc\n")

	if len(ta.lines) != 2 {
		t.Fatalf("want 2 lines, got %d", len(ta.lines))
	}

	if ta.lines[1].startByte != 4 || ta.lines[1].endByte != 4 {
		t.Errorf("trailing empty line: got {%d,%d}", ta.lines[1].startByte, ta.lines[1].endByte)
	}
}

func TestTextArea_LineIndex_ConsecutiveNewlines(t *testing.T) {
	ta := NewTextArea()
	ctx := mkTextAreaCtx(ta)
	ta.SetText(ctx, "a\n\nb")

	if len(ta.lines) != 3 {
		t.Fatalf("want 3 lines, got %d", len(ta.lines))
	}
	// Line 1 is empty
	if ta.lines[1].startByte != 2 || ta.lines[1].endByte != 2 {
		t.Errorf("empty middle line: got {%d,%d}", ta.lines[1].startByte, ta.lines[1].endByte)
	}
}

func TestTextArea_UpDown_Basic(t *testing.T) {
	ta := NewTextArea()
	ctx := mkTextAreaCtx(ta)
	ta.SetText(ctx, "abc\ndef\nghi")

	// Cursor at end (line 2, col 3)
	// Move to start of line 0
	ta.cursor = 1 // 'b' on line 0
	ta.desiredCol = -1

	// Move down to line 1
	ta.HandleAction(int(ui.ActionMoveDown), ctx)

	if ta.cursorLine() != 1 {
		t.Errorf("after down: want line 1, got %d", ta.cursorLine())
	}

	if ta.cursor != 5 { // 'e' on line 1
		t.Errorf("after down: want cursor 5, got %d", ta.cursor)
	}

	// Move down to line 2
	ta.HandleAction(int(ui.ActionMoveDown), ctx)

	if ta.cursorLine() != 2 {
		t.Errorf("after 2nd down: want line 2, got %d", ta.cursorLine())
	}

	// Move up back to line 1
	ta.HandleAction(int(ui.ActionMoveUp), ctx)

	if ta.cursorLine() != 1 {
		t.Errorf("after up: want line 1, got %d", ta.cursorLine())
	}
}

func TestTextArea_DesiredCol_Stickiness(t *testing.T) {
	ta := NewTextArea()
	ctx := mkTextAreaCtx(ta)
	ta.SetText(ctx, "abcde\nab\nabcde")

	// Start at col 4 on line 0 (byte 4)
	ta.cursor = 4
	ta.desiredCol = -1

	// Move down to short line — should clamp to end (col 2)
	ta.HandleAction(int(ui.ActionMoveDown), ctx)

	if ta.cursorLine() != 1 {
		t.Fatalf("want line 1, got %d", ta.cursorLine())
	}
	// Cursor should be at end of short line
	if ta.cursor != ta.lines[1].endByte {
		t.Errorf("want cursor at end of short line (%d), got %d", ta.lines[1].endByte, ta.cursor)
	}

	// Move down again to long line — should restore col 4
	ta.HandleAction(int(ui.ActionMoveDown), ctx)

	if ta.cursorLine() != 2 {
		t.Fatalf("want line 2, got %d", ta.cursorLine())
	}

	wantCol := 4

	gotCol := ta.cursorCol()
	if gotCol != wantCol {
		t.Errorf("desiredCol stickiness: want col %d, got %d (cursor=%d)", wantCol, gotCol, ta.cursor)
	}
}

func TestTextArea_Insert_And_LineIndex(t *testing.T) {
	ta := NewTextArea()
	ctx := mkTextAreaCtx(ta)
	ta.SetText(ctx, "ab")

	ta.cursor = 1
	ta.insertText("X")

	if ta.text != "aXb" {
		t.Errorf("after insert: got %q, want %q", ta.text, "aXb")
	}

	if ta.cursor != 2 {
		t.Errorf("cursor after insert: got %d, want 2", ta.cursor)
	}

	if len(ta.lines) != 1 {
		t.Errorf("still 1 line, got %d", len(ta.lines))
	}
}

func TestTextArea_Delete_And_LineIndex(t *testing.T) {
	ta := NewTextArea()
	ctx := mkTextAreaCtx(ta)
	ta.SetText(ctx, "abc\ndef")

	// Delete \n to join lines: cursor at byte 3 (the \n)
	ta.cursor = 3
	ta.HandleAction(int(ui.ActionDeleteForward), ctx)

	if ta.text != "abcdef" {
		t.Errorf("after delete: got %q, want %q", ta.text, "abcdef")
	}

	if len(ta.lines) != 1 {
		t.Errorf("after joining: want 1 line, got %d", len(ta.lines))
	}
}

func TestTextArea_Home_End(t *testing.T) {
	ta := NewTextArea()
	ctx := mkTextAreaCtx(ta)
	ta.SetText(ctx, "abc\ndef")

	// Cursor in middle of line 1
	ta.cursor = 5 // 'e'

	ta.HandleAction(int(ui.ActionHome), ctx)

	if ta.cursor != 4 { // start of "def"
		t.Errorf("Home: want cursor 4, got %d", ta.cursor)
	}

	ta.HandleAction(int(ui.ActionEnd), ctx)

	if ta.cursor != 7 { // end of "def"
		t.Errorf("End: want cursor 7, got %d", ta.cursor)
	}
}

func TestTextArea_ScrollToCursor(t *testing.T) {
	ta := NewTextArea()
	ctx := mkTextAreaCtx(ta)
	ta.rect = tui.Rect{X: 0, Y: 0, W: 20, H: 3}
	ta.SetText(ctx, "line0\nline1\nline2\nline3\nline4")

	// Cursor on line 4 — should scroll
	ta.cursor = ta.lines[4].startByte
	ta.scrollToCursor()

	if ta.scrollY > 4 || ta.scrollY+ta.rect.H <= 4 {
		t.Errorf("scrollY=%d doesn't show line 4 with H=3", ta.scrollY)
	}
}

func TestTextArea_Enter_SplitsLine(t *testing.T) {
	ta := NewTextArea()
	ctx := mkTextAreaCtx(ta)
	ta.SetText(ctx, "abcd")
	ta.rect = tui.Rect{X: 0, Y: 0, W: 20, H: 5}

	// Insert newline at position 2
	ta.cursor = 2
	ta.HandleAction(int(ui.ActionSubmit), ctx)

	if ta.text != "ab\ncd" {
		t.Errorf("after Enter: got %q, want %q", ta.text, "ab\ncd")
	}

	if len(ta.lines) != 2 {
		t.Errorf("after Enter: want 2 lines, got %d", len(ta.lines))
	}

	if ta.cursor != 3 { // after the \n
		t.Errorf("cursor after Enter: want 3, got %d", ta.cursor)
	}
}

func TestTextArea_ShiftRight_CreatesSelection(t *testing.T) {
	ta := NewTextArea()
	ctx := mkTextAreaCtx(ta)
	ta.SetText(ctx, "hello")
	ta.rect = tui.Rect{X: 0, Y: 0, W: 20, H: 5}

	ta.cursor = 0
	ta.anchor = -1

	// Simulate shift+right
	ctx.Mod = tui.ModShift
	ta.HandleAction(int(ui.ActionMoveRight), ctx)

	if !ta.hasSelection() {
		t.Fatal("expected selection after shift+right")
	}

	if ta.anchor != 0 {
		t.Errorf("anchor: want 0, got %d", ta.anchor)
	}

	if ta.cursor <= 0 {
		t.Error("cursor should have advanced")
	}
}

func TestTextArea_TypeWithSelection_Replaces(t *testing.T) {
	ta := NewTextArea()
	ctx := mkTextAreaCtx(ta)
	ta.SetText(ctx, "hello")
	ta.rect = tui.Rect{X: 0, Y: 0, W: 20, H: 5}

	// Select "hel"
	ta.cursor = 3
	ta.anchor = 0

	// Type 'X' — should replace selection
	ke := tui.KeyEvent{Key: tui.KeyRune, Rune: 'X'}
	ta.Handle(ke, ctx)

	if ta.text != "Xlo" {
		t.Errorf("after typing with selection: got %q, want %q", ta.text, "Xlo")
	}
}

func TestTextArea_BackspaceWithSelection_DeletesRange(t *testing.T) {
	ta := NewTextArea()
	ctx := mkTextAreaCtx(ta)
	ta.SetText(ctx, "hello world")
	ta.rect = tui.Rect{X: 0, Y: 0, W: 20, H: 5}

	// Select "lo wo"
	ta.anchor = 3
	ta.cursor = 8

	ta.HandleAction(int(ui.ActionDeleteBackward), ctx)

	if ta.text != "helrld" {
		t.Errorf("after backspace with selection: got %q, want %q", ta.text, "helrld")
	}
}

func TestTextArea_CrossLineSelection(t *testing.T) {
	ta := NewTextArea()
	ctx := mkTextAreaCtx(ta)
	ta.SetText(ctx, "abc\ndef\nghi")
	ta.rect = tui.Rect{X: 0, Y: 0, W: 20, H: 5}

	// Select from middle of line 0 to middle of line 1
	ta.anchor = 1 // 'b'
	ta.cursor = 5 // 'e'

	sel := ta.selectedText()

	want := "bc\nd"
	if sel != want {
		t.Errorf("cross-line selection: got %q, want %q", sel, want)
	}
}

func TestTextArea_SelectAll(t *testing.T) {
	ta := NewTextArea()
	ctx := mkTextAreaCtx(ta)
	ta.SetText(ctx, "abc\ndef")
	ta.rect = tui.Rect{X: 0, Y: 0, W: 20, H: 5}

	ta.HandleAction(int(ui.ActionSelectAll), ctx)

	if ta.anchor != 0 || ta.cursor != len(ta.text) {
		t.Errorf("select all: anchor=%d cursor=%d, want 0 and %d", ta.anchor, ta.cursor, len(ta.text))
	}

	if ta.selectedText() != "abc\ndef" {
		t.Errorf("select all text: got %q", ta.selectedText())
	}
}

func TestTextArea_Backspace_JoinsLines(t *testing.T) {
	ta := NewTextArea()
	ctx := mkTextAreaCtx(ta)
	ta.SetText(ctx, "ab\ncd")
	ta.rect = tui.Rect{X: 0, Y: 0, W: 20, H: 5}

	// Cursor at start of line 1 (byte 3, first char of "cd")
	ta.cursor = 3
	ta.HandleAction(int(ui.ActionDeleteBackward), ctx)

	if ta.text != "abcd" {
		t.Errorf("after backspace: got %q, want %q", ta.text, "abcd")
	}

	if len(ta.lines) != 1 {
		t.Errorf("after backspace: want 1 line, got %d", len(ta.lines))
	}
}
