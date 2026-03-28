// Package main demonstrates the tui toolkit with a 3-pane MVP app.
//
// The app features:
// - Left pane: Form with name/email fields and submit button
// - Center pane: Editor-like canvas with text editing
// - Right pane: VirtualList with 100k items
// - Top: Status bar showing app feedback
// - Tab focus traversal between panes
// - Esc or Ctrl+C to quit
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/losinggeneration/tui"
	"github.com/losinggeneration/tui/event"
	"github.com/losinggeneration/tui/geom"
	"github.com/losinggeneration/tui/style"
	"github.com/losinggeneration/tui/ui"
	"github.com/losinggeneration/tui/ui/layout"
	"github.com/losinggeneration/tui/ui/virtual"
	"github.com/losinggeneration/tui/ui/widgets"
	cellwidgets "github.com/losinggeneration/tui/ui/widgets/cell"
)

const (
	ActionQuit ui.Action = iota + 100
	ActionPerfToggle
)

type appKeymap struct{}

func (appKeymap) Resolve(ctx ui.KeyContext, k ui.Keystroke) (ui.Action, bool) {
	if ctx == ui.KeyCtxTextInput {
		switch k.Key {
		case event.KeyEsc, event.KeyCtrlC:
			return ActionQuit, true
		default:
		}

		return ui.ActionNone, false
	}

	switch k.Key {
	case event.KeyEsc, event.KeyCtrlC:
		return ActionQuit, true
	default:
	}

	if k.Key == event.KeyRune && (k.Rune == 'P' || k.Rune == 'p') {
		return ActionPerfToggle, true
	}

	return ui.ActionNone, false
}

// Root embeds the main layout and handles app-level actions.
type Root struct {
	*layout.VStack

	state *appState
	app   *tui.App
}

func (r *Root) HandleAction(act int, ctx *tui.Ctx) bool {
	switch ui.Action(act) {
	case ActionQuit:
		r.app.Quit()

		return true
	case ActionPerfToggle:
		if r.state.perfEnabled && r.state.perfMonitor != nil {
			r.state.status = r.state.perfMonitor.Summary()

			if ctx != nil && ctx.InvalidateAll != nil {
				ctx.InvalidateAll()
			}

			return true
		}
	default:
	}

	return false
}

type appState struct {
	status string

	// Form state
	name  string
	email string

	// Editor state
	lines   []string
	cursorX int
	cursorY int

	// Performance monitoring
	perfEnabled bool
	perfMonitor *PerfMonitor
}

func newAppState(perfEnabled bool) *appState {
	st := &appState{
		status: "Tab to move focus. Esc or Ctrl+C to quit.",
		name:   "",
		email:  "",
		lines: []string{
			"This is a simple editor-like canvas.",
			"You can type text and move the cursor.",
			"",
			"Try typing here:",
		},
		cursorX:     0,
		cursorY:     3,
		perfEnabled: perfEnabled,
	}

	if perfEnabled {
		st.perfMonitor = NewPerfMonitor(1*time.Millisecond, 500*time.Microsecond)
		st.status += " Press 'P' for performance report."
	}

	return st
}

