package main

import (
	"testing"
	"time"

	"github.com/losinggeneration/rovel"
	"github.com/losinggeneration/rovel/backend/memory"
	"github.com/losinggeneration/rovel/event"
	"github.com/losinggeneration/rovel/geom"
	"github.com/losinggeneration/rovel/style"
	"github.com/losinggeneration/rovel/ui/overlay"
)

// The demo composes ordinary views; these tests drive that exact composition
// through the memory backend — show, focus-follows-raise, drag, click-raise,
// and close — the same black-box seam as the library tests.

func waitRune(t *testing.T, be *memory.Backend, x, y int, want rune, what string) {
	t.Helper()

	deadline := time.After(2 * time.Second)

	for {
		frame, ok := be.LastFrame()
		if ok && x < frame.W && y < frame.H && frame.RowRunes(y)[x] == want {
			return
		}

		select {
		case <-deadline:
			t.Fatalf("timeout waiting for %q at (%d,%d)", what, x, y)
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}
}

// These mirror the window's own layout so the assertions below read as
// intent ("the title", "the close box") rather than magic cells.
func titleCell(r geom.Rect, title string) geom.Point {
	label := " " + title + " "

	// +1 skips the leading space, landing on the title's first letter.
	return geom.Point{X: r.X + (r.W-shadowW-len(label))/2 + 1, Y: r.Y}
}

func closeCell(r geom.Rect) geom.Point { return geom.Point{X: r.X + 2, Y: r.Y} }

func inputCell(r geom.Rect) geom.Point { return geom.Point{X: r.X + 2, Y: r.Y + 2} }

// grabCell is a spot on the top edge clear of both the close box and title.
func grabCell(r geom.Rect) geom.Point { return geom.Point{X: r.X + r.W - shadowW - 2, Y: r.Y} }

// frameCornerTL is the one corner no focus change ever damages, which makes it
// the honest place to check that losing focus repainted the whole frame.
func frameCornerTL(r geom.Rect) geom.Point { return geom.Point{X: r.X, Y: r.Y} }

// stubDrawer swallows painting; this test only cares which rects the window
// asks to have repainted.
type stubDrawer struct{}

func (stubDrawer) FillRect(geom.Rect, style.Style)               {}
func (stubDrawer) DrawText(geom.Point, string, style.Style)      {}
func (stubDrawer) DrawBorder(geom.Rect, rovel.BoxStyle)          {}
func (stubDrawer) ClipRect() geom.Rect                           { return geom.Rect{W: 200, H: 200} }
func (d stubDrawer) WithClip(_ geom.Rect, fn func(rovel.Drawer)) { fn(d) }
func (d stubDrawer) WithOffset(_, _ int, fn func(rovel.Drawer))  { fn(d) }

// A focus change damages only the focused view's own rect — the input's single
// row — but it changes every glyph in the window's frame. Unless the window
// asks for a full repaint, the terminal (which is only sent damaged runs)
// keeps showing the double border of a window that no longer has the keyboard.
func TestWindowRepaintsWholeFrameOnFocusChange(t *testing.T) {
	rect := geom.Rect{X: 2, Y: 1, W: 30, H: 9}
	w := newWindow("Notes", &overlay.Floating{Rect: rect})
	w.Layout(rect)

	var invalidated []geom.Rect

	ctx := &rovel.Ctx{
		Theme:      rovel.DefaultTheme(),
		Invalidate: func(r geom.Rect) { invalidated = append(invalidated, r) },
	}

	// Painting unfocused twice: no state change, nothing to repair.
	w.Paint(stubDrawer{}, ctx)
	w.Paint(stubDrawer{}, ctx)

	if len(invalidated) != 0 {
		t.Fatalf("steady unfocused paint invalidated %v, want none", invalidated)
	}

	// Gaining focus, then losing it, must each repaint the whole window.
	ctx.FocusedID = w.input.ID()
	w.Paint(stubDrawer{}, ctx)

	ctx.FocusedID = 0
	w.Paint(stubDrawer{}, ctx)

	if len(invalidated) != 2 {
		t.Fatalf("focus gain and loss invalidated %v, want two full-window repaints", invalidated)
	}

	for i, got := range invalidated {
		if got != rect {
			t.Errorf("invalidated[%d] = %v, want the whole window %v", i, got, rect)
		}
	}
}

func TestDemo_MovableWindows(t *testing.T) {
	be := memory.New(geom.Size{W: 80, H: 24})
	caps := style.Capability{HasBasic: true}

	app, err := rovel.New(rovel.AppOpts{Backend: be, Capability: &caps, Theme: rovel.DefaultTheme()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	app.SetRoot(newDesktop())

	if err := app.Enable(); err != nil {
		t.Fatalf("Enable: %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- app.Run() }()

	t.Cleanup(func() {
		app.Quit()
		be.Close()

		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("timeout waiting for Run to return")
		}
	})

	notesRect := geom.Rect{X: 2, Y: 1, W: 30, H: 9}
	searchRect := geom.Rect{X: 12, Y: 8, W: 28, H: 6}

	notes := newWindow("Notes", &overlay.Floating{Rect: notesRect})
	search := newWindow("Search", &overlay.Floating{Rect: searchRect})

	for _, w := range []*window{notes, search} {
		w := w
		err := app.Post(func(ctx *rovel.UpdateCtx) {
			o := ctx.ShowOverlay(rovel.OverlayOpts{Root: w, Place: w.place})
			w.overlayID = o.ID()
		})
		if err != nil {
			t.Fatalf("Post ShowOverlay: %v", err)
		}
	}

	// Both windows visible; Search (shown last) is on top.
	notesTitle := titleCell(notesRect, "Notes")
	searchTitle := titleCell(searchRect, "Search")

	waitRune(t, be, notesTitle.X, notesTitle.Y, 'N', "Notes title")
	waitRune(t, be, searchTitle.X, searchTitle.Y, 'S', "Search title")

	// Press Search's frame: it gains focus, so typing lands in its input.
	searchGrab := grabCell(searchRect)
	searchInput := inputCell(searchRect)

	be.SendEvent(event.MouseEvent{X: searchGrab.X, Y: searchGrab.Y, Button: event.MouseButtonLeft, Action: event.MousePress})
	be.SendEvent(event.MouseEvent{X: searchGrab.X, Y: searchGrab.Y, Button: event.MouseButtonLeft, Action: event.MouseRelease})

	be.SendEvent(event.KeyEvent{Key: event.KeyRune, Rune: 'Q'})
	waitRune(t, be, searchInput.X, searchInput.Y, 'Q', "typed rune in focused Search input")

	// Press Notes's exposed frame and drag it right: Notes raises above
	// Search (its frame now covers Search's title) and the title moves.
	notesGrab := grabCell(notesRect)
	const dragBy = 5

	be.SendEvent(event.MouseEvent{X: notesGrab.X, Y: notesGrab.Y, Button: event.MouseButtonLeft, Action: event.MousePress})
	be.SendEvent(event.MouseEvent{X: notesGrab.X + dragBy, Y: notesGrab.Y, Button: event.MouseButtonLeft, Action: event.MouseMove})
	be.SendEvent(event.MouseEvent{X: notesGrab.X + dragBy, Y: notesGrab.Y, Button: event.MouseButtonLeft, Action: event.MouseRelease})

	movedRect := notesRect
	movedRect.X += dragBy
	movedTitle := titleCell(movedRect, "Notes")

	waitRune(t, be, movedTitle.X, movedTitle.Y, 'N', "Notes title after drag")
	// '═' because the raised window is now the active one, so its frame is
	// drawn double-line.
	waitRune(t, be, searchTitle.X, searchTitle.Y, '═', "Notes frame covers Search title after raise")

	// Click Search's input: the Clickable wrapper raises Search again (and
	// focuses the input), so its title reappears.
	be.SendEvent(event.MouseEvent{X: searchInput.X, Y: searchInput.Y, Button: event.MouseButtonLeft, Action: event.MousePress})
	be.SendEvent(event.MouseEvent{X: searchInput.X, Y: searchInput.Y, Button: event.MouseButtonLeft, Action: event.MouseRelease})

	waitRune(t, be, searchTitle.X, searchTitle.Y, 'S', "Search title back after click-raise")

	// The [■] box on the frame closes Search, baring the desktop behind it.
	searchClose := closeCell(searchRect)

	be.SendEvent(event.MouseEvent{X: searchClose.X, Y: searchClose.Y, Button: event.MouseButtonLeft, Action: event.MousePress})
	be.SendEvent(event.MouseEvent{X: searchClose.X, Y: searchClose.Y, Button: event.MouseButtonLeft, Action: event.MouseRelease})

	waitRune(t, be, searchInput.X, searchInput.Y, '░', "desktop behind Search after close")

	// Notes is still there and draggable.
	secondGrab := grabCell(movedRect)

	be.SendEvent(event.MouseEvent{X: secondGrab.X, Y: secondGrab.Y, Button: event.MouseButtonLeft, Action: event.MousePress})
	be.SendEvent(event.MouseEvent{X: secondGrab.X + 2, Y: secondGrab.Y + 1, Button: event.MouseButtonLeft, Action: event.MouseMove})
	be.SendEvent(event.MouseEvent{X: secondGrab.X + 2, Y: secondGrab.Y + 1, Button: event.MouseButtonLeft, Action: event.MouseRelease})

	finalRect := movedRect
	finalRect.X += 2
	finalRect.Y++
	finalTitle := titleCell(finalRect, "Notes")

	waitRune(t, be, finalTitle.X, finalTitle.Y, 'N', "Notes title after second drag")
}
