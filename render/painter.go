package render

import (
	"github.com/losinggeneration/tui/geom"
	"github.com/losinggeneration/tui/style"
)

// Painter provides an immediate-mode drawing API for rendering to a buffer.
type Painter struct {
	buf       *Buffer
	clip      geom.Rect
	baseStyle style.Style
	offsetX   int
	offsetY   int
}

// NewPainter creates a new painter.
func NewPainter(buf *Buffer, clip geom.Rect, baseStyle style.Style) *Painter {
	return &Painter{
		buf:       buf,
		clip:      clip,
		baseStyle: baseStyle,
	}
}

// Offset returns the current offset.
func (p *Painter) Offset() (x, y int) {
	return p.offsetX, p.offsetY
}

// SetOffset sets the drawing offset. All coordinates are offset by this amount.
func (p *Painter) SetOffset(x, y int) {
	p.offsetX = x
	p.offsetY = y
}

// SetCell writes a single cell with proper wide-char handling.
// Coordinates are offset by the painter's offset.
func (p *Painter) SetCell(x, y int, r rune, style style.Style) {
	p.setCellAt(x+p.offsetX, y+p.offsetY, r, style)
}

// setCellAt writes a cell at the exact coordinates without applying offset.
// This is used internally when offset has already been applied.
func (p *Painter) setCellAt(x, y int, r rune, style style.Style) {
	if IsClipped(x, y, p.clip) {
		return
	}

	width := RuneWidth(r)
	if width == 0 {
		// The buffer is cell-based and cannot represent width-zero code points
		// (combining marks, ZWJ, variation selectors) without corrupting the
		// surrounding cells. Drop them consistently.
		return
	}

	// Handle edge case: wide character at buffer edge
	if width == 2 && x+1 >= p.buf.W {
		r = '?'
		width = 1
	}

	// Wide-char overwrite rules (design doc 16.4):
	// 1. If target is a continuation, clear the lead at (x-1, y)
	// 2. If target is a wide lead and we're writing narrow, clear continuation at (x+1, y)
	// 3. If writing wide, check for wide-glyph collision at (x+1, y)

	cell := p.buf.At(x, y)

	// Rule 1: Clear lead if target is a continuation
	if cell.WideCont && x > 0 {
		leadCell := p.buf.At(x-1, y)
		if leadCell.Wide {
			*leadCell = Cell{R: ' ', Style: p.baseStyle}
		}
	}

	// Rule 2: Clear continuation if we're overwriting a wide lead with narrow
	if cell.Wide && width == 1 && x+1 < p.buf.W {
		contCell := p.buf.At(x+1, y)
		*contCell = Cell{R: ' ', Style: p.baseStyle}
	}

	// Rule 3: Check for wide-glyph collision
	if width == 2 && x+1 < p.buf.W {
		nextCell := p.buf.At(x+1, y)
		// If next cell is a wide lead, clear its continuation
		if nextCell.Wide && x+2 < p.buf.W {
			contCell := p.buf.At(x+2, y)
			*contCell = Cell{R: ' ', Style: p.baseStyle}
		}
	}

	// Write the cell
	cell.R = r
	cell.Style = style
	cell.Wide = (width == 2)
	cell.WideCont = false

	// Write continuation cell for wide characters
	if width == 2 && x+1 < p.buf.W {
		contCell := p.buf.At(x+1, y)
		contCell.R = 0
		contCell.Style = style
		contCell.Wide = false
		contCell.WideCont = true
	}
}

// ClipRect returns the current clip rect.
func (p *Painter) ClipRect() geom.Rect {
	return p.clip
}

// SetClipRect sets the clip rect (no normalization, caller responsible).
func (p *Painter) SetClipRect(r geom.Rect) {
	p.clip = r
}

// Text writes a string at position, advancing by rune width.
func (p *Painter) Text(x, y int, s string, style style.Style) {
	x += p.offsetX
	y += p.offsetY
	for _, r := range s {
		if IsClipped(x, y, p.clip) {
			break
		}
		p.setCellAt(x, y, r, style)
		x += RuneWidth(r)
	}
}

// Fill fills a rect with a repeated rune.
// The rect is in logical coordinates and will be offset and clipped.
func (p *Painter) Fill(r geom.Rect, ch rune, style style.Style) {
	// Apply offset to get absolute coordinates
	r.X += p.offsetX
	r.Y += p.offsetY
	r = ClipRect(r, p.clip)
	if r.Empty() {
		return
	}

	for y := r.Y; y < r.Y+r.H; y++ {
		for x := r.X; x < r.X+r.W; x++ {
			p.setCellAt(x, y, ch, style)
		}
	}
}

// HLine draws a horizontal line.
// Coordinates are offset by the painter's offset.
func (p *Painter) HLine(x, y, w int, ch rune, style style.Style) {
	r := geom.Rect{X: x, Y: y, W: w, H: 1}
	p.Fill(r, ch, style)
}

// VLine draws a vertical line.
// Coordinates are offset by the painter's offset.
func (p *Painter) VLine(x, y, h int, ch rune, style style.Style) {
	r := geom.Rect{X: x, Y: y, W: 1, H: h}
	p.Fill(r, ch, style)
}

