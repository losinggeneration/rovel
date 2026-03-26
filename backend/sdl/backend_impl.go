package sdl

import (
	"errors"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/losinggeneration/tui/backend"
	"github.com/losinggeneration/tui/backend/cellsurface"
	tevent "github.com/losinggeneration/tui/event"
	"github.com/losinggeneration/tui/geom"
	"github.com/losinggeneration/tui/render"
	"github.com/losinggeneration/tui/style"
	gsdl "github.com/veandco/go-sdl2/sdl"
	gttf "github.com/veandco/go-sdl2/ttf"
)

// Backend is the concrete SDL backend.
type Backend struct {
	*Core

	window   *gsdl.Window
	renderer *gsdl.Renderer
	font     *gttf.Font

	renderMu sync.Mutex

	running  atomic.Bool
	pollDone chan struct{}
}

var (
	_ backend.Backend             = (*Backend)(nil)
	_ backend.CapabilityReporter  = (*Backend)(nil)
	_ backend.InputFeatureEnabler = (*Backend)(nil)
	_ backend.CellFrameSink       = (*Backend)(nil)
	_ backend.ClipboardBackend    = (*Backend)(nil)
)

func newBackend(opts Options) (*Backend, error) {
	return &Backend{
		Core: NewCore(opts),
	}, nil
}

// Enable enables SDL, opens the window and font, and starts event polling.
func (b *Backend) Enable() (geom.Size, error) {
	size, err := b.Core.Enable()
	if err != nil {
		return geom.Size{}, err
	}

	if err := gsdl.Init(gsdl.INIT_VIDEO); err != nil {
		return geom.Size{}, err
	}

	if err := gttf.Init(); err != nil {
		gsdl.Quit()
		return geom.Size{}, err
	}

	opts := b.Options()
	fontPath, err := discoverFontPath(opts.FontPath)
	if err != nil {
		gttf.Quit()
		gsdl.Quit()
		return geom.Size{}, err
	}

	window, err := gsdl.CreateWindow(
		opts.Title,
		gsdl.WINDOWPOS_UNDEFINED,
		gsdl.WINDOWPOS_UNDEFINED,
		int32(opts.WindowWidth),
		int32(opts.WindowHeight),
		gsdl.WINDOW_SHOWN|gsdl.WINDOW_RESIZABLE,
	)
	if err != nil {
		gttf.Quit()
		gsdl.Quit()
		return geom.Size{}, err
	}

	renderer, err := createRenderer(window)
	if err != nil {
		_ = window.Destroy()
		gttf.Quit()
		gsdl.Quit()
		return geom.Size{}, err
	}

	font, err := gttf.OpenFont(fontPath, opts.FontSize)
	if err != nil {
		_ = renderer.Destroy()
		_ = window.Destroy()
		gttf.Quit()
		gsdl.Quit()
		return geom.Size{}, err
	}

	b.window = window
	b.renderer = renderer
	b.font = font

	cellW, cellH := measureFontMetrics(font, opts.CellWidth, opts.CellHeight)
	b.mu.Lock()
	b.metrics = cellsurface.Metrics{CellWidth: cellW, CellHeight: cellH}
	b.size = geom.Size{
		W: max(1, opts.WindowWidth/cellW),
		H: max(1, opts.WindowHeight/cellH),
	}
	b.mu.Unlock()

	gsdl.StartTextInput()

	b.running.Store(true)
	b.pollDone = make(chan struct{})
	go b.pollEvents()

	return size, nil
}

func (b *Backend) Restore() error {
	b.running.Store(false)
	if b.pollDone != nil {
		<-b.pollDone
	}

	gsdl.StopTextInput()

	b.renderMu.Lock()
	defer b.renderMu.Unlock()

	if b.font != nil {
		b.font.Close()
		b.font = nil
	}

	if b.renderer != nil {
		_ = b.renderer.Destroy()
		b.renderer = nil
	}

	if b.window != nil {
		_ = b.window.Destroy()
		b.window = nil
	}

	gttf.Quit()
	gsdl.Quit()
	b.Core.Close()

	return b.Core.Restore()
}

func (b *Backend) PresentCellFrame(frame backend.CellFrame) error {
	if err := b.Core.PresentCellFrame(frame); err != nil {
		return err
	}

	return b.drawFrame(frame)
}

func (b *Backend) ClipboardWrite(text string) error {
	return gsdl.SetClipboardText(text)
}

func (b *Backend) ClipboardRead() (string, error) {
	return gsdl.GetClipboardText()
}

