package sdl3

import (
	"errors"
	"math"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Zyko0/go-sdl3/bin/binsdl"
	"github.com/Zyko0/go-sdl3/bin/binttf"
	"github.com/Zyko0/go-sdl3/sdl"
	"github.com/Zyko0/go-sdl3/ttf"
	"github.com/losinggeneration/rovel/backend"
	"github.com/losinggeneration/rovel/backend/cellsurface"
	tevent "github.com/losinggeneration/rovel/event"
	"github.com/losinggeneration/rovel/geom"
	"github.com/losinggeneration/rovel/render"
	"github.com/losinggeneration/rovel/style"
)

// SDL3_ttf font style flag values. The Zyko0 binding does not export
// named constants for these, but the bits are stable across SDL3_ttf releases.
const (
	ttfStyleNormal    ttf.FontStyleFlags = 0
	ttfStyleBold      ttf.FontStyleFlags = 1
	ttfStyleItalic    ttf.FontStyleFlags = 2
	ttfStyleUnderline ttf.FontStyleFlags = 4
)

var ErrNoMonospaceFont = errors.New("tui/backend/sdl3: no usable monospace font found; set Options.FontPath")

// loadLibrariesOnce loads the SDL3 and SDL3_ttf shared libraries exactly once
// per process via the bundled binaries in github.com/Zyko0/go-sdl3/bin. The
// Zyko0 binding resolves every SDL3 entry point through the library handle
// returned by these loads; if either library is not loaded before sdl.Init /
// ttf.Init is called, the binding's function pointers are nil and the call
// segfaults.
//
// Libraries are loaded once for the process lifetime (mirroring the cgo
// SDL2 backend, where libSDL2 stays linked in for the lifetime of the
// binary). sdl.Init / sdl.Quit still cycle the SDL3 subsystem refcount on
// every Enable / Restore, so multiple enable/restore cycles work normally.
var loadLibrariesOnce sync.Once

func loadLibraries() {
	loadLibrariesOnce.Do(func() {
		// binsdl.Load() and binttf.Load() each panic via log.Fatal if the
		// bundle cannot be extracted or the dlsym fails. That is the
		// upstream contract: there is no recoverable failure path on a
		// supported platform.
		_ = binsdl.Load()
		_ = binttf.Load()
	})
}

// Backend is the concrete SDL3 backend.
type Backend struct {
	*Core

	window   *sdl.Window
	renderer *sdl.Renderer
	font     *ttf.Font

	renderMu sync.Mutex

	running  atomic.Bool
	pollDone chan struct{}
}

func newBackend(opts Options) (*Backend, error) {
	return &Backend{
		Core: NewCore(opts),
	}, nil
}