func main() {
	perfFlag := flag.Bool("perf", false, "Enable performance monitoring")
	flag.Parse()

	st := newAppState(*perfFlag)

	app, err := tui.New(tui.AppOpts{
		Theme: tui.Theme{
			Base: style.Style{
				FG:   style.ColorDefault,
				BG:   style.ColorDefault,
				Attr: 0,
			},
			Palette: tui.Palette{
				Focus: style.Style{
					FG:   style.ColorBlack,
					BG:   style.ColorWhite,
					Attr: 0,
				},
			},
		},
		ResolveAction: ui.NewResolver(appKeymap{}),
	})
	if err != nil {
		panic(err)
	}

	// Create editor canvas
	var editor *cellwidgets.Canvas

	editor = cellwidgets.NewCanvasOpts(cellwidgets.CanvasOpts{
		Focusable: true,
		MinSize:   geom.Size{W: 20, H: 10},
		Paint: func(p *tui.Painter, rect geom.Rect, ctx *tui.Ctx) {
			paintEditor(st, p, rect, ctx)
		},
		Handle: func(e tui.Event, ctx *tui.Ctx) bool {
			return handleEditor(st, editorRef{canvas: editor}, e, ctx)
		},
	})

	// Create virtual list
	baseHistory := virtual.NewVirtualList(virtual.VirtualListOpts{
		RowHeight: 1,
		Count: func() int {
			return 100000
		},
		RenderRow: func(
			i int,
			selected bool,
			focused bool,
			p *tui.Painter,
			r geom.Rect,
		) {
			if r.W <= 0 {
				return
			}

			row := fmt.Sprintf("Item %d", i)

			switch {
			case selected && focused:
				// Selected + focused: reverse video
				p.Fill(r, ' ', tui.Style{Attr: style.AttrReverse})

				truncated := truncate(row, r.W-1)
				if r.W > 1 {
					p.Text(r.X, r.Y, ">"+truncated, tui.Style{Attr: style.AttrReverse})
				}

			case selected:
				// Selected but unfocused: lighter treatment
				p.Fill(r, ' ', tui.Style{FG: style.ColorWhite, BG: style.ColorBlue})

				truncated := truncate(row, r.W-1)
				if r.W > 1 {
					p.Text(r.X, r.Y, ">"+truncated, tui.Style{FG: style.ColorWhite, BG: style.ColorBlue})
				}

			default:
				// Normal row
				truncated := truncate(row, r.W)
				p.Text(r.X, r.Y, truncated, tui.Style{})
			}
		},
		OnActivate: func(i int, ctx *tui.Ctx) {
			st.status = fmt.Sprintf("Activated item %d", i)

			if ctx != nil && ctx.InvalidateAll != nil {
				ctx.InvalidateAll()
			}
		},
	})

	var history tui.View = baseHistory
	if st.perfEnabled && st.perfMonitor != nil {
		history = NewInstrumentedVirtualList("history", baseHistory, st.perfMonitor)
	}

	// Create form pane
	formPane := buildFormPane(st)

	// Create status bar
	statusBar := buildStatusBar(st)

	// Build root layout with app reference for quit handling
	root := buildRoot(st, app, statusBar, formPane, editor, history)

	app.SetRoot(root)

	if err := app.Enable(); err != nil {
		panic(err)
	}

	defer func() {
		err := app.Restore()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to restore terminal: %v\n", err)
		}
	}()

	if err := app.Run(); err != nil {
		panic(err)
	}
}

type editorRef struct {
	canvas *cellwidgets.Canvas
}

// paintEditor renders the editor canvas.
func paintEditor(st *appState, p *tui.Painter, rect geom.Rect, ctx *tui.Ctx) {
	if rect.W <= 0 || rect.H <= 0 {
		return
	}

	// Draw visible lines
	for i := 0; i < len(st.lines) && i < rect.H; i++ {
		lineY := rect.Y + i
		line := truncate(st.lines[i], rect.W)
		padded := padRight(line, rect.W)
		p.Text(rect.X, lineY, padded, tui.Style{})
	}

	// Draw cursor
	if st.cursorY >= 0 && st.cursorY < len(st.lines) && st.cursorY < rect.H {
		line := st.lines[st.cursorY]
		runes := []rune(line)

		// Calculate cursor display position (not rune index)
		cursorDisplayX := 0
		for i := 0; i < st.cursorX && i < len(runes); i++ {
			cursorDisplayX += tui.RuneWidth(runes[i])
		}

		cx := rect.X + cursorDisplayX

		cy := rect.Y + st.cursorY

		if cx >= rect.X && cx < rect.X+rect.W {
			ch := " "
			if st.cursorX >= 0 && st.cursorX < len(runes) {
				ch = string(runes[st.cursorX])
			}

			p.Text(cx, cy, ch, tui.Style{Attr: style.AttrReverse})
		}
	}
}

// handleEditor processes keyboard events for the editor.
func handleEditor(
	st *appState,
	ref editorRef,
	e tui.Event,
	ctx *tui.Ctx,
) bool {
	ke, ok := e.(tui.KeyEvent)
	if !ok {
		return false
	}

	_, oldY := st.cursorX, st.cursorY

	switch ke.Key {
	case tui.KeyRune:
		insertRune(st, ke.Rune)
		ref.canvas.InvalidateRow(ctx, st.cursorY)

		return true

	case tui.KeyLeft:
		moveLeft(st)
		invalidateCursorMove(ref.canvas, ctx, oldY, st.cursorY)

		return true

	case tui.KeyRight:
		moveRight(st)
		invalidateCursorMove(ref.canvas, ctx, oldY, st.cursorY)

		return true

	case tui.KeyUp:
		moveUp(st)
		invalidateCursorMove(ref.canvas, ctx, oldY, st.cursorY)

		return true

	case tui.KeyDown:
		moveDown(st)
		invalidateCursorMove(ref.canvas, ctx, oldY, st.cursorY)

		return true

	case tui.KeyEnter:
		splitLine(st)
		ref.canvas.Invalidate(ctx)

		return true

	case tui.KeyBackspace:
		backspace(st)
		ref.canvas.Invalidate(ctx)

		return true

	case tui.KeyEsc:
		// Don't handle - let quitWrapper handle it
		return false

	default:
		return false
	}
}

