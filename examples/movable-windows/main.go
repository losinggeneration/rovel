// Example movable-windows demonstrates TurboVision-style floating windows:
// drag a window by its title bar, click a window to raise it above the
// others, close it with the [■] box on its frame, and type into whichever
// window is active. Everything is composed from ordinary views on top of
// three library pieces — the mutable Floating placement, the implicit mouse
// grab (a press keeps receiving drags until release), and RaiseOverlay
// (z-order change without dismiss side effects). There is no Window widget.
//
// Requires a mouse-capable terminal. Esc quits.
package main

import (
	"fmt"

	"github.com/losinggeneration/rovel"
	"github.com/losinggeneration/rovel/geom"
	"github.com/losinggeneration/rovel/style"
	"github.com/losinggeneration/rovel/ui/overlay"
	"github.com/losinggeneration/rovel/ui/widgets"
)

// The TurboVision palette is hardcoded rather than taken from ctx.Theme: the
// look is the point of this example. A real app would pull these from the
// theme so it could be restyled.
var (
	tvDesktop = style.Style{FG: style.ColorBrightBlue, BG: style.ColorBlue}
	tvWindow  = style.Style{FG: style.ColorBlack, BG: style.ColorWhite}
	tvActive  = style.Style{FG: style.ColorBlack, BG: style.ColorWhite, Attr: style.AttrBold}
	tvShadow  = style.Style{FG: style.ColorBlack, BG: style.ColorBlack}
	tvStatus  = style.Style{FG: style.ColorBlack, BG: style.ColorWhite}
	tvLabel   = style.Style{FG: style.ColorBlue, BG: style.ColorWhite}
	tvField   = style.Style{FG: style.ColorBlack, BG: style.ColorCyan}
	tvCursor  = style.Style{FG: style.ColorCyan, BG: style.ColorBlack}
)

// A window casts a drop shadow down and to the right, so the frame is inset
// from the overlay rect by that much — overlay painting clips to the overlay
// rect, so the shadow has to live inside it.
const (
	shadowW = 2
	shadowH = 1

	// closeBoxW is the width of the "[■]" close box on the top edge.
	closeBoxW = 3
)

// desktop is the background view: a muted panel with a hint line. Esc quits.
type desktop struct {
	id   rovel.ID
	rect rovel.Rect
}

func newDesktop() *desktop { return &desktop{id: rovel.NewID()} }

func (d *desktop) ID() rovel.ID        { return d.id }
func (d *desktop) MinSize() geom.Size  { return geom.Size{W: 1, H: 1} }
func (d *desktop) Rect() rovel.Rect    { return d.rect }
func (d *desktop) Layout(r rovel.Rect) { d.rect = r }

func (d *desktop) Paint(cd rovel.Drawer, ctx *rovel.Ctx) {
	// Speckled backdrop, the way the Borland IDE fills the screen behind its
	// windows. The pattern also proves shadows and moved windows repaint.
	for y := d.rect.Y; y < d.rect.Y+d.rect.H; y++ {
		cd.DrawText(geom.Point{X: d.rect.X, Y: y}, repeat('░', d.rect.W), tvDesktop)
	}

	// Status line along the bottom, in the TurboVision manner.
	status := d.rect.Y + d.rect.H - 1
	cd.FillRect(geom.Rect{X: d.rect.X, Y: status, W: d.rect.W, H: 1}, tvStatus)

	hint := " drag a title bar to move · click to raise · [■] closes · Esc quits "
	if len(hint) <= d.rect.W {
		cd.DrawText(geom.Point{X: d.rect.X + 1, Y: status}, hint, tvStatus)
	}
}

func repeat(r rune, n int) string {
	if n <= 0 {
		return ""
	}

	out := make([]rune, n)
	for i := range out {
		out[i] = r
	}

	return string(out)
}

func (d *desktop) Handle(e rovel.Event, ctx *rovel.Ctx) bool {
	if ke, ok := e.(rovel.KeyEvent); ok && ke.Key == rovel.KeyEsc {
		ctx.Quit()

		return true
	}

	return false
}

// window is one floating window: a title bar (raise on press, drag to move,
// × to close) over a text input that shows keyboard focus.
type window struct {
	id        rovel.ID
	rect      rovel.Rect
	title     string
	place     *overlay.Floating
	input     *widgets.TextInput
	clickable *widgets.Clickable
	overlayID rovel.ID // set once shown
	drag      bool
	anchor    geom.Point

	// lastActive is the focus state this window last painted, so Paint can
	// notice a change and repair the whole frame. See Paint.
	lastActive bool
}

func newWindow(title string, place *overlay.Floating) *window {
	w := &window{
		id:    rovel.NewID(),
		title: title,
		place: place,
		// The input carries explicit styles: the theme's are tuned for a
		// dark surface and would vanish on a TurboVision window body.
		input: widgets.NewTextInputOpts(widgets.TextInputOpts{
			StyleNormal:  &tvField,
			StyleFocused: &tvCursor,
		}),
	}

	// Clicking the input raises the window too (Clickable requests focus on
	// the wrapped view and forwards the event).
	w.clickable = &widgets.Clickable{
		View: w.input,
		OnClick: func(ctx *rovel.Ctx) {
			if w.overlayID != 0 {
				ctx.RaiseOverlay(w.overlayID)
			}
		},
	}

	return w
}

func (w *window) ID() rovel.ID { return w.id }

// MinSize leaves room for the frame, the shadow, and one interior row.
func (w *window) MinSize() geom.Size     { return geom.Size{W: 20, H: 5} }
func (w *window) Rect() rovel.Rect       { return w.rect }
func (w *window) Children() []rovel.View { return []rovel.View{w.clickable} }

