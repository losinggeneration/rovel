package main

import (
	"fmt"

	"github.com/losinggeneration/rovel"
	"github.com/losinggeneration/rovel/event"
	"github.com/losinggeneration/rovel/ui"
	"github.com/losinggeneration/rovel/ui/layout"
	"github.com/losinggeneration/rovel/ui/widgets"
)

const ActionQuit ui.Action = iota + 100

type quitKeymap struct{}

func (quitKeymap) Resolve(_ ui.KeyContext, k ui.Keystroke) (ui.Action, bool) {
	if k.Key == event.KeyEsc {
		return ActionQuit, true
	}

	return ui.ActionNone, false
}

func main() {
	app, err := rovel.New(rovel.AppOpts{ResolveAction: ui.NewResolver(quitKeymap{})})
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

	// Create a wrapper that handles quit and button status
	quitHandler := &QuitHandler{
		app:      app,
		mainView: vstack,
		id:       rovel.NewID(),
		status:   "Tab to navigate, Enter to press buttons, 'q' to quit",
	}

	// Wire up button callbacks to update wrapper state
	button1.SetOnPress(quitHandler.OnButton(1))
	button2.SetOnPress(quitHandler.OnButton(2))

	// Set root and run
	app.SetRoot(quitHandler)

	if err := app.Enable(); err != nil {
		fmt.Printf("Failed to enable app: %v\n", err)

		return
	}

	defer func() {
		err := app.Restore()
		if err != nil {
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
	app      *rovel.App
	mainView rovel.View
	id       rovel.ID
	rect     rovel.Rect
	status   string
}

func (q *QuitHandler) ID() rovel.ID {
	return q.id
}

func (q *QuitHandler) Rect() rovel.Rect {
	return q.rect
}

func (q *QuitHandler) Layout(r rovel.Rect) {
	q.rect = r
	// Reserve top row for status, rest for main view
	mainRect := rovel.Rect{
		X: r.X,
		Y: r.Y + 1,
		W: r.W,
		H: r.H - 1,
	}
	q.mainView.Layout(mainRect)
}

func (q *QuitHandler) MinSize() rovel.Size {
	mainMin := q.mainView.MinSize()

	return rovel.Size{W: mainMin.W, H: mainMin.H + 1}
}

func (q *QuitHandler) Paint(d rovel.Drawer, ctx *rovel.Ctx) {
	// Paint status row
	d.DrawText(rovel.Point{X: q.rect.X, Y: q.rect.Y}, q.status, ctx.Theme.Base)

	// Paint main view below status
	mainRect := rovel.Rect{
		X: q.rect.X,
		Y: q.rect.Y + 1,
		W: q.rect.W,
		H: q.rect.H - 1,
	}
	d.WithClip(mainRect, func(d rovel.Drawer) {
		q.mainView.Paint(d, ctx)
	})
}

func (q *QuitHandler) HandleAction(act ui.Action, ctx *rovel.Ctx) bool {
	if act == ActionQuit {
		q.app.Quit()

		return true
	}

	return false
}

func (q *QuitHandler) Handle(e rovel.Event, ctx *rovel.Ctx) bool {
	return q.mainView.Handle(e, ctx)
}

func (q *QuitHandler) OnButton(btn int) func(ctx *rovel.Ctx) {
	return func(ctx *rovel.Ctx) {
		msg := fmt.Sprintf("Button %d pressed - press again to clear", btn)
		if q.status == msg {
			q.status = "Tab to navigate, Enter to press buttons, Esc to quit"
		} else {
			q.status = msg
		}

		if ctx != nil {
			statusRect := rovel.Rect{X: q.rect.X, Y: q.rect.Y, W: q.rect.W, H: 1}
			ctx.Invalidate(statusRect)
		}
	}
}

func (q *QuitHandler) Focusable() bool {
	return false
}

func (q *QuitHandler) Children() []rovel.View {
	return []rovel.View{q.mainView}
}
