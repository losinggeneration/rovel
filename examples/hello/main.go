package main

import (
	"fmt"
	"os"

	"github.com/losinggeneration/tui"
	"github.com/losinggeneration/tui/ui"
	"github.com/losinggeneration/tui/ui/layout"
	"github.com/losinggeneration/tui/ui/widgets"
)

const ActionQuit ui.Action = iota + 100

var appOpts = tui.AppOpts{
	ResolveAction: ui.NewResolver(quitKeymap{}),
}

type quitKeymap struct{}

func (quitKeymap) Resolve(_ ui.KeyContext, _ ui.Keystroke) (ui.Action, bool) {
	return ActionQuit, true
}

type Root struct {
	*layout.HStack
}

func (r *Root) HandleAction(act int, ctx *tui.Ctx) bool {
	if ui.Action(act) == ActionQuit {
		ctx.Quit()

		return true
	}

	return false
}

func main() {
	app, err := tui.New(appOpts)
	if err != nil {
		panic(err)
	}

	border := layout.NewBorder(widgets.NewLabel("world"))
	border.SetTitle("hello")

	root := layout.NewHStack()
	root.AddChild(layout.AlignChild(border, layout.AlignCenter, layout.AlignCenter))

	app.SetRoot(&Root{root})

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