// BoxEdges is a bitmask of which edges to draw.
type BoxEdges uint8

const (
	BoxEdgeTop BoxEdges = 1 << iota
	BoxEdgeRight
	BoxEdgeBottom
	BoxEdgeLeft

	BoxEdgesAll = BoxEdgeTop | BoxEdgeRight | BoxEdgeBottom | BoxEdgeLeft
)

// BoxGlyphs defines the runes used to draw a box.
type BoxGlyphs struct {
	H  rune // horizontal edge
	V  rune // vertical edge
	TL rune // top-left corner
	TR rune // top-right corner
	BL rune // bottom-left corner
	BR rune // bottom-right corner
}

// BoxPart indicates which portion of the box is being painted.
type BoxPart uint8

const (
	BoxPartTop BoxPart = iota + 1
	BoxPartRight
	BoxPartBottom
	BoxPartLeft
	BoxPartCornerTL
	BoxPartCornerTR
	BoxPartCornerBL
	BoxPartCornerBR
)

// BoxStyle controls how a box border is drawn.
type BoxStyle struct {
	Glyphs BoxGlyphs
	Edges  BoxEdges

	// Style is used when StyleFn is nil.
	Style style.Style

	// StyleFn optionally provides per-cell styling (for gradients, per-edge
	// emphasis, etc). If non-nil, it is used for every cell written.
	StyleFn func(part BoxPart, x, y int, r geom.Rect) style.Style
}

var (
	BoxGlyphsLight = BoxGlyphs{
		H:  '─',
		V:  '│',
		TL: '┌',
		TR: '┐',
		BL: '└',
		BR: '┘',
	}
	BoxGlyphsASCII = BoxGlyphs{
		H:  '-',
		V:  '|',
		TL: '+',
		TR: '+',
		BL: '+',
		BR: '+',
	}
)

// Box draws a box border.
func (p *Painter) Box(r geom.Rect, style style.Style) {
	p.BoxStyled(r, BoxStyle{
		Glyphs: BoxGlyphsLight,
		Edges:  BoxEdgesAll,
		Style:  style,
	})
}

// BoxStyled draws a box border with configurable glyphs, edges, and optional
// per-cell styling.
func (p *Painter) BoxStyled(r geom.Rect, bs BoxStyle) {
	if r.Empty() {
		return
	}

	// Apply offset
	r.X += p.offsetX
	r.Y += p.offsetY

	g := bs.Glyphs
	edges := bs.Edges

	cellStyle := func(part BoxPart, x, y int) style.Style {
		if bs.StyleFn != nil {
			return bs.StyleFn(part, x, y, r)
		}
		return bs.Style
	}

	// Edges (inclusive of corners; corners may be overwritten below).
	if edges&BoxEdgeTop != 0 {
		y := r.Y
		for x := r.X; x < r.X+r.W; x++ {
			p.setCellAt(x, y, g.H, cellStyle(BoxPartTop, x, y))
		}
	}
	if edges&BoxEdgeBottom != 0 && r.H > 1 {
		y := r.Y + r.H - 1
		for x := r.X; x < r.X+r.W; x++ {
			p.setCellAt(x, y, g.H, cellStyle(BoxPartBottom, x, y))
		}
	}

	if edges&BoxEdgeLeft != 0 {
		x := r.X
		for y := r.Y; y < r.Y+r.H; y++ {
			p.setCellAt(x, y, g.V, cellStyle(BoxPartLeft, x, y))
		}
	}
	if edges&BoxEdgeRight != 0 && r.W > 0 {
		x := r.X + r.W - 1
		for y := r.Y; y < r.Y+r.H; y++ {
			p.setCellAt(x, y, g.V, cellStyle(BoxPartRight, x, y))
		}
	}

	// Corners only when both adjacent edges are present. This supports styles like
	// left/right bars without top/bottom edges.
	if r.W > 0 && r.H > 0 {
		if edges&BoxEdgeTop != 0 && edges&BoxEdgeLeft != 0 {
			x, y := r.X, r.Y
			p.setCellAt(x, y, g.TL, cellStyle(BoxPartCornerTL, x, y))
		}
		if edges&BoxEdgeTop != 0 && edges&BoxEdgeRight != 0 {
			x, y := r.X+r.W-1, r.Y
			p.setCellAt(x, y, g.TR, cellStyle(BoxPartCornerTR, x, y))
		}
		if edges&BoxEdgeBottom != 0 && edges&BoxEdgeLeft != 0 && r.H > 1 {
			x, y := r.X, r.Y+r.H-1
			p.setCellAt(x, y, g.BL, cellStyle(BoxPartCornerBL, x, y))
		}
		if edges&BoxEdgeBottom != 0 && edges&BoxEdgeRight != 0 && r.H > 1 {
			x, y := r.X+r.W-1, r.Y+r.H-1
			p.setCellAt(x, y, g.BR, cellStyle(BoxPartCornerBR, x, y))
		}
	}
}