func (w *window) Layout(r rovel.Rect) {
	w.rect = r

	f := w.frame()
	w.input.Layout(geom.Rect{X: f.X + 2, Y: f.Y + 2, W: max(f.W-4, 0), H: 1})
}

// frame is the window proper: the overlay rect less the shadow it casts.
func (w *window) frame() rovel.Rect {
	return geom.Rect{X: w.rect.X, Y: w.rect.Y, W: max(w.rect.W-shadowW, 0), H: max(w.rect.H-shadowH, 0)}
}

func (w *window) Paint(cd rovel.Drawer, ctx *rovel.Ctx) {
	f := w.frame()
	if f.W < 2 || f.H < 2 {
		return
	}

	// The active window wears a double frame, the others a single one — how
	// TurboVision shows which window has the keyboard.
	active := ctx.FocusedID == w.input.ID()

	// A focus change only damages the focused view's own rect — here, the
	// input's single row — but it changes every glyph in this frame. Ask for
	// the whole window back, or the border painted while active survives on
	// screen until something else damages those cells. The terminal only
	// receives damaged runs, so an undamaged stale cell is never corrected.
	// Same dance as widgets.FocusRing.
	if active != w.lastActive {
		w.lastActive = active

		ctx.Invalidate(w.rect)
	}

	frameStyle, glyphs := tvWindow, rovel.BoxGlyphsLight
	if active {
		frameStyle, glyphs = tvActive, rovel.BoxGlyphsDouble
	}

	// Shadow first, over the whole overlay rect: the frame then paints on
	// top, leaving the drop shadow along the right and bottom edges. Painting
	// the full rect matters — a cell inside the overlay rect that nobody
	// paints is left blank rather than showing the desktop through it.
	cd.FillRect(w.rect, tvShadow)

	cd.FillRect(f, tvWindow)
	cd.DrawBorder(f, rovel.BoxStyle{Glyphs: glyphs, Edges: rovel.BoxEdgesAll, Style: frameStyle})

	// [■] close box on the top edge, then the title centred beside it.
	if f.W >= closeBoxW+4 {
		cd.DrawText(geom.Point{X: f.X + 2, Y: f.Y}, "[■]", frameStyle)
	}

	if title := " " + w.title + " "; len(title) <= f.W-(closeBoxW+4) {
		cd.DrawText(geom.Point{X: f.X + (f.W-len(title))/2, Y: f.Y}, title, frameStyle)
	}

	// Field label, so the window reads as a window and not an empty box.
	if f.H >= 4 && f.W >= 12 {
		cd.DrawText(geom.Point{X: f.X + 2, Y: f.Y + 1}, "Type here:", tvLabel)
	}

	w.clickable.Paint(cd, ctx)
}

func (w *window) Handle(e rovel.Event, ctx *rovel.Ctx) bool {
	me, ok := e.(rovel.MouseEvent)
	if !ok {
		// Keys go to the input (it only acts when focused).
		return w.input.Handle(e, ctx)
	}

	switch me.Action {
	case rovel.MousePress:
		f := w.frame()

		// A press on the shadow is a press on whatever shows through it.
		if !f.Contains(geom.Point{X: me.X, Y: me.Y}) {
			return false
		}

		if w.overlayID != 0 {
			ctx.RaiseOverlay(w.overlayID)
		}

		// The [■] close box closes the window.
		if me.Y == f.Y && me.X >= f.X+2 && me.X < f.X+2+closeBoxW {
			ctx.DismissOverlayByID(w.overlayID)

			return true
		}

		w.drag = true
		w.anchor = geom.Point{X: me.X, Y: me.Y}
		ctx.RequestFocus(w.input.ID())

		return true
	case rovel.MouseDrag:
		if !w.drag {
			return false
		}

		w.place.MoveBy(me.X-w.anchor.X, me.Y-w.anchor.Y)
		w.anchor = geom.Point{X: me.X, Y: me.Y}
		ctx.InvalidateLayout()

		return true
	case rovel.MouseRelease:
		w.drag = false

		return true
	}

	return false
}

func main() {
	app, err := rovel.New(rovel.AppOpts{
		Input: rovel.InputOpts{Mouse: true, BracketedPaste: true, ModifiedKeys: true},
	})
	if err != nil {
		fmt.Printf("failed to create app: %v\n", err)

		return
	}

	app.SetRoot(newDesktop())

	windows := []*window{
		newWindow("Notes", &overlay.Floating{Rect: geom.Rect{X: 2, Y: 1, W: 30, H: 9}}),
		newWindow("Chat", &overlay.Floating{Rect: geom.Rect{X: 30, Y: 5, W: 34, H: 10}}),
		newWindow("Search", &overlay.Floating{Rect: geom.Rect{X: 12, Y: 8, W: 28, H: 6}}),
	}

	for _, w := range windows {
		_ = app.Post(func(ctx *rovel.UpdateCtx) {
			o := ctx.ShowOverlay(rovel.OverlayOpts{Root: w, Place: w.place})
			w.overlayID = o.ID()
		})
	}

	// Focus the top window so it is usable immediately.
	_ = app.Post(func(ctx *rovel.UpdateCtx) {
		ctx.RequestFocus(windows[len(windows)-1].input.ID())
	})

	defer func() { _ = app.Restore() }()

	if err := app.Enable(); err != nil {
		fmt.Printf("failed to enable app: %v\n", err)

		return
	}

	if err := app.Run(); err != nil {
		fmt.Printf("app exited with error: %v\n", err)
	}
}
