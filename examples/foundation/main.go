package main

import (
	"fmt"
	"os"

	"github.com/losinggeneration/rovel"
	"github.com/losinggeneration/rovel/event"
	"github.com/losinggeneration/rovel/geom"
	"github.com/losinggeneration/rovel/style"
	"github.com/losinggeneration/rovel/text"
	"github.com/losinggeneration/rovel/ui"
	"github.com/losinggeneration/rovel/ui/layout"
	"github.com/losinggeneration/rovel/ui/widgets"
	cellwidgets "github.com/losinggeneration/rovel/ui/widgets/cell"
)

const ActionQuit ui.Action = iota + 100

type quitKeymap struct{}

func (quitKeymap) Resolve(_ ui.KeyContext, k ui.Keystroke) (ui.Action, bool) {
	switch k.Key {
	case event.KeyEsc, event.KeyCtrlC:
		return ActionQuit, true
	default:
		return ui.ActionNone, false
	}
}

type Root struct {
	*layout.VStack
}

func (r *Root) HandleAction(act ui.Action, ctx *rovel.Ctx) bool {
	if act == ActionQuit {
		ctx.Quit()

		return true
	}

	return false
}

func main() {
	app, err := rovel.New(rovel.AppOpts{
		Theme: rovel.Theme{
			Base: style.Style{
				FG:   style.ColorDefault,
				BG:   style.ColorDefault,
				Attr: 0,
			},
			Palette: rovel.Palette{
				Focus: style.Style{
					FG:   style.ColorBlack,
					BG:   style.ColorWhite,
					Attr: 0,
				},
			},
		},
		ResolveAction: ui.NewResolver(quitKeymap{}),
	})
	if err != nil {
		panic(err)
	}

	root := buildRoot()
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

func buildRoot() *Root {
	root := layout.NewVStack()

	header := buildHeader()
	root.Add(header)

	content := buildContent()
	root.AddChild(layout.GrowChild(content, 1, 1))

	return &Root{root}
}

func buildHeader() rovel.View {
	return cellwidgets.NewCanvasOpts(cellwidgets.CanvasOpts{
		MinSize: geom.Size{W: 1, H: 2},
		Paint: func(d rovel.CellDrawer, rect geom.Rect, ctx *rovel.Ctx) {
			if rect.H < 2 {
				return
			}

			title := "P1 Foundation Demo: Text, Size Policy, Box Layout"

			fill := rovel.Style{FG: style.ColorYellow, BG: style.ColorBlue}
			d.FillRect(rect, fill)
			d.DrawText(rovel.Point{X: rect.X, Y: rect.Y}, title, rovel.Style{FG: style.ColorYellow, BG: style.ColorBlue, Attr: style.AttrBold})
			d.DrawText(rovel.Point{X: rect.X, Y: rect.Y + 1}, "Esc or Ctrl+C to quit", rovel.Style{FG: style.ColorWhite, BG: style.ColorBlue})
		},
		Handle: func(e rovel.Event, ctx *rovel.Ctx) bool {
			return false
		},
	})
}

func buildContent() rovel.View {
	hpanes := layout.NewHStack()

	left := buildTextDemo()
	leftBorder := layout.NewBorder(left)
	leftBorder.SetTitle(" Text Primitives ")
	hpanes.AddChild(layout.GrowChild(leftBorder, 1, 1))

	center := buildSizePolicyDemo()
	centerBorder := layout.NewBorder(center)
	centerBorder.SetTitle(" Size Policy ")
	hpanes.AddChild(layout.GrowChild(centerBorder, 2, 1))

	right := buildBoxLayoutDemo()
	rightBorder := layout.NewBorder(right)
	rightBorder.SetTitle(" Box Layout ")
	hpanes.AddChild(layout.GrowChild(rightBorder, 2, 1))

	return hpanes
}

func buildTextDemo() rovel.View {
	stack := layout.NewVStack()

	stack.Add(widgets.NewLabel("Width(\"hello\") = 5"))
	stack.Add(widgets.NewLabel("Width(\"日本語\") = 6"))

	stack.Add(widgets.NewLabel(""))
	stack.Add(widgets.NewLabel("FitPrefix demo:"))

	canvas := cellwidgets.NewCanvasOpts(cellwidgets.CanvasOpts{
		MinSize: geom.Size{W: 20, H: 8},
		Paint: func(d rovel.CellDrawer, rect geom.Rect, ctx *rovel.Ctx) {
			y := rect.Y

			s1 := "Hello, World!"
			end, cols, clipped := text.FitPrefix(s1, 10)
			line := fmt.Sprintf("FitPrefix(%q, 10)", s1)
			d.DrawText(rovel.Point{X: rect.X, Y: y}, line, rovel.Style{})
			y++
			d.DrawText(rovel.Point{X: rect.X, Y: y}, fmt.Sprintf("  -> end=%d, cols=%d, clipped=%v", end, cols, clipped), rovel.Style{})
			y += 2

			s2 := "日本語テスト"
			end, cols, clipped = text.FitPrefix(s2, 5)
			line = fmt.Sprintf("FitPrefix(%q, 5)", s2)
			d.DrawText(rovel.Point{X: rect.X, Y: y}, line, rovel.Style{})
			y++
			d.DrawText(rovel.Point{X: rect.X, Y: y}, fmt.Sprintf("  -> end=%d, cols=%d, clipped=%v", end, cols, clipped), rovel.Style{})
			y += 2

			d.DrawText(rovel.Point{X: rect.X, Y: y}, "Truncate(\"hello world\", 8, true):", rovel.Style{})
			y++
			truncated := text.Truncate("hello world", 8, true)
			d.DrawText(rovel.Point{X: rect.X, Y: y}, fmt.Sprintf("  -> %q", truncated), rovel.Style{})
		},
		Handle: func(e rovel.Event, ctx *rovel.Ctx) bool {
			return false
		},
	})
	stack.Add(canvas)

	stack.Add(widgets.NewLabel(""))
	stack.Add(widgets.NewLabel("Wrap demo:"))

	wrapCanvas := cellwidgets.NewCanvasOpts(cellwidgets.CanvasOpts{
		MinSize: geom.Size{W: 20, H: 6},
		Paint: func(d rovel.CellDrawer, rect geom.Rect, ctx *rovel.Ctx) {
			y := rect.Y
			s := "The quick brown fox jumps"
			lines := text.Wrap(s, 10)
			d.DrawText(rovel.Point{X: rect.X, Y: y}, fmt.Sprintf("Wrap(%q, 10):", s), rovel.Style{})

			y++
			for i, line := range lines {
				d.DrawText(rovel.Point{X: rect.X, Y: y}, fmt.Sprintf("  L%d: [%d,%d) w=%d", i, line.Start, line.End, line.Width), rovel.Style{})
				y++
			}
		},
		Handle: func(e rovel.Event, ctx *rovel.Ctx) bool {
			return false
		},
	})
	stack.Add(wrapCanvas)

	return stack
}

type sizedLabel struct {
	*widgets.Label

	prefSize geom.Size
}

func (l *sizedLabel) PreferredSize() geom.Size {
	return l.prefSize
}

func newSizedLabel(text string, prefW, prefH int) *sizedLabel {
	return &sizedLabel{
		Label:    widgets.NewLabel(text),
		prefSize: geom.Size{W: prefW, H: prefH},
	}
}

func buildSizePolicyDemo() rovel.View {
	stack := layout.NewVStack()

	stack.Add(widgets.NewLabel("GrowX vs fixed width:"))
	stack.Add(widgets.NewLabel(""))

	h1 := layout.NewHStack()
	h1.AddChild(layout.AlignChild(newSizedLabel("[Fixed]", 8, 1), layout.AlignCenter, layout.AlignCenter))
	h1.AddChild(layout.GrowXChild(newSizedLabel("[Grows]", 8, 1), 1))
	h1.AddChild(layout.AlignChild(newSizedLabel("[Fixed]", 8, 1), layout.AlignCenter, layout.AlignCenter))
	stack.Add(h1)

	stack.Add(widgets.NewLabel(""))
	stack.Add(widgets.NewLabel("Stretch factors (1:2:3):"))
	stack.Add(widgets.NewLabel(""))

	h2 := layout.NewHStack()
	h2.AddChild(layout.GrowXChild(widgets.NewLabel("[1]"), 1))
	h2.AddChild(layout.GrowXChild(widgets.NewLabel("[2]"), 2))
	h2.AddChild(layout.GrowXChild(widgets.NewLabel("[3]"), 3))
	stack.Add(h2)

	stack.Add(widgets.NewLabel(""))
	stack.Add(widgets.NewLabel("Cross-axis alignment:"))
	stack.Add(widgets.NewLabel(""))

	h3 := layout.NewHStack()
	tall1 := cellwidgets.NewCanvasOpts(cellwidgets.CanvasOpts{
		MinSize: geom.Size{W: 8, H: 3},
		Paint: func(d rovel.CellDrawer, rect geom.Rect, ctx *rovel.Ctx) {
			d.FillRect(rect, rovel.Style{BG: style.ColorGreen})
			d.DrawText(rovel.Point{X: rect.X, Y: rect.Y}, "Start", rovel.Style{BG: style.ColorGreen})
		},
		Handle: func(e rovel.Event, ctx *rovel.Ctx) bool { return false },
	})
	tall2 := cellwidgets.NewCanvasOpts(cellwidgets.CanvasOpts{
		MinSize: geom.Size{W: 8, H: 3},
		Paint: func(d rovel.CellDrawer, rect geom.Rect, ctx *rovel.Ctx) {
			d.FillRect(rect, rovel.Style{BG: style.ColorYellow})
			d.DrawText(rovel.Point{X: rect.X, Y: rect.Y + 1}, "Center", rovel.Style{BG: style.ColorYellow})
		},
		Handle: func(e rovel.Event, ctx *rovel.Ctx) bool { return false },
	})
	tall3 := cellwidgets.NewCanvasOpts(cellwidgets.CanvasOpts{
		MinSize: geom.Size{W: 8, H: 3},
		Paint: func(d rovel.CellDrawer, rect geom.Rect, ctx *rovel.Ctx) {
			d.FillRect(rect, rovel.Style{BG: style.ColorRed})
			d.DrawText(rovel.Point{X: rect.X, Y: rect.Y + 2}, "End", rovel.Style{BG: style.ColorRed})
		},
		Handle: func(e rovel.Event, ctx *rovel.Ctx) bool { return false },
	})

	h3.AddChild(layout.Child{View: tall1, Opts: layout.SizePolicy{AlignY: layout.AlignStart}})
	h3.AddChild(layout.Child{View: tall2, Opts: layout.SizePolicy{AlignY: layout.AlignCenter}})
	h3.AddChild(layout.Child{View: tall3, Opts: layout.SizePolicy{AlignY: layout.AlignEnd}})
	stack.Add(h3)

	return stack
}

func buildBoxLayoutDemo() rovel.View {
	stack := layout.NewVStack()

	stack.Add(widgets.NewLabel("Gap=1 between items:"))
	stack.Add(widgets.NewLabel(""))

	h1 := layout.NewHStack()
	h1.SetGap(1)
	h1.Add(widgets.NewLabel("[A]"))
	h1.Add(widgets.NewLabel("[B]"))
	h1.Add(widgets.NewLabel("[C]"))
	h1.Add(widgets.NewLabel("[D]"))
	stack.Add(h1)

	stack.Add(widgets.NewLabel(""))
	stack.Add(widgets.NewLabel("Gap=2 between items:"))
	stack.Add(widgets.NewLabel(""))

	h2 := layout.NewHStack()
	h2.SetGap(2)
	h2.Add(widgets.NewLabel("[1]"))
	h2.Add(widgets.NewLabel("[2]"))
	h2.Add(widgets.NewLabel("[3]"))
	stack.Add(h2)

	stack.Add(widgets.NewLabel(""))
	stack.Add(widgets.NewLabel("VStack with gap=1:"))
	stack.Add(widgets.NewLabel(""))

	v1 := layout.NewVStack()
	v1.SetGap(1)
	v1.Add(widgets.NewLabel("  Item A"))
	v1.Add(widgets.NewLabel("  Item B"))
	v1.Add(widgets.NewLabel("  Item C"))

	vBorder := layout.NewBorder(v1)
	vBorder.SetTitle(" v ")
	stack.Add(vBorder)

	stack.Add(widgets.NewLabel(""))
	stack.Add(widgets.NewLabel("Stretch in VStack:"))
	stack.Add(widgets.NewLabel(""))

	v2 := layout.NewVStack()
	v2.SetGap(1)
	v2.AddChild(layout.Child{View: widgets.NewLabel("Fixed (min-size)")})
	v2.AddChild(layout.GrowYChild(widgets.NewLabel("Grows (stretch=1)"), 1))
	v2.AddChild(layout.Child{View: widgets.NewLabel("Fixed (min-size)")})

	v2Border := layout.NewBorder(v2)
	v2Border.SetTitle(" stretch ")
	stack.AddChild(layout.GrowChild(v2Border, 1, 1))

	return stack
}