func (b *Backend) drawFrame(frame backend.CellFrame) error {
	b.renderMu.Lock()
	defer b.renderMu.Unlock()

	if b.renderer == nil || b.font == nil {
		return nil
	}

	opts := b.Options()
	metrics := b.Metrics()

	bg := opts.DefaultBG
	if err := b.renderer.SetDrawColor(bg.R, bg.G, bg.B, bg.A); err != nil {
		return err
	}
	if err := b.renderer.Clear(); err != nil {
		return err
	}

	base := style.Style{}
	for y := 0; y < frame.H; y++ {
		runs := cellsurface.ResolveGlyphRuns(frame, y, base, opts.DefaultFG, opts.DefaultBG)
		for _, run := range runs {
			dst := metrics.CellRect(run.X, y)
			dst.W = cellRunWidth(run.Text, metrics.CellWidth)

			if err := b.renderer.SetDrawColor(run.BG.R, run.BG.G, run.BG.B, run.BG.A); err != nil {
				return err
			}
			if err := b.renderer.FillRect(&gsdl.Rect{
				X: int32(dst.X),
				Y: int32(dst.Y),
				W: int32(dst.W),
				H: int32(dst.H),
			}); err != nil {
				return err
			}

			if strings.TrimSpace(run.Text) == "" {
				continue
			}

			b.font.SetStyle(ttfStyle(run.Attr))

			if shouldRenderPerCell(run.Text) {
				if err := b.drawPerCellRun(run, metrics, y); err != nil {
					return err
				}
				continue
			}

			surface, err := b.font.RenderUTF8Blended(run.Text, rgbaToSDL(run.FG))
			if err != nil {
				return err
			}

			w := surface.W
			h := surface.H

			texture, err := b.renderer.CreateTextureFromSurface(surface)
			surface.Free()
			if err != nil {
				return err
			}

			copyErr := b.renderer.Copy(texture, nil, &gsdl.Rect{
				X: int32(dst.X),
				Y: int32(dst.Y),
				W: w,
				H: h,
			})
			_ = texture.Destroy()
			if copyErr != nil {
				return copyErr
			}
		}
	}

	b.renderer.Present()

	return nil
}

func (b *Backend) drawPerCellRun(run cellsurface.ResolvedGlyphRun, metrics cellsurface.Metrics, y int) error {
	x := run.X
	for _, r := range run.Text {
		span := max(1, render.RuneWidth(r))
		cellRect := metrics.CellRect(x, y)
		cellRect.W = span * metrics.CellWidth

		if drawBoxRune(b.renderer, r, cellRect, run.FG) {
			x += span
			continue
		}

		surface, err := b.font.RenderGlyphBlended(r, rgbaToSDL(run.FG))
		if err != nil {
			return err
		}

		texture, err := b.renderer.CreateTextureFromSurface(surface)
		if err != nil {
			surface.Free()
			return err
		}

		dst := alignedGlyphRect(b.font, r, cellRect, surface.W, surface.H)
		surface.Free()

		copyErr := b.renderer.Copy(texture, nil, &gsdl.Rect{
			X: int32(dst.X),
			Y: int32(dst.Y),
			W: int32(dst.W),
			H: int32(dst.H),
		})
		_ = texture.Destroy()
		if copyErr != nil {
			return copyErr
		}

		x += span
	}

	return nil
}

func (b *Backend) pollEvents() {
	defer close(b.pollDone)

	for b.running.Load() {
		ev := gsdl.PollEvent()
		if ev == nil {
			time.Sleep(8 * time.Millisecond)
			continue
		}

		switch e := ev.(type) {
		case gsdl.QuitEvent:
			b.running.Store(false)
			b.Core.Close()
			return
		case gsdl.WindowEvent:
			if e.Event == gsdl.WINDOWEVENT_SIZE_CHANGED || e.Event == gsdl.WINDOWEVENT_RESIZED {
				b.ResizeWindow(int(e.Data1), int(e.Data2))
			}
		case gsdl.TextInputEvent:
			for _, r := range e.Text {
				b.SendEvent(tevent.KeyEvent{Key: tevent.KeyRune, Rune: r})
			}
		case gsdl.KeyboardEvent:
			if e.Type != gsdl.KEYDOWN {
				continue
			}

			if key, ok := mapKey(e.Keysym.Sym, e.Keysym.Mod); ok {
				b.SendEvent(tevent.KeyEvent{Key: key, Mod: mapMod(e.Keysym.Mod)})
			}
		case gsdl.MouseMotionEvent:
			b.SendEvent(b.MapMouse(int(e.X), int(e.Y), tevent.MouseButtonNone, tevent.MouseMove, 0))
		case gsdl.MouseButtonEvent:
			action := tevent.MouseRelease
			if e.State == gsdl.PRESSED {
				action = tevent.MousePress
			}
			b.SendEvent(b.MapMouse(int(e.X), int(e.Y), mapMouseButton(e.Button), action, 0))
		case gsdl.MouseWheelEvent:
			x, y, _ := gsdl.GetMouseState()
			button := tevent.MouseButtonNone
			if e.Y > 0 {
				button = tevent.MouseButtonWheelUp
			} else if e.Y < 0 {
				button = tevent.MouseButtonWheelDown
			}
			if button != tevent.MouseButtonNone {
				b.SendEvent(b.MapMouse(int(x), int(y), button, tevent.MousePress, 0))
			}
		}
	}
}