// Enable enables SDL3, opens the window and font, and starts event polling.
//
// The first call loads the SDL3 and SDL3_ttf shared libraries via the bundled
// github.com/Zyko0/go-sdl3/bin/binsdl and .../bin/binttf binaries, so callers
// do not need to manage library loading themselves. Subsequent Enable calls
// reuse the already-loaded libraries and only cycle SDL3's subsystem refcount.
func (b *Backend) Enable() (geom.Size, error) {
	size, err := b.Core.Enable()
	if err != nil {
		return geom.Size{}, err
	}

	loadLibraries()

	if err := sdl.Init(sdl.INIT_VIDEO); err != nil {
		return geom.Size{}, err
	}

	if err := ttf.Init(); err != nil {
		sdl.Quit()

		return geom.Size{}, err
	}

	opts := b.Options()

	fontPath, err := discoverFontPath(opts.FontPath)
	if err != nil {
		ttf.Quit()
		sdl.Quit()

		return geom.Size{}, err
	}

	window, renderer, err := sdl.CreateWindowAndRenderer(
		opts.Title,
		opts.WindowWidth,
		opts.WindowHeight,
		sdl.WINDOW_RESIZABLE,
	)
	if err != nil {
		ttf.Quit()
		sdl.Quit()

		return geom.Size{}, err
	}

	font, err := ttf.OpenFont(fontPath, float32(opts.FontSize))
	if err != nil {
		renderer.Destroy()
		window.Destroy()

		ttf.Quit()
		sdl.Quit()

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

	if err := window.StartTextInput(); err != nil {
		renderer.Destroy()
		window.Destroy()

		ttf.Quit()
		sdl.Quit()

		return geom.Size{}, err
	}

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

	if b.window != nil {
		_ = b.window.StopTextInput()
	}

	b.renderMu.Lock()
	defer b.renderMu.Unlock()

	if b.font != nil {
		b.font.Close()
		b.font = nil
	}

	if b.renderer != nil {
		b.renderer.Destroy()
		b.renderer = nil
	}

	if b.window != nil {
		b.window.Destroy()
		b.window = nil
	}

	ttf.Quit()
	sdl.Quit()
	b.Close()

	return b.Core.Restore()
}

func (b *Backend) PresentCellFrame(frame backend.CellFrame) error {
	err := b.Core.PresentCellFrame(frame)
	if err != nil {
		return err
	}

	return b.drawFrame(frame)
}

func (b *Backend) ClipboardWrite(text string) error {
	return sdl.SetClipboardText(text)
}

func (b *Backend) ClipboardRead() (string, error) {
	return sdl.GetClipboardText()
}

func (b *Backend) ClipboardReadRequest() error {
	text, err := sdl.GetClipboardText()
	if err != nil {
		return err
	}

	b.SendEvent(tevent.ClipboardResponseEvent{Text: text})

	return nil
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
	for y := range frame.H {
		runs := cellsurface.ResolveGlyphRuns(frame, y, base, opts.DefaultFG, opts.DefaultBG)
		for _, run := range runs {
			dst := metrics.CellRect(run.X, y)
			dst.W = cellRunWidth(run.Text, metrics.CellWidth)

			if err := b.renderer.SetDrawColor(run.BG.R, run.BG.G, run.BG.B, run.BG.A); err != nil {
				return err
			}

			if err := b.renderer.RenderFillRect(&sdl.FRect{
				X: float32(dst.X),
				Y: float32(dst.Y),
				W: float32(dst.W),
				H: float32(dst.H),
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

			surface, err := b.font.RenderTextBlended(run.Text, sdlColor(run.FG))
			if err != nil {
				return err
			}

			w := surface.W
			h := surface.H

			texture, err := b.renderer.CreateTextureFromSurface(surface)
			// surface is no longer needed; the texture owns its pixel copy.
			if err != nil {
				return err
			}

			copyErr := b.renderer.RenderTexture(texture, nil, &sdl.FRect{
				X: float32(dst.X),
				Y: float32(dst.Y),
				W: float32(w),
				H: float32(h),
			})
			texture.Destroy()

			if copyErr != nil {
				return copyErr
			}
		}
	}

	return b.renderer.Present()
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

		surface, err := b.font.RenderGlyphBlended(r, sdlColor(run.FG))
		if err != nil {
			return err
		}

		texture, err := b.renderer.CreateTextureFromSurface(surface)
		if err != nil {
			return err
		}

		dst := alignedGlyphRect(b.font, r, cellRect, surface.W, surface.H)
		copyErr := b.renderer.RenderTexture(texture, nil, &sdl.FRect{
			X: float32(dst.X),
			Y: float32(dst.Y),
			W: float32(dst.W),
			H: float32(dst.H),
		})
		texture.Destroy()

		if copyErr != nil {
			return copyErr
		}

		x += span
	}

	return nil
}

func (b *Backend) pollEvents() {
	defer func() {
		if r := recover(); r != nil {
			b.running.Store(false)
			b.Close()
		}

		close(b.pollDone)
	}()

	var event sdl.Event

	for b.running.Load() {
		for sdl.PollEvent(&event) {
			b.handleEvent(&event)
		}

		time.Sleep(8 * time.Millisecond)
	}
}

func (b *Backend) handleEvent(event *sdl.Event) {
	switch event.Type {
	case sdl.EVENT_QUIT:
		b.running.Store(false)
		b.Close()
	case sdl.EVENT_WINDOW_RESIZED, sdl.EVENT_WINDOW_PIXEL_SIZE_CHANGED:
		we := event.WindowEvent()
		if we == nil {
			return
		}

		b.ResizeWindow(int(we.Data1), int(we.Data2))
	case sdl.EVENT_WINDOW_SHOWN,
		sdl.EVENT_WINDOW_EXPOSED,
		sdl.EVENT_WINDOW_FOCUS_GAINED,
		sdl.EVENT_WINDOW_RESTORED:
		b.Refresh()
	case sdl.EVENT_TEXT_INPUT:
		te := event.TextInputEvent()
		if te == nil {
			return
		}

		for _, r := range te.Text {
			b.SendEvent(tevent.KeyEvent{Key: tevent.KeyRune, Rune: r})
		}
	case sdl.EVENT_KEY_DOWN:
		ke := event.KeyboardEvent()
		if ke == nil {
			return
		}

		if b.handleCtrlModifiedKey(ke.Key, ke.Mod) {
			return
		}

		if key, ok := mapKey(ke.Key, ke.Mod); ok {
			b.SendEvent(tevent.KeyEvent{Key: key, Mod: mapMod(ke.Mod)})
		}
	case sdl.EVENT_MOUSE_MOTION:
		me := event.MouseMotionEvent()
		if me == nil {
			return
		}

		b.SendEvent(b.MapMouse(int(me.X), int(me.Y), tevent.MouseButtonNone, tevent.MouseMove, 0))
	case sdl.EVENT_MOUSE_BUTTON_DOWN, sdl.EVENT_MOUSE_BUTTON_UP:
		me := event.MouseButtonEvent()
		if me == nil {
			return
		}

		action := tevent.MouseRelease
		if me.Down {
			action = tevent.MousePress
		}

		b.SendEvent(b.MapMouse(int(me.X), int(me.Y), mapMouseButton(me.Button), action, 0))
	case sdl.EVENT_MOUSE_WHEEL:
		we := event.MouseWheelEvent()
		if we == nil {
			return
		}

		x, y := we.MouseX, we.MouseY
		if x == 0 && y == 0 {
			_, mx, my := sdl.GetMouseState()

			x, y = mx, my
		}

		steps, button := wheelEventSteps(we)
		if steps > 0 && button != tevent.MouseButtonNone {
			me := b.MapMouse(int(x), int(y), button, tevent.MousePress, 0)
			me.WheelDelta = steps
			b.SendEvent(me)
		}
	}
}

func (b *Backend) handleCtrlModifiedKey(key sdl.Keycode, mod sdl.Keymod) bool {
	if mod&(sdl.KMOD_LCTRL|sdl.KMOD_RCTRL) == 0 {
		return false
	}

	if key == sdl.K_V {
		if text, err := sdl.GetClipboardText(); err == nil {
			b.SendEvent(tevent.PasteEvent{Text: text})
		}

		return true
	}

	if key >= sdl.K_A && key <= sdl.K_Z {
		b.SendEvent(tevent.KeyEvent{
			Key:  tevent.KeyRune,
			Rune: rune(key),
			Mod:  tevent.ModCtrl,
		})

		return true
	}

	if key >= sdl.K_0 && key <= sdl.K_9 {
		b.SendEvent(tevent.KeyEvent{
			Key:  tevent.KeyRune,
			Rune: rune(key) - 0x30,
			Mod:  tevent.ModCtrl,
		})

		return true
	}

	return false
}

// New creates a windowed SDL3 backend.
//
// The first call to Enable on any SDL3 backend loads the SDL3 and SDL3_ttf
// shared libraries via github.com/Zyko0/go-sdl3/bin/binsdl and .../bin/binttf;
// callers do not need to manage library loading themselves.
func New(opts Options) (backend.Backend, error) {
	return newBackend(opts)
}

func wheelEventSteps(e *sdl.MouseWheelEvent) (int, tevent.MouseButton) {
	dx := e.X
	dy := e.Y

	// SDL3 sends MOUSEWHEEL_FLIPPED on some platforms with the natural-scroll
	// convention; invert so up still means up.
	if e.Direction == sdl.MOUSEWHEEL_FLIPPED {
		dx = -dx
		dy = -dy
	}

	value := dy
	if value == 0 {
		value = dx
	}

	button := tevent.MouseButtonNone
	if value > 0 {
		button = tevent.MouseButtonWheelUp
	} else if value < 0 {
		button = tevent.MouseButtonWheelDown
	}

	steps := int(math.Ceil(math.Abs(float64(value))))
	if steps == 0 {
		return 0, tevent.MouseButtonNone
	}

	return steps, button
}

func mapKey(key sdl.Keycode, mod sdl.Keymod) (tevent.Key, bool) {
	switch key {
	case sdl.K_RETURN, sdl.K_RETURN2:
		return tevent.KeyEnter, true
	case sdl.K_ESCAPE:
		return tevent.KeyEsc, true
	case sdl.K_TAB:
		if mod&(sdl.KMOD_LSHIFT|sdl.KMOD_RSHIFT) != 0 {
			return tevent.KeyShiftTab, true
		}

		return tevent.KeyTab, true
	case sdl.K_BACKSPACE:
		return tevent.KeyBackspace, true
	case sdl.K_UP:
		return tevent.KeyUp, true
	case sdl.K_DOWN:
		return tevent.KeyDown, true
	case sdl.K_LEFT:
		return tevent.KeyLeft, true
	case sdl.K_RIGHT:
		return tevent.KeyRight, true
	case sdl.K_HOME:
		return tevent.KeyHome, true
	case sdl.K_END:
		return tevent.KeyEnd, true
	case sdl.K_INSERT:
		return tevent.KeyInsert, true
	case sdl.K_DELETE:
		return tevent.KeyDelete, true
	case sdl.K_PAGEUP:
		return tevent.KeyPageUp, true
	case sdl.K_PAGEDOWN:
		return tevent.KeyPageDown, true
	case sdl.K_F1:
		return tevent.KeyF1, true
	case sdl.K_F2:
		return tevent.KeyF2, true
	case sdl.K_F3:
		return tevent.KeyF3, true
	case sdl.K_F4:
		return tevent.KeyF4, true
	case sdl.K_F5:
		return tevent.KeyF5, true
	case sdl.K_F6:
		return tevent.KeyF6, true
	case sdl.K_F7:
		return tevent.KeyF7, true
	case sdl.K_F8:
		return tevent.KeyF8, true
	case sdl.K_F9:
		return tevent.KeyF9, true
	case sdl.K_F10:
		return tevent.KeyF10, true
	case sdl.K_F11:
		return tevent.KeyF11, true
	case sdl.K_F12:
		return tevent.KeyF12, true
	case sdl.K_C:
		if mod&(sdl.KMOD_LCTRL|sdl.KMOD_RCTRL) != 0 {
			return tevent.KeyCtrlC, true
		}
	default:
	}

	return tevent.KeyNone, false
}

func mapMod(mod sdl.Keymod) tevent.ModMask {
	var out tevent.ModMask
	if mod&(sdl.KMOD_LSHIFT|sdl.KMOD_RSHIFT) != 0 {
		out |= tevent.ModShift
	}

	if mod&(sdl.KMOD_LALT|sdl.KMOD_RALT) != 0 {
		out |= tevent.ModAlt
	}

	if mod&(sdl.KMOD_LCTRL|sdl.KMOD_RCTRL) != 0 {
		out |= tevent.ModCtrl
	}

	return out
}

func mapMouseButton(btn uint8) tevent.MouseButton {
	// SDL3 mouse button indices: 1=left, 2=middle, 3=right, 4=X1, 5=X2.
	switch btn {
	case 1:
		return tevent.MouseButtonLeft
	case 2:
		return tevent.MouseButtonMiddle
	case 3:
		return tevent.MouseButtonRight
	default:
		return tevent.MouseButtonNone
	}
}

func sdlColor(c style.RGBA) sdl.Color {
	return sdl.Color{R: c.R, G: c.G, B: c.B, A: c.A}
}

func alignedGlyphRect(font *ttf.Font, r rune, cellRect cellsurface.PixelRect, glyphW, glyphH int32) cellsurface.PixelRect {
	dst := cellRect
	dst.W = int(glyphW)
	dst.H = int(glyphH)
	dst.X += max(0, (cellRect.W-int(glyphW))/2)
	dst.Y += max(0, (cellRect.H-int(glyphH))/2)

	if gm, err := font.GlyphMetrics(uint32(r)); err == nil {
		advancePad := max(0, (cellRect.W-int(gm.Advance))/2)
		dst.X = cellRect.X + advancePad + int(gm.MinX)
	}

	return dst
}

func drawBoxRune(renderer *sdl.Renderer, r rune, cellRect cellsurface.PixelRect, fg style.RGBA) bool {
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

func fillSDLRect(renderer *sdl.Renderer, x, y, w, h int) error {
	return renderer.RenderFillRect(&sdl.FRect{
		X: float32(x),
		Y: float32(y),
		W: float32(max(1, w)),
		H: float32(max(1, h)),
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

func ttfStyle(attr style.AttrMask) ttf.FontStyleFlags {
	s := ttfStyleNormal
	if attr&style.AttrBold != 0 {
		s |= ttfStyleBold
	}

	if attr&style.AttrItalic != 0 {
		s |= ttfStyleItalic
	}

	if attr&style.AttrUnderline != 0 {
		s |= ttfStyleUnderline
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

	for _, path := range fontCandidates {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", ErrNoMonospaceFont
}

var fontCandidates = []string{
	"/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf",
	"/usr/share/fonts/TTF/DejaVuSansMono.ttf",
	"/usr/share/fonts/truetype/liberation2/LiberationMono-Regular.ttf",
	"/usr/share/fonts/liberation/LiberationMono-Regular.ttf",
}

func measureFontMetrics(font *ttf.Font, fallbackW, fallbackH int) (int, int) {
	cellW := fallbackW
	cellH := fallbackH

	if w, _, err := font.StringSize("M"); err == nil && w > 0 {
		cellW = int(w)
	}

	if h := font.LineSkip(); h > 0 {
		cellH = int(h)
	} else if h := font.Height(); h > 0 {
		cellH = int(h)
	}

	if cellW <= 0 {
		cellW = 8
	}

	if cellH <= 0 {
		cellH = 16
	}

	return cellW, cellH
}
