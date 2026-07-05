package main

import (
	"fmt"

	"github.com/losinggeneration/tui"
	"github.com/losinggeneration/tui/event"
	"github.com/losinggeneration/tui/ui"
	"github.com/losinggeneration/tui/ui/layout"
	"github.com/losinggeneration/tui/ui/widgets"
)

const ActionQuit ui.Action = iota + 100

type quitKeymap struct{}

func (quitKeymap) Resolve(_ ui.KeyContext, k ui.Keystroke) (ui.Action, bool) {
	if k.Key == event.KeyRune && k.Rune == 'q' {
		return ActionQuit, true
	}

	return ui.ActionNone, false
}

type Root struct {
	tui.CompositeView
}

func (r *Root) HandleAction(act ui.Action, ctx *tui.Ctx) bool {
	if act == ActionQuit {
		ctx.Quit()

		return true
	}

	return false
}

func main() {
	app, err := tui.New(tui.AppOpts{ResolveAction: ui.NewResolver(quitKeymap{})})
	if err != nil {
		fmt.Printf("Failed to create app: %v\n", err)

		return
	}

	content := &ContentView{
		id:    tui.NewID(),
		lines: generateLines(500),
	}

	scrollView := widgets.NewScrollView(widgets.ScrollViewOpts{
		Child:     content,
		Focusable: true,
	})

	border := layout.NewBorder(scrollView)
	border.SetTitle("Scrollable Content")

	status := &StatusView{
		text: "Use arrow keys, PageUp/Down, Home/End to scroll. Press 'q' to quit.",
	}

	// Use a vertical split to constrain scroll view to top portion of screen
	split := layout.NewSplit(layout.Vertical)
	split.SetFirst(border)
	split.SetSecond(status)
	split.SetRatio(0.8) // 80% for scroll view, 20% for status

	app.SetRoot(&Root{split})

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

	if err := app.Run(); err != nil {
		fmt.Printf("App error: %v\n", err)
	}
}

func generateLines(n int) []string {
	lines := make([]string, n)
	for i := range n {
		lines[i] = fmt.Sprintf("Line %d: This is some sample content that demonstrates scrolling.", i+1)
	}

	return lines
}

type ContentView struct {
	id    tui.ID
	rect  tui.Rect
	lines []string
}

func (c *ContentView) ID() tui.ID        { return c.id }
func (c *ContentView) Rect() tui.Rect    { return c.rect }
func (c *ContentView) MinSize() tui.Size { return tui.Size{W: 40, H: len(c.lines)} }
func (c *ContentView) Focusable() bool   { return true }
func (c *ContentView) Layout(r tui.Rect) { c.rect = r }

func (c *ContentView) Paint(d tui.Drawer, ctx *tui.Ctx) {
	for i, line := range c.lines {
		d.DrawText(tui.Point{X: c.rect.X, Y: c.rect.Y + i}, line, ctx.Theme.Base)
	}
}

func (c *ContentView) Handle(e tui.Event, ctx *tui.Ctx) bool {
	return false
}

type StatusView struct {
	rect tui.Rect
	text string
}

func (s *StatusView) ID() tui.ID        { return 0 }
func (s *StatusView) Rect() tui.Rect    { return s.rect }
func (s *StatusView) MinSize() tui.Size { return tui.Size{W: len(s.text), H: 1} }
func (s *StatusView) Focusable() bool   { return false }

func (s *StatusView) Layout(r tui.Rect) {
	s.rect = r
}

func (s *StatusView) Paint(d tui.Drawer, ctx *tui.Ctx) {
	d.DrawText(tui.Point{X: s.rect.X, Y: s.rect.Y}, s.text, ctx.Theme.Base)
}

func (s *StatusView) Handle(e tui.Event, ctx *tui.Ctx) bool {
	return false
}