func mapKey(sym gsdl.Keycode, mod gsdl.Keymod) (tevent.Key, bool) {
	switch sym {
	case gsdl.K_RETURN, gsdl.K_RETURN2:
		return tevent.KeyEnter, true
	case gsdl.K_ESCAPE:
		return tevent.KeyEsc, true
	case gsdl.K_TAB:
		return tevent.KeyTab, true
	case gsdl.K_BACKSPACE:
		return tevent.KeyBackspace, true
	case gsdl.K_UP:
		return tevent.KeyUp, true
	case gsdl.K_DOWN:
		return tevent.KeyDown, true
	case gsdl.K_LEFT:
		return tevent.KeyLeft, true
	case gsdl.K_RIGHT:
		return tevent.KeyRight, true
	case gsdl.K_HOME:
		return tevent.KeyHome, true
	case gsdl.K_END:
		return tevent.KeyEnd, true
	case gsdl.K_INSERT:
		return tevent.KeyInsert, true
	case gsdl.K_DELETE:
		return tevent.KeyDelete, true
	case gsdl.K_PAGEUP:
		return tevent.KeyPageUp, true
	case gsdl.K_PAGEDOWN:
		return tevent.KeyPageDown, true
	case gsdl.K_F1:
		return tevent.KeyF1, true
	case gsdl.K_F2:
		return tevent.KeyF2, true
	case gsdl.K_F3:
		return tevent.KeyF3, true
	case gsdl.K_F4:
		return tevent.KeyF4, true
	case gsdl.K_F5:
		return tevent.KeyF5, true
	case gsdl.K_F6:
		return tevent.KeyF6, true
	case gsdl.K_F7:
		return tevent.KeyF7, true
	case gsdl.K_F8:
		return tevent.KeyF8, true
	case gsdl.K_F9:
		return tevent.KeyF9, true
	case gsdl.K_F10:
		return tevent.KeyF10, true
	case gsdl.K_F11:
		return tevent.KeyF11, true
	case gsdl.K_F12:
		return tevent.KeyF12, true
	case gsdl.K_c:
		if mod&gsdl.KMOD_CTRL != 0 {
			return tevent.KeyCtrlC, true
		}
	}

	return tevent.KeyNone, false
}

func mapMod(mod gsdl.Keymod) tevent.ModMask {
	var out tevent.ModMask
	if mod&gsdl.KMOD_SHIFT != 0 {
		out |= tevent.ModShift
	}
	if mod&gsdl.KMOD_ALT != 0 {
		out |= tevent.ModAlt
	}
	if mod&gsdl.KMOD_CTRL != 0 {
		out |= tevent.ModCtrl
	}

	return out
}

func mapMouseButton(btn gsdl.Button) tevent.MouseButton {
	switch btn {
	case gsdl.ButtonLeft:
		return tevent.MouseButtonLeft
	case gsdl.ButtonMiddle:
		return tevent.MouseButtonMiddle
	case gsdl.ButtonRight:
		return tevent.MouseButtonRight
	default:
		return tevent.MouseButtonNone
	}
}

func rgbaToSDL(c style.RGBA) gsdl.Color {
	return gsdl.Color{R: c.R, G: c.G, B: c.B, A: c.A}
}

func alignedGlyphRect(font *gttf.Font, r rune, cellRect cellsurface.PixelRect, glyphW, glyphH int32) cellsurface.PixelRect {
	dst := cellRect
	dst.W = int(glyphW)
	dst.H = int(glyphH)
	dst.X += max(0, (cellRect.W-int(glyphW))/2)
	dst.Y += max(0, (cellRect.H-int(glyphH))/2)

	if gm, err := font.GlyphMetrics(r); err == nil {
		advancePad := max(0, (cellRect.W-gm.Advance)/2)
		dst.X = cellRect.X + advancePad + gm.MinX
	}

	return dst
}

