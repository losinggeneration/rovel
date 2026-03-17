package main

import (
	"fmt"
	"os"

	"github.com/losinggeneration/tui"
	"github.com/losinggeneration/tui/geom"
	"github.com/losinggeneration/tui/ui/layout"
	"github.com/losinggeneration/tui/ui/widgets"
)

func main() {
	app, err := tui.New(tui.AppOpts{})
	if err != nil {
		panic(err)
	}

	border := layout.NewBorder(widgets.NewLabel("world"))
	border.SetTitle("hello")

	root := layout.NewHStack()
	root.AddChild(layout.AlignChild(border, layout.AlignCenter, layout.AlignCenter))

	app.SetRoot(&quitWrapper{id: tui.NewID(), root: root})

	if err := app.Enable(); err != nil {
		panic(err)
	}

	defer func() {
		if err := app.Restore(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to restore terminal: %v\n", err)
		}
	}()

	if err := app.Run(); err != nil {
		panic(err)
	}
}

type quitWrapper struct {
	id   tui.ID
	root tui.View
}

func (w *quitWrapper) ID() tui.ID         { return w.id }
func (w *quitWrapper) MinSize() geom.Size { return w.root.MinSize() }
func (w *quitWrapper) Layout(r geom.Rect) { w.root.Layout(r) }
func (w *quitWrapper) Rect() geom.Rect    { return w.root.Rect() }
func (w *quitWrapper) Paint(p *tui.Painter, ctx *tui.Ctx) {
	w.root.Paint(p, ctx)
}

func (w *quitWrapper) Handle(e tui.Event, ctx *tui.Ctx) bool {
	if w.root.Handle(e, ctx) {
		return true
	}

	if _, ok := e.(tui.KeyEvent); ok {
		ctx.Quit()
		return true
	}

	return false
}

func (w *quitWrapper) Focusable() bool { return false }
