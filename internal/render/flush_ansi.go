package render

import (
	"bufio"
	"fmt"
	"io"
	"unicode/utf8"

	"github.com/losinggeneration/tui"
)

type ANSIFlusher struct {
	W *bufio.Writer

	CurX int
	CurY int

	CurStyle tui.Style
	HasStyle bool
}

func NewANSIFlusher(w io.Writer) *ANSIFlusher {
	return &ANSIFlusher{W: bufio.NewWriterSize(w, 64*1024)}
}

func (f *ANSIFlusher) FlushRuns(back, front *Buffer, runs []Run) error {
	for _, run := range runs {
		y := run.Y
		x := run.X0

		if err := f.moveCursorTo(y, x); err != nil {
			return err
		}

		for x < run.X1 {
			c := *back.At(x, y)

			if c.WideCont {
				x++
				f.CurX++
				continue
			}

			if !f.HasStyle || c.Style != f.CurStyle {
				if err := f.emitSGR(c.Style); err != nil {
					return err
				}
				f.CurStyle = c.Style
				f.HasStyle = true
			}

			if err := f.emitRune(c.R); err != nil {
				return err
			}

			*front.At(x, y) = c

			if c.Wide {
				if x+1 < back.W {
					*front.At(x+1, y) = *back.At(x+1, y)
				}
				x += 2
				f.CurX += 2
				continue
			}

			x++
			f.CurX++
		}
	}

	return f.W.Flush()
}

func (f *ANSIFlusher) moveCursorTo(y, x int) error {
	if f.CurX == x && f.CurY == y {
		return nil
	}

	_, err := fmt.Fprintf(f.W, "\x1b[%d;%dH", y+1, x+1)
	if err != nil {
		return err
	}

	f.CurX = x
	f.CurY = y
	return nil
}

func (f *ANSIFlusher) emitSGR(s tui.Style) error {
	// Build SGR parameters - start with reset (0)
	params := []int{0}

	// Foreground color (30-37 for basic, 90-97 for bright, 38;5;n for 256-color)
	if s.FG != tui.ColorDefault {
		params = appendFGColor(params, s.FG)
	}

	// Background color (40-47 for basic, 100-107 for bright, 48;5;n for 256-color)
	if s.BG != tui.ColorDefault {
		params = appendBGColor(params, s.BG)
	}

	// Attributes (1=bold, 2=dim, 3=italic, 4=underline, 5=blink, 7=reverse)
	if s.Attr&tui.AttrBold != 0 {
		params = append(params, 1)
	}
	if s.Attr&tui.AttrDim != 0 {
		params = append(params, 2)
	}
	if s.Attr&tui.AttrItalic != 0 {
		params = append(params, 3)
	}
	if s.Attr&tui.AttrUnderline != 0 {
		params = append(params, 4)
	}
	if s.Attr&tui.AttrBlink != 0 {
		params = append(params, 5)
	}
	if s.Attr&tui.AttrReverse != 0 {
		params = append(params, 7)
	}

	return emitSGRParams(f.W, params)
}

// appendFGColor appends foreground color SGR parameters.
func appendFGColor(params []int, c tui.Color) []int {
	if c == tui.ColorDefault {
		return params
	}
	// Check for 256-color palette (colors 232-255)
	if c >= tui.Color256 {
		return append(params, 38, 5, int(c))
	}
	// Basic 16 colors: 30-37 (normal), 90-97 (bright)
	// Color constants are 1-indexed (ColorBlack=1), ANSI is 30 for black
	if c <= tui.ColorWhite {
		return append(params, 30+int(c-1))
	}
	// Bright colors (ColorBrightBlack=9, ANSI 90)
	return append(params, 90+int(c-tui.ColorBrightBlack))
}

// appendBGColor appends background color SGR parameters.
func appendBGColor(params []int, c tui.Color) []int {
	if c == tui.ColorDefault {
		return params
	}
	// Check for 256-color palette (colors 232-255)
	if c >= tui.Color256 {
		return append(params, 48, 5, int(c))
	}
	// Basic 16 colors: 40-47 (normal), 100-107 (bright)
	// Color constants are 1-indexed (ColorBlack=1), ANSI is 40 for black
	if c <= tui.ColorWhite {
		return append(params, 40+int(c-1))
	}
	// Bright colors (ColorBrightBlack=9, ANSI 100)
	return append(params, 100+int(c-tui.ColorBrightBlack))
}

// emitSGRParams writes the SGR escape sequence with parameters.
func emitSGRParams(w *bufio.Writer, params []int) error {
	if _, err := w.WriteString("\x1b["); err != nil {
		return err
	}
	for i, p := range params {
		if i > 0 {
			if _, err := w.WriteString(";"); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(w, "%d", p); err != nil {
			return err
		}
	}
	_, err := w.WriteString("m")
	return err
}

func (f *ANSIFlusher) emitRune(r rune) error {
	if r == 0 {
		r = ' '
	}

	var buf [utf8.UTFMax]byte
	n := utf8.EncodeRune(buf[:], r)
	_, err := f.W.Write(buf[:n])
	return err
}
