package main

import (
	"fmt"
	"os"

	"github.com/losinggeneration/tui"
	"github.com/losinggeneration/tui/geom"
	"github.com/losinggeneration/tui/style"
	"github.com/losinggeneration/tui/text"
	"github.com/losinggeneration/tui/ui/layout"
	"github.com/losinggeneration/tui/ui/widgets"
)

func main() {
	app, err := tui.New(tui.AppOpts{
		Theme: tui.Theme{
			Base: style.Style{
				FG:   style.ColorDefault,
				BG:   style.ColorDefault,
				Attr: 0,
			},
			Focus: style.Style{
				FG:   style.ColorBlack,
				BG:   style.ColorWhite,
				Attr: 0,
			},
		},
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
		if err := app.Restore(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to restore terminal: %v\n", err)
		}
	}()

	if err := app.Run(); err != nil {
		panic(err)
	}
}

func buildRoot() tui.View {
	root := layout.NewVStack()

	header := buildHeader()
	root.Add(header)

	content := buildContent()
	root.AddChild(layout.GrowChild(content, 1, 1))

	return &quitWrapper{id: tui.NewID(), root: root}
}

func buildHeader() tui.View {
	return widgets.NewCanvasOpts(widgets.CanvasOpts{
		MinSize: geom.Size{W: 1, H: 2},
		Paint: func(p *tui.Painter, rect geom.Rect, ctx *tui.Ctx) {
			if rect.H < 2 {
				return
			}
			title := "P1 Foundation Demo: Text, Size Policy, Box Layout"
			p.Fill(rect, ' ', tui.Style{FG: style.ColorYellow, BG: style.ColorBlue})
			p.Text(rect.X, rect.Y, title, tui.Style{FG: style.ColorYellow, BG: style.ColorBlue, Attr: style.AttrBold})
			p.Text(rect.X, rect.Y+1, "Esc or Ctrl+C to quit", tui.Style{FG: style.ColorWhite, BG: style.ColorBlue})
		},
		Handle: func(e tui.Event, ctx *tui.Ctx) bool {
			return false
		},
	})
}

func buildContent() tui.View {
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

func buildTextDemo() tui.View {
	stack := layout.NewVStack()

	stack.Add(widgets.NewLabel("Width(\"hello\") = 5"))
	stack.Add(widgets.NewLabel("Width(\"日本語\") = 6"))

	stack.Add(widgets.NewLabel(""))
	stack.Add(widgets.NewLabel("FitPrefix demo:"))

	canvas := widgets.NewCanvasOpts(widgets.CanvasOpts{
		MinSize: geom.Size{W: 20, H: 8},
		Paint: func(p *tui.Painter, rect geom.Rect, ctx *tui.Ctx) {
			y := rect.Y

			s1 := "Hello, World!"
			end, cols, clipped := text.FitPrefix(s1, 10)
			line := fmt.Sprintf("FitPrefix(%q, 10)", s1)
			p.Text(rect.X, y, line, tui.Style{})
			y++
			p.Text(rect.X, y, fmt.Sprintf("  -> end=%d, cols=%d, clipped=%v", end, cols, clipped), tui.Style{})
			y += 2

			s2 := "日本語テスト"
			end, cols, clipped = text.FitPrefix(s2, 5)
			line = fmt.Sprintf("FitPrefix(%q, 5)", s2)
			p.Text(rect.X, y, line, tui.Style{})
			y++
			p.Text(rect.X, y, fmt.Sprintf("  -> end=%d, cols=%d, clipped=%v", end, cols, clipped), tui.Style{})
			y += 2

			p.Text(rect.X, y, "Truncate(\"hello world\", 8, true):", tui.Style{})
			y++
			truncated := text.Truncate("hello world", 8, true)
			p.Text(rect.X, y, fmt.Sprintf("  -> %q", truncated), tui.Style{})
		},
		Handle: func(e tui.Event, ctx *tui.Ctx) bool {
			return false
		},
	})
	stack.Add(canvas)

	stack.Add(widgets.NewLabel(""))
	stack.Add(widgets.NewLabel("Wrap demo:"))

	wrapCanvas := widgets.NewCanvasOpts(widgets.CanvasOpts{
		MinSize: geom.Size{W: 20, H: 6},
		Paint: func(p *tui.Painter, rect geom.Rect, ctx *tui.Ctx) {
			y := rect.Y
			s := "The quick brown fox jumps"
			lines := text.Wrap(s, 10)
			p.Text(rect.X, y, fmt.Sprintf("Wrap(%q, 10):", s), tui.Style{})
			y++
			for i, line := range lines {
				p.Text(rect.X, y, fmt.Sprintf("  L%d: [%d,%d) w=%d", i, line.Start, line.End, line.Width), tui.Style{})
				y++
			}
		},
		Handle: func(e tui.Event, ctx *tui.Ctx) bool {
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

func buildSizePolicyDemo() tui.View {
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
	tall1 := widgets.NewCanvasOpts(widgets.CanvasOpts{
		MinSize: geom.Size{W: 8, H: 3},
		Paint: func(p *tui.Painter, rect geom.Rect, ctx *tui.Ctx) {
			p.Fill(rect, ' ', tui.Style{BG: style.ColorGreen})
			p.Text(rect.X, rect.Y, "Start", tui.Style{BG: style.ColorGreen})
		},
		Handle: func(e tui.Event, ctx *tui.Ctx) bool { return false },
	})
	tall2 := widgets.NewCanvasOpts(widgets.CanvasOpts{
		MinSize: geom.Size{W: 8, H: 3},
		Paint: func(p *tui.Painter, rect geom.Rect, ctx *tui.Ctx) {
			p.Fill(rect, ' ', tui.Style{BG: style.ColorYellow})
			p.Text(rect.X, rect.Y+1, "Center", tui.Style{BG: style.ColorYellow})
		},
		Handle: func(e tui.Event, ctx *tui.Ctx) bool { return false },
	})
	tall3 := widgets.NewCanvasOpts(widgets.CanvasOpts{
		MinSize: geom.Size{W: 8, H: 3},
		Paint: func(p *tui.Painter, rect geom.Rect, ctx *tui.Ctx) {
			p.Fill(rect, ' ', tui.Style{BG: style.ColorRed})
			p.Text(rect.X, rect.Y+2, "End", tui.Style{BG: style.ColorRed})
		},
		Handle: func(e tui.Event, ctx *tui.Ctx) bool { return false },
	})
	h3.AddChild(layout.Child{View: tall1, Opts: layout.SizePolicy{AlignY: layout.AlignStart}})
	h3.AddChild(layout.Child{View: tall2, Opts: layout.SizePolicy{AlignY: layout.AlignCenter}})
	h3.AddChild(layout.Child{View: tall3, Opts: layout.SizePolicy{AlignY: layout.AlignEnd}})
	stack.Add(h3)

	return stack
}

func buildBoxLayoutDemo() tui.View {
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

type quitWrapper struct {
	id   tui.ID
	root tui.View
}

func (w *quitWrapper) ID() tui.ID {
	return w.id
}

func (w *quitWrapper) MinSize() geom.Size {
	return w.root.MinSize()
}

func (w *quitWrapper) Layout(r geom.Rect) {
	w.root.Layout(r)
}

func (w *quitWrapper) Rect() geom.Rect {
	return w.root.Rect()
}

func (w *quitWrapper) Paint(p *tui.Painter, ctx *tui.Ctx) {
	w.root.Paint(p, ctx)
}

func (w *quitWrapper) Handle(e tui.Event, ctx *tui.Ctx) bool {
	if w.root.Handle(e, ctx) {
		return true
	}

	if ke, ok := e.(tui.KeyEvent); ok {
		if ke.Key == tui.KeyEsc || ke.Key == tui.KeyCtrlC {
			ctx.Quit()
			return true
		}
	}
	return false
}

func (w *quitWrapper) Focusable() bool {
	return false
}

func (w *quitWrapper) Children() []tui.View {
	if c, ok := w.root.(interface{ Children() []tui.View }); ok {
		return c.Children()
	}
	return nil
}