// invalidateCursorMove invalidates rows affected by cursor movement.
func invalidateCursorMove(
	c *cellwidgets.Canvas,
	ctx *tui.Ctx,
	oldRow int,
	newRow int,
) {
	if oldRow == newRow {
		c.InvalidateRow(ctx, oldRow)

		return
	}

	c.InvalidateRow(ctx, oldRow)
	c.InvalidateRow(ctx, newRow)
}

// insertRune inserts a rune at the cursor position.
func insertRune(st *appState, r rune) {
	ensureCursor(st)

	lineRunes := []rune(st.lines[st.cursorY])

	if st.cursorX < 0 {
		st.cursorX = 0
	}

	if st.cursorX > len(lineRunes) {
		st.cursorX = len(lineRunes)
	}

	lineRunes = append(
		lineRunes[:st.cursorX],
		append([]rune{r}, lineRunes[st.cursorX:]...)...,
	)
	st.lines[st.cursorY] = string(lineRunes)
	st.cursorX++
}

// moveLeft moves the cursor left.
func moveLeft(st *appState) {
	ensureCursor(st)

	if st.cursorX > 0 {
		st.cursorX--

		return
	}

	if st.cursorY > 0 {
		st.cursorY--
		st.cursorX = len([]rune(st.lines[st.cursorY]))
	}
}

// moveRight moves the cursor right.
func moveRight(st *appState) {
	ensureCursor(st)

	lineLen := len([]rune(st.lines[st.cursorY]))
	if st.cursorX < lineLen {
		st.cursorX++

		return
	}

	if st.cursorY+1 < len(st.lines) {
		st.cursorY++
		st.cursorX = 0
	}
}

// moveUp moves the cursor up.
func moveUp(st *appState) {
	ensureCursor(st)

	if st.cursorY > 0 {
		st.cursorY--
		clampCursorX(st)
	}
}

// moveDown moves the cursor down.
func moveDown(st *appState) {
	ensureCursor(st)

	if st.cursorY+1 < len(st.lines) {
		st.cursorY++
		clampCursorX(st)
	}
}

// splitLine splits the current line at the cursor position.
func splitLine(st *appState) {
	ensureCursor(st)

	lineRunes := []rune(st.lines[st.cursorY])

	if st.cursorX < 0 {
		st.cursorX = 0
	}

	if st.cursorX > len(lineRunes) {
		st.cursorX = len(lineRunes)
	}

	left := string(lineRunes[:st.cursorX])
	right := string(lineRunes[st.cursorX:])

	st.lines[st.cursorY] = left
	insertAt := st.cursorY + 1

	st.lines = append(st.lines, "")
	copy(st.lines[insertAt+1:], st.lines[insertAt:])
	st.lines[insertAt] = right

	st.cursorY++
	st.cursorX = 0
}

// backspace deletes the character before the cursor.
func backspace(st *appState) {
	ensureCursor(st)

	if st.cursorX > 0 {
		lineRunes := []rune(st.lines[st.cursorY])
		lineRunes = append(
			lineRunes[:st.cursorX-1],
			lineRunes[st.cursorX:]...,
		)
		st.lines[st.cursorY] = string(lineRunes)
		st.cursorX--

		return
	}

	if st.cursorY == 0 {
		return
	}

	prev := st.lines[st.cursorY-1]
	curr := st.lines[st.cursorY]
	prevLen := len([]rune(prev))

	st.lines[st.cursorY-1] = prev + curr
	st.lines = append(st.lines[:st.cursorY], st.lines[st.cursorY+1:]...)

	st.cursorY--
	st.cursorX = prevLen
}