func drawBoxRune(renderer *gsdl.Renderer, r rune, cellRect cellsurface.PixelRect, fg style.RGBA) bool {
	if !isBoxDrawingRune(r) {
		return false
	}

	if err := renderer.SetDrawColor(fg.R, fg.G, fg.B, fg.A); err != nil {
		return false
	}

	cx := cellRect.X + cellRect.W/2
	cy := cellRect.Y + cellRect.H/2
	thickness := max(1, min(cellRect.W, cellRect.H)/8)
	left := cellRect.X
	right := cellRect.X + cellRect.W
	top := cellRect.Y
	bottom := cellRect.Y + cellRect.H

	vTop := func() error {
		return fillSDLRect(renderer, cx-thickness/2, top, thickness, max(1, cy-top+thickness/2))
	}
	vBottom := func() error {
		return fillSDLRect(renderer, cx-thickness/2, cy-thickness/2, thickness, max(1, bottom-(cy-thickness/2)))
	}
	hLeft := func() error {
		return fillSDLRect(renderer, left, cy-thickness/2, max(1, cx-left+thickness/2), thickness)
	}
	hRight := func() error {
		return fillSDLRect(renderer, cx-thickness/2, cy-thickness/2, max(1, right-(cx-thickness/2)), thickness)
	}

	var err error
	switch r {
	case '│':
		err = joinErr(vTop(), vBottom())
	case '─':
		err = joinErr(hLeft(), hRight())
	case '┌', '╭':
		err = joinErr(vBottom(), hRight())
	case '┐', '╮':
		err = joinErr(vBottom(), hLeft())
	case '└', '╰':
		err = joinErr(vTop(), hRight())
	case '┘', '╯':
		err = joinErr(vTop(), hLeft())
	case '├':
		err = joinErr(vTop(), vBottom(), hRight())
	case '┤':
		err = joinErr(vTop(), vBottom(), hLeft())
	case '┬':
		err = joinErr(hLeft(), hRight(), vBottom())
	case '┴':
		err = joinErr(hLeft(), hRight(), vTop())
	case '┼':
		err = joinErr(hLeft(), hRight(), vTop(), vBottom())
	default:
		return false
	}

	return err == nil
}

func fillSDLRect(renderer *gsdl.Renderer, x, y, w, h int) error {
	return renderer.FillRect(&gsdl.Rect{
		X: int32(x),
		Y: int32(y),
		W: int32(max(1, w)),
		H: int32(max(1, h)),
	})
}

func joinErr(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}

	return nil
}

func ttfStyle(attr style.AttrMask) gttf.Style {
	var s gttf.Style = gttf.STYLE_NORMAL
	if attr&style.AttrBold != 0 {
		s |= gttf.STYLE_BOLD
	}
	if attr&style.AttrItalic != 0 {
		s |= gttf.STYLE_ITALIC
	}
	if attr&style.AttrUnderline != 0 {
		s |= gttf.STYLE_UNDERLINE
	}

	return s
}

func cellRunWidth(text string, cellWidth int) int {
	w := 0
	for _, r := range text {
		w += render.RuneWidth(r)
	}

	return w * cellWidth
}

func shouldRenderPerCell(text string) bool {
	for _, r := range text {
		if isCellAlignedRune(r) {
			return true
		}
	}

	return false
}

func isCellAlignedRune(r rune) bool {
	return isBoxDrawingRune(r) || isBlockRune(r)
}

func isBoxDrawingRune(r rune) bool {
	switch {
	case r >= 0x2500 && r <= 0x257F:
		return true
	default:
		return false
	}
}

func isBlockRune(r rune) bool {
	switch {
	case r >= 0x2580 && r <= 0x259F:
		return true
	case r >= 0x25A0 && r <= 0x25FF:
		return true
	default:
		return false
	}
}

func discoverFontPath(explicit string) (string, error) {
	if explicit != "" {
		if _, err := os.Stat(explicit); err == nil {
			return explicit, nil
		} else {
			return "", err
		}
	}

	candidates := []string{
		"/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf",
		"/usr/share/fonts/TTF/DejaVuSansMono.ttf",
		"/usr/share/fonts/truetype/liberation2/LiberationMono-Regular.ttf",
		"/usr/share/fonts/liberation/LiberationMono-Regular.ttf",
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", errors.New("tui/backend/sdl: no usable monospace font found; set Options.FontPath")
}

func createRenderer(window *gsdl.Window) (*gsdl.Renderer, error) {
	flags := []gsdl.RendererFlags{
		gsdl.RENDERER_ACCELERATED | gsdl.RENDERER_PRESENTVSYNC,
		gsdl.RENDERER_ACCELERATED,
		gsdl.RENDERER_SOFTWARE,
		0,
	}

	var lastErr error
	for _, flag := range flags {
		renderer, err := gsdl.CreateRenderer(window, -1, flag)
		if err == nil {
			return renderer, nil
		}

		lastErr = err
	}

	return nil, lastErr
}

func measureFontMetrics(font *gttf.Font, fallbackW, fallbackH int) (int, int) {
	cellW := fallbackW
	cellH := fallbackH

	if w, _, err := font.SizeUTF8("M"); err == nil && w > 0 {
		cellW = w
	}

	if h := font.LineSkip(); h > 0 {
		cellH = h
	} else if h := font.Height(); h > 0 {
		cellH = h
	}

	if cellW <= 0 {
		cellW = 8
	}
	if cellH <= 0 {
		cellH = 16
	}

	return cellW, cellH
}
