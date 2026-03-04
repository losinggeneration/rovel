package render

import (
	"github.com/losinggeneration/tui/geom"
	"github.com/losinggeneration/tui/style"
)

// Painter provides an immediate-mode drawing API for rendering to a buffer.
type Painter struct {
	buf    *Buffer
	clip   geom.Rect
	damage *Damage
}

// NewPainter creates a new painter.
func NewPainter(buf *Buffer, clip geom.Rect, damage *Damage) *Painter {
	return &Painter{
		buf:    buf,
		clip:   clip,
		damage: damage,
	}
}

// SetCell writes a single cell with proper wide-char handling.
func (p *Painter) SetCell(x, y int, r rune, style style.Style) {
	if IsClipped(x, y, p.clip) {
		return
	}

	width := RuneWidth(r)

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
			leadCell.R = ' '
			leadCell.Wide = false
			p.damage.AddSpan(y, x-1, x)
		}
	}

	// Rule 2: Clear continuation if we're overwriting a wide lead with narrow
	if cell.Wide && width == 1 && x+1 < p.buf.W {
		contCell := p.buf.At(x+1, y)
		contCell.R = ' '
		contCell.WideCont = false
		p.damage.AddSpan(y, x+1, x+2)
	}

	// Rule 3: Check for wide-glyph collision
	if width == 2 && x+1 < p.buf.W {
		nextCell := p.buf.At(x+1, y)
		// If next cell is a wide lead, clear its continuation
		if nextCell.Wide && x+2 < p.buf.W {
			contCell := p.buf.At(x+2, y)
			contCell.R = ' '
			contCell.WideCont = false
			p.damage.AddSpan(y, x+2, x+3)
		}
	}

	// Write the cell
	cell.R = r
	cell.Style = style
	cell.Wide = (width == 2)
	cell.WideCont = false

	// Mark damage
	p.damage.AddSpan(y, x, x+width)

	// Write continuation cell for wide characters
	if width == 2 && x+1 < p.buf.W {
		contCell := p.buf.At(x+1, y)
		contCell.R = 0
		contCell.Style = style
		contCell.Wide = false
		contCell.WideCont = true
	}
}

// Text writes a string at position, advancing by rune width.
func (p *Painter) Text(x, y int, s string, style style.Style) {
	for _, r := range s {
		if IsClipped(x, y, p.clip) {
			break
		}
		p.SetCell(x, y, r, style)
		x += RuneWidth(r)
	}
}

// Fill fills a rect with a repeated rune.
func (p *Painter) Fill(r geom.Rect, ch rune, style style.Style) {
	r = ClipRect(r, p.clip)
	if r.Empty() {
		return
	}

	for y := r.Y; y < r.Y+r.H; y++ {
		for x := r.X; x < r.X+r.W; x++ {
			p.SetCell(x, y, ch, style)
		}
	}
}

// HLine draws a horizontal line.
func (p *Painter) HLine(x, y, w int, ch rune, style style.Style) {
	r := geom.Rect{X: x, Y: y, W: w, H: 1}
	p.Fill(r, ch, style)
}

// VLine draws a vertical line.
func (p *Painter) VLine(x, y, h int, ch rune, style style.Style) {
	r := geom.Rect{X: x, Y: y, W: 1, H: h}
	p.Fill(r, ch, style)
}

// Box draws a box border.
func (p *Painter) Box(r geom.Rect, style style.Style) {
	if r.Empty() {
		return
	}

	// Horizontal lines
	p.HLine(r.X, r.Y, r.W, '─', style)
	if r.H > 1 {
		p.HLine(r.X, r.Y+r.H-1, r.W, '─', style)
	}

	// Vertical lines
	p.VLine(r.X, r.Y, r.H, '│', style)
	if r.W > 1 {
		p.VLine(r.X+r.W-1, r.Y, r.H, '│', style)
	}

	// Corners
	if r.W > 0 && r.H > 0 {
		p.SetCell(r.X, r.Y, '┌', style)       // Top-left
		p.SetCell(r.X+r.W-1, r.Y, '┐', style) // Top-right
	}
	if r.W > 0 && r.H > 1 {
		p.SetCell(r.X, r.Y+r.H-1, '└', style)       // Bottom-left
		p.SetCell(r.X+r.W-1, r.Y+r.H-1, '┘', style) // Bottom-right
	}
}