// ensureCursor ensures the cursor position is valid.
func ensureCursor(st *appState) {
	if len(st.lines) == 0 {
		st.lines = []string{""}
	}

	if st.cursorY < 0 {
		st.cursorY = 0
	}

	if st.cursorY >= len(st.lines) {
		st.cursorY = len(st.lines) - 1
	}

	clampCursorX(st)
}

// clampCursorX ensures cursorX is within the current line bounds.
func clampCursorX(st *appState) {
	lineLen := len([]rune(st.lines[st.cursorY]))

	if st.cursorX < 0 {
		st.cursorX = 0
	}

	if st.cursorX > lineLen {
		st.cursorX = lineLen
	}
}

// truncate returns a string truncated to max display width.
func truncate(s string, w int) string {
	if w <= 0 {
		return ""
	}

	width := 0

	runes := []rune(s)
	for i, r := range runes {
		rw := tui.RuneWidth(r)
		if width+rw > w {
			return string(runes[:i])
		}

		width += rw
	}

	return s
}

// padRight pads a string with spaces to the given display width.
func padRight(s string, w int) string {
	width := 0
	for _, r := range s {
		width += tui.RuneWidth(r)
	}

	if width >= w {
		return truncate(s, w)
	}

	return s + strings.Repeat(" ", w-width)
}

// buildFormPane creates the form pane with name/email fields and submit button.
func buildFormPane(st *appState) tui.View {
	// Create a VStack for the form layout
	form := layout.NewVStack()

	// Name label
	nameLabel := widgets.NewLabel("Name:")
	form.Add(nameLabel)

	// Name input
	nameInput := widgets.NewTextInput()
	nameInput.SetText(nil, st.name)
	form.Add(nameInput)

	// Email label
	emailLabel := widgets.NewLabel("Email:")
	form.Add(emailLabel)

	// Email input
	emailInput := widgets.NewTextInput()
	emailInput.SetText(nil, st.email)
	form.Add(emailInput)

	// Submit button
	submitBtn := widgets.NewButton("Submit")
	submitBtn.SetOnPress(func(ctx *tui.Ctx) {
		st.name = nameInput.Text()
		st.email = emailInput.Text()
		st.status = fmt.Sprintf("Submitted: name=%q email=%q", st.name, st.email)

		if ctx != nil && ctx.InvalidateAll != nil {
			ctx.InvalidateAll()
		}
	})

	split := layout.NewHStack()
	split.Add(submitBtn)
	split.Add(widgets.NewLabel(""))
	form.Add(split)
	form.Add(widgets.NewLabel(""))

	// Wrap in a border for visual separation
	border := layout.NewBorder(form)
	border.SetTitle(" Form ")

	return border
}

// buildStatusBar creates the status bar at the top.
func buildStatusBar(st *appState) tui.View {
	statusCanvas := cellwidgets.NewCanvasOpts(cellwidgets.CanvasOpts{
		MinSize: geom.Size{W: 1, H: 1},
		Paint: func(p *tui.Painter, rect geom.Rect, ctx *tui.Ctx) {
			if rect.W <= 0 || rect.H <= 0 {
				return
			}
			// Draw status text with background
			status := truncate(st.status, rect.W)
			p.Fill(rect, ' ', tui.Style{FG: style.ColorBlack, BG: style.ColorCyan})
			p.Text(rect.X, rect.Y, status, tui.Style{FG: style.ColorBlack, BG: style.ColorCyan})
		},
		Handle: func(e tui.Event, ctx *tui.Ctx) bool {
			// Let events bubble up to quitWrapper
			return false
		},
	})

	return statusCanvas
}

// buildRoot creates the root layout with status bar and 3-pane main area.
func buildRoot(st *appState, app *tui.App, status tui.View, form tui.View, editor tui.View, history tui.View) *Root {
	root := layout.NewVStack()

	// Add status bar
	root.Add(status)

	// Create horizontal container for the 3 panes
	hpanes := layout.NewSplit(layout.Horizontal)

	// Left: Form (with border already)
	hpanes.SetFirst(form)

	// Center: Editor (with border)
	editorBorder := layout.NewBorder(editor)
	editorBorder.SetTitle(" Editor ")
	hpanes.SetSecond(editorBorder)

	// Right: History list (with border)
	historyBorder := layout.NewBorder(history)
	historyBorder.SetTitle(" History (100k) ")

	root.Add(hpanes)
	root.Add(historyBorder)

	return &Root{VStack: root, state: st, app: app}
}
