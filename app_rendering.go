package tui

import (
	"github.com/losinggeneration/tui/backend"
	"github.com/losinggeneration/tui/geom"
	"github.com/losinggeneration/tui/render"
	"github.com/losinggeneration/tui/style"
)

type runtimeFrame interface {
	Painter() *Painter
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
	Attach(backend.Backend)
	InitScreen() error
	RestoreScreen() error
	ResetCursor()
	ResetStyle()
	PresentFrame(runtimeFrame) error
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

func (r *cellRenderer) Painter(size geom.Size, base style.Style) *Painter {
	clip := geom.Rect{X: 0, Y: 0, W: size.W, H: size.H}
	rp := render.NewPainter(r.backBuf, clip, base)
	return NewPainter(rp, base)
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

func (f *cellFrame) Painter() *Painter {
	clip := geom.Rect{X: 0, Y: 0, W: f.size.W, H: f.size.H}
	rp := render.NewPainter(f.backBuf, clip, f.base)
	return NewPainter(rp, f.base)
}

func makeBackendCellFrame(f *cellFrame) backend.CellFrame {
	cells := make([]backend.FrameCell, f.size.W*f.size.H)

	for y := 0; y < f.size.H; y++ {
		for x := 0; x < f.size.W; x++ {
			cell := f.backBuf.At(x, y)
			cells[y*f.size.W+x] = backend.FrameCell{
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
		Cells: cells,
	}
}

type ansiPresenter struct {
	backend backend.Backend
	flusher *render.ANSIFlusher
}

func newANSIPresenter() *ansiPresenter {
	return &ansiPresenter{
		flusher: render.NewANSIFlusher(nil),
	}
}

func (p *ansiPresenter) Attach(b backend.Backend) {
	p.backend = b
	p.flusher = render.NewANSIFlusher(&backendWriter{b: b})
}

func (p *ansiPresenter) InitScreen() error {
	if err := p.flusher.ClearScreen(); err != nil {
		return err
	}

	if err := p.flusher.HideCursor(); err != nil {
		return err
	}

	return p.flusher.Flush()
}

func (p *ansiPresenter) RestoreScreen() error {
	if err := p.flusher.ShowCursor(); err != nil {
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
		if err := p.flusher.FlushRuns(f.backBuf, f.frontBuf, runs); err != nil {
			return err
		}
	}

	if p.backend != nil {
		return p.backend.Flush()
	}

	return nil
}

type cellFramePresenter struct {
	sink backend.CellFrameSink
}

func newCellFramePresenter() *cellFramePresenter {
	return &cellFramePresenter{}
}

func (p *cellFramePresenter) Attach(b backend.Backend) {
	sink, _ := b.(backend.CellFrameSink)
	p.sink = sink
}

func (p *cellFramePresenter) InitScreen() error {
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

	return p.sink.PresentCellFrame(makeBackendCellFrame(f))
}

func presenterForBackend(b backend.Backend) runtimePresenter {
	if _, ok := b.(backend.CellFrameSink); ok {
		return newCellFramePresenter()
	}

	return newANSIPresenter()
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

	ctx := a.mkCtx(a.root)
	p := frame.Painter()
	a.paintView(a.root, p, ctx)
	a.overlays.paintOverlays(p, ctx)
}

func (a *App) presentFrame() {
	frame := a.renderer.Frame(a.size, a.resolvedTheme.Base)
	a.errs.Add(a.presenter.PresentFrame(frame))
}

func (a *App) paintView(v View, p *Painter, ctx *Ctx) {
	PaintView(v, p, ctx)
}
