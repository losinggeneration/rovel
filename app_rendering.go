package rovel

import (
	"errors"
	"fmt"

	"github.com/losinggeneration/rovel/backend"
	"github.com/losinggeneration/rovel/geom"
	"github.com/losinggeneration/rovel/render"
	"github.com/losinggeneration/rovel/style"
)

var errNoSinkOrTransport = errors.New("backend provides neither CellFrameSink nor ANSITransport")

type runtimeFrame interface {
	Drawer() Drawer
}

type runtimeRenderer interface {
	Resize(w, h int)
	ResetDamage()
	AddDamageRects(rects []geom.Rect)
	IsDamaged() bool
	ResetFrontBuffer()
	ClearDamaged(base style.Style)
	Frame(size geom.Size, base style.Style) runtimeFrame
}

type runtimePresenter interface {
	InitScreen(size geom.Size) error
	RestoreScreen() error
	ResetCursor()
	ResetStyle()
	PresentFrame(frame runtimeFrame) error
}

type cellRenderer struct {
	backBuf  *render.Buffer
	frontBuf *render.Buffer
	damage   *render.Damage
}

type cellFrame struct {
	backBuf  *render.Buffer
	frontBuf *render.Buffer
	damage   *render.Damage
	size     geom.Size
	base     style.Style
}

func newCellRenderer(size geom.Size) *cellRenderer {
	return &cellRenderer{
		backBuf:  render.NewBuffer(size.W, size.H),
		frontBuf: render.NewBuffer(size.W, size.H),
		damage:   render.NewDamage(size.W, size.H),
	}
}

func (r *cellRenderer) Resize(w, h int) {
	r.backBuf.Resize(w, h)
	r.frontBuf.Resize(w, h)
	r.damage.Reset(w, h)
}

func (r *cellRenderer) ResetDamage() {
	r.damage.Clear()
}

func (r *cellRenderer) AddDamageRect(rect geom.Rect) {
	r.damage.AddRect(rect)
}

func (r *cellRenderer) AddDamageRects(rects []geom.Rect) {
	for _, rect := range rects {
		r.damage.AddRect(rect)
	}
}

func (r *cellRenderer) IsDamaged() bool {
	return !r.damage.IsEmpty()
}

func (r *cellRenderer) ResetFrontBuffer() {
	r.frontBuf.Clear(render.Cell{})
}

func (r *cellRenderer) ClearDamaged(base style.Style) {
	baseCell := render.Cell{
		R:     ' ',
		Style: base,
		Wide:  false,
	}

	for y := range r.damage.H {
		spans := r.damage.Rows[y]
		for _, sp := range spans {
			for x := sp.X0; x < sp.X1; x++ {
				cell := r.backBuf.At(x, y)
				*cell = baseCell
			}
		}
	}
}

func (r *cellRenderer) Frame(size geom.Size, base style.Style) runtimeFrame {
	return &cellFrame{
		backBuf:  r.backBuf,
		frontBuf: r.frontBuf,
		damage:   r.damage,
		size:     size,
		base:     base,
	}
}

func (f *cellFrame) Drawer() Drawer {
	clip := geom.Rect{X: 0, Y: 0, W: f.size.W, H: f.size.H}
	rp := render.NewPainter(f.backBuf, clip, f.base)

	return NewDrawer(NewPainter(rp, f.base))
}

// cellFrameFor fills the presenter's reusable cell buffer from f and returns a
// CellFrame borrowing it. The returned Cells slice is valid only until the next
// call; sinks that retain the frame must copy it.
func (p *cellFramePresenter) cellFrameFor(f *cellFrame) backend.CellFrame {
	n := f.size.W * f.size.H
	if cap(p.cells) < n {
		p.cells = make([]backend.FrameCell, n)
	}

	p.cells = p.cells[:n]

	for y := range f.size.H {
		for x := range f.size.W {
			cell := f.backBuf.At(x, y)
			p.cells[y*f.size.W+x] = backend.FrameCell{
				R:        cell.R,
				Style:    cell.Style,
				Wide:     cell.Wide,
				WideCont: cell.WideCont,
			}
		}
	}

	return backend.CellFrame{
		W:     f.size.W,
		H:     f.size.H,
		Cells: p.cells,
	}
}

type ansiPresenter struct {
	transport   backend.ANSITransport
	flusher     *render.ANSIFlusher
	mode        backend.TerminalMode
	regionH     int // inline-region height, set in InitScreen for cbreak
	clearOnExit bool
}

func newANSIPresenter(t backend.ANSITransport, mode backend.TerminalMode, clearOnExit bool) *ansiPresenter {
	return &ansiPresenter{
		transport:   t,
		flusher:     render.NewANSIFlusher(&transportWriter{t: t}),
		mode:        mode,
		clearOnExit: clearOnExit,
	}
}

