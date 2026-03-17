// Package main demonstrates the tui.App event loop and rendering.
package main

import (
	"fmt"
	"os"

	"github.com/losinggeneration/tui"
	"github.com/losinggeneration/tui/geom"
)

func main() {
	opts := tui.DefaultAppOpts()

	app, err := tui.New(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create app: %v\n", err)
		os.Exit(1)
	}

	// Create a simple demo view with quit callback
	quitCh := make(chan struct{})
	demo := NewDemoView(func() {
		close(quitCh)
	})
	app.SetRoot(demo)

	if err := app.Enable(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to enable app: %v\n", err)
		os.Exit(1)
	}

	defer func() {
		if err := app.Restore(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to restore terminal: %v\n", err)
		}
	}()

	// Start app in a goroutine so we can wait for quit
	go func() {
		if err := app.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "App error: %v\n", err)
			os.Exit(1)
		}
	}()

	// Wait for quit signal
	<-quitCh
}

// DemoView is a simple view that demonstrates the rendering system.
type DemoView struct {
	id     tui.ID
	rect   geom.Rect
	text   string
	ctr    int
	ticker bool
	quit   func()
}

func NewDemoView(quit func()) *DemoView {
	return &DemoView{
		id:     tui.NewID(),
		text:   "Hello, tui.App!",
		ctr:    0,
		ticker: false,
		quit:   quit,
	}
}

func (v *DemoView) ID() tui.ID {
	return v.id
}

func (v *DemoView) MinSize() geom.Size {
	return geom.Size{W: 20, H: 10}
}

func (v *DemoView) Layout(r geom.Rect) {
	v.rect = r
}

func (v *DemoView) Rect() geom.Rect {
	return v.rect
}

func (v *DemoView) Paint(p *tui.Painter, ctx *tui.Ctx) {
	// Draw a border box
	p.Box(v.rect, tui.Style{
		FG:   tui.ColorCyan,
		BG:   tui.ColorBlack,
		Attr: tui.AttrBold,
	})

	// Draw title
	title := " Demo View "
	titleX := v.rect.X + (v.rect.W-len(title))/2
	p.Text(titleX, v.rect.Y, title, tui.Style{
		FG:   tui.ColorYellow,
		BG:   tui.ColorBlue,
		Attr: tui.AttrBold,
	})

	// Draw counter text
	counterText := fmt.Sprintf("Counter: %d", v.ctr)
	p.Text(v.rect.X+2, v.rect.Y+2, counterText, tui.Style{
		FG:   tui.ColorGreen,
		BG:   tui.ColorBlack,
		Attr: tui.AttrBold,
	})

	// Draw instructions
	p.Text(v.rect.X+2, v.rect.Y+4, "Press:", tui.Style{
		FG: tui.ColorWhite,
		BG: tui.ColorBlack,
	})
	p.Text(v.rect.X+2, v.rect.Y+5, "  space - increment counter", tui.Style{
		FG: tui.ColorBrightWhite,
		BG: tui.ColorBlack,
	})
	p.Text(v.rect.X+2, v.rect.Y+6, "  t     - toggle ticker", tui.Style{
		FG: tui.ColorBrightWhite,
		BG: tui.ColorBlack,
	})
	p.Text(v.rect.X+2, v.rect.Y+7, "  q     - quit", tui.Style{
		FG: tui.ColorBrightWhite,
		BG: tui.ColorBlack,
	})

	// Draw ticker status if enabled
	if v.ticker {
		tickerText := "[ticker ON]"
		p.Text(v.rect.X+v.rect.W-len(tickerText)-2, v.rect.Y, tickerText, tui.Style{
			FG:   tui.ColorGreen,
			BG:   tui.ColorBlack,
			Attr: tui.AttrBold,
		})
	}
}

func (v *DemoView) Handle(e tui.Event, ctx *tui.Ctx) bool {
	switch ke := e.(type) {
	case tui.KeyEvent:
		switch ke.Key {
		case tui.KeyRune:
			switch ke.Rune {
			case 'q':
				// Quit the app
				if v.quit != nil {
					v.quit()
				}

				return true

			case ' ':
				// Increment counter and invalidate to repaint
				v.ctr++
				// Invalidate just the counter area
				counterRect := geom.Rect{
					X: v.rect.X + 2,
					Y: v.rect.Y + 2,
					W: 20,
					H: 1,
				}
				ctx.Invalidate(counterRect)

			case 't':
				// Toggle ticker
				v.ticker = !v.ticker
				// Invalidate the header area to show/hide ticker status
				headerRect := geom.Rect{
					X: v.rect.X,
					Y: v.rect.Y,
					W: v.rect.W,
					H: 1,
				}
				ctx.Invalidate(headerRect)
			}

			return true
		}
	}

	return false
}
