package main

import (
	"fmt"

	"github.com/losinggeneration/tui"
	"github.com/losinggeneration/tui/event"
	"github.com/losinggeneration/tui/ui/layout"
	"github.com/losinggeneration/tui/ui/widgets"
)

func main() {
	// Create app with default theme
	app, err := tui.New(tui.DefaultAppOpts())
	if err != nil {
		fmt.Printf("Failed to create app: %v\n", err)

		return
	}

	// Create widgets
	button1 := widgets.NewButton("Button 1")
	button2 := widgets.NewButton("Button 2")
	textInput := widgets.NewTextInput()

	// Create layout
	hstack := layout.NewHStack()
	hstack.Add(button1)
	hstack.Add(button2)

	border := layout.NewBorder(textInput)
	border.SetTitle("Input")

	vstack := layout.NewVStack()
	vstack.Add(hstack)
	vstack.Add(border)

	focusRing := widgets.NewFocusRing(vstack)

	// Create a wrapper that handles quit and button status
	quitHandler := &QuitHandler{
		app:      app,
		mainView: focusRing,
		id:       tui.NewID(),
		status:   "Tab to navigate, Enter to press buttons, 'q' to quit",
	}

	// Wire up button callbacks to update wrapper state
	button1.SetOnPress(func(ctx *tui.Ctx) {
		quitHandler.OnButton1(ctx)
	})
	button2.SetOnPress(func(ctx *tui.Ctx) {
		quitHandler.OnButton2(ctx)
	})

	// Set root and run
	app.SetRoot(quitHandler)

	if err := app.Enable(); err != nil {
		fmt.Printf("Failed to enable app: %v\n", err)

		return
	}

	defer func() {
		if err := app.Restore(); err != nil {
			fmt.Printf("Failed to restore terminal: %v\n", err)
		}
	}()

	// Run the app
	if err := app.Run(); err != nil {
		fmt.Printf("App error: %v\n", err)
	}
}

// QuitHandler handles the 'q' key to quit the app and wraps the main view.
// It also manages status display for button presses.
type QuitHandler struct {
	app      *tui.App
	mainView tui.View
	id       tui.ID
	rect     tui.Rect
	status   string
}

func (q *QuitHandler) ID() tui.ID {
	return q.id
}

func (q *QuitHandler) Rect() tui.Rect {
	return q.rect
}

func (q *QuitHandler) Layout(r tui.Rect) {
	q.rect = r
	// Reserve top row for status, rest for main view
	mainRect := tui.Rect{
		X: r.X,
		Y: r.Y + 1,
		W: r.W,
		H: r.H - 1,
	}
	q.mainView.Layout(mainRect)
}

func (q *QuitHandler) MinSize() tui.Size {
	mainMin := q.mainView.MinSize()

	return tui.Size{W: mainMin.W, H: mainMin.H + 1}
}

func (q *QuitHandler) Paint(p *tui.Painter, ctx *tui.Ctx) {
	// Paint status row
	p.Text(q.rect.X, q.rect.Y, q.status, ctx.Theme.Base)

	// Paint main view below status
	mainRect := tui.Rect{
		X: q.rect.X,
		Y: q.rect.Y + 1,
		W: q.rect.W,
		H: q.rect.H - 1,
	}
	p.WithClip(mainRect, func(p *tui.Painter) {
		q.mainView.Paint(p, ctx)
	})
}

func (q *QuitHandler) Handle(e tui.Event, ctx *tui.Ctx) bool {
	ke, ok := e.(event.KeyEvent)
	if !ok {
		return q.mainView.Handle(e, ctx)
	}

	// Check for 'q' to quit
	if ke.Key == event.KeyRune && ke.Rune == 'q' {
		q.app.Quit()

		return true
	}

	// Route all other events to main view
	return q.mainView.Handle(e, ctx)
}

// OnButton1 handles Button 1 press - toggles status message.
func (q *QuitHandler) OnButton1(ctx *tui.Ctx) {
	if q.status == "Button 1 pressed - press again to clear" {
		q.status = "Tab to navigate, Enter to press buttons, 'q' to quit"
	} else {
		q.status = "Button 1 pressed - press again to clear"
	}

	if ctx != nil {
		statusRect := tui.Rect{X: q.rect.X, Y: q.rect.Y, W: q.rect.W, H: 1}
		ctx.Invalidate(statusRect)
	}
}

// OnButton2 handles Button 2 press - toggles status message.
func (q *QuitHandler) OnButton2(ctx *tui.Ctx) {
	if q.status == "Button 2 pressed - press again to clear" {
		q.status = "Tab to navigate, Enter to press buttons, 'q' to quit"
	} else {
		q.status = "Button 2 pressed - press again to clear"
	}

	if ctx != nil {
		statusRect := tui.Rect{X: q.rect.X, Y: q.rect.Y, W: q.rect.W, H: 1}
		ctx.Invalidate(statusRect)
	}
}

func (q *QuitHandler) Focusable() bool {
	return false
}

func (q *QuitHandler) Children() []tui.View {
	return []tui.View{q.mainView}
}