func (p *ansiPresenter) InitScreen(size geom.Size) error {
	if p.mode == backend.ModeCBreak {
		// Create an inline region of size.H rows anchored at the current
		// cursor position. Relative positioning keeps the region stable.
		p.regionH = size.H

		if err := p.flusher.SetupInlineRegion(size.H); err != nil {
			return err
		}
	} else {
		// Raw mode: clear the screen so rendering starts from a blank state.
		if err := p.flusher.ClearScreen(); err != nil {
			return err
		}
	}

	if err := p.flusher.HideCursor(); err != nil {
		return err
	}

	return p.flusher.Flush()
}

func (p *ansiPresenter) RestoreScreen() error {
	// In cbreak mode, either clear the inline region (erase the rendered
	// content and reset the cursor to where rendering began) or park the
	// cursor just below it so the next shell prompt lands on a fresh line.
	if p.mode == backend.ModeCBreak && p.regionH > 0 {
		var err error
		if p.clearOnExit {
			err = p.flusher.ClearInlineRegion(p.regionH)
		} else {
			err = p.flusher.LeaveInlineRegion(p.regionH)
		}

		if err != nil {
			return err
		}
	}

	// Reset SGR so we don't leak styles into the shell after exit.
	if _, err := p.transport.Write([]byte("\x1b[0m")); err != nil {
		return err
	}

	err := p.flusher.ShowCursor()
	if err != nil {
		return err
	}

	return p.flusher.Flush()
}

func (p *ansiPresenter) ResetCursor() {
	p.flusher.ResetCursor()
}

func (p *ansiPresenter) ResetStyle() {
	p.flusher.ResetStyle()
}

func (p *ansiPresenter) PresentFrame(frame runtimeFrame) error {
	f, ok := frame.(*cellFrame)
	if !ok {
		return nil
	}

	runs := render.DiffRuns(f.backBuf, f.frontBuf, f.damage)
	if len(runs) > 0 {
		err := p.flusher.FlushRuns(f.backBuf, f.frontBuf, runs)
		if err != nil {
			return err
		}
	}

	return p.transport.Flush()
}

type cellFramePresenter struct {
	sink backend.CellFrameSink
	// cells is reused across frames to avoid allocating a W×H slice per
	// present. The CellFrame handed to the sink borrows it, so sinks that
	// retain the frame must copy (the memory and SDL backends do).
	cells []backend.FrameCell
}

func newCellFramePresenter(sink backend.CellFrameSink) *cellFramePresenter {
	return &cellFramePresenter{sink: sink}
}

func (p *cellFramePresenter) InitScreen(_ geom.Size) error {
	return nil
}

func (p *cellFramePresenter) RestoreScreen() error {
	return nil
}

func (p *cellFramePresenter) ResetCursor() {}

func (p *cellFramePresenter) ResetStyle() {}

func (p *cellFramePresenter) PresentFrame(frame runtimeFrame) error {
	if p.sink == nil {
		return nil
	}

	f, ok := frame.(*cellFrame)
	if !ok {
		return nil
	}

	return p.sink.PresentCellFrame(p.cellFrameFor(f))
}

func presenterForBackend(b backend.Backend, mode backend.TerminalMode, clearOnExit bool) (runtimePresenter, error) {
	if sink, ok := b.(backend.CellFrameSink); ok {
		return newCellFramePresenter(sink), nil
	}

	if t, ok := b.(backend.ANSITransport); ok {
		return newANSIPresenter(t, mode, clearOnExit), nil
	}

	return nil, fmt.Errorf("%w: %T", errNoSinkOrTransport, b)
}

func (a *App) beginFrame(rects []geom.Rect) bool {
	a.renderer.ResetDamage()
	a.renderer.AddDamageRects(rects)

	return a.renderer.IsDamaged()
}

func (a *App) paintFramePass() {
	if a.root == nil {
		return
	}

	a.renderer.ClearDamaged(a.resolvedTheme.Base)
	frame := a.renderer.Frame(a.size, a.resolvedTheme.Base)

	ctx := a.mkCtx()
	d := frame.Drawer()
	a.paintView(a.root, d, ctx)
	a.overlays.paintOverlays(d, ctx)
}

func (a *App) presentFrame() {
	frame := a.renderer.Frame(a.size, a.resolvedTheme.Base)
	a.errs.Add(a.presenter.PresentFrame(frame))
}

func (a *App) paintView(v View, d Drawer, ctx *Ctx) {
	v.Paint(d, ctx)
}
