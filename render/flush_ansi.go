package render

import (
	"bufio"
	"fmt"
	"io"
	"unicode/utf8"

	"github.com/losinggeneration/tui/style"
)

type ANSIFlusher struct {
	w *bufio.Writer

	curX int
	curY int

	curStyle style.Style
	hasStyle bool
}

func NewANSIFlusher(w io.Writer) *ANSIFlusher {
	return &ANSIFlusher{w: bufio.NewWriterSize(w, 64*1024)}
}

// ResetStyle resets the style tracking state.
// Call this when the theme changes to force re-emission of all SGR codes.
func (f *ANSIFlusher) ResetStyle() {
	f.curStyle = style.Style{}
	f.hasStyle = false
}

// ResetCursor resets the tracked cursor position so the next move emits an
// absolute positioning sequence. Call after terminal resize.
func (f *ANSIFlusher) ResetCursor() {
	f.curX = -1
	f.curY = -1
}

func (f *ANSIFlusher) FlushRuns(back, front *Buffer, runs []Run) error {
	for _, run := range runs {
		y := run.Y
		x := run.X0

		err := f.moveCursorTo(y, x)
		if err != nil {
			return err
		}

		for x < run.X1 {
			c := *back.At(x, y)

			if c.WideCont {
				x++
				f.curX++

				continue
			}

			if !f.hasStyle || c.Style != f.curStyle {
				err := f.emitSGR(c.Style)
				if err != nil {
					return err
				}

				f.curStyle = c.Style
				f.hasStyle = true
			}

			err := f.emitRune(c.R)
			if err != nil {
				return err
			}

			*front.At(x, y) = c

			if c.Wide {
				if x+1 < back.W {
					*front.At(x+1, y) = *back.At(x+1, y)
				}

				x += 2
				f.curX += 2

				continue
			}

			x++
			f.curX++
		}
	}

	return f.w.Flush()
}

// appendFGColor appends foreground color SGR parameters.
func appendFGColor(params []int, c style.Color) []int {
	switch c.Kind() {
	case style.ColorKindBasic:
		idx, _ := c.BasicIndex()
		if idx <= 7 {
			return append(params, 30+int(idx))
		}

		return append(params, 90+int(idx-8))
	case style.ColorKindIndexed:
		idx, _ := c.Index()

		return append(params, 38, 5, int(idx))
	case style.ColorKindRGB:
		r, g, b, _ := c.RGB()

		return append(params, 38, 2, int(r), int(g), int(b))
	case style.ColorKindDefault:
		return params
	}

	return params
}

// appendBGColor appends background color SGR parameters.
func appendBGColor(params []int, c style.Color) []int {
	switch c.Kind() {
	case style.ColorKindBasic:
		idx, _ := c.BasicIndex()
		if idx <= 7 {
			return append(params, 40+int(idx))
		}

		return append(params, 100+int(idx-8))
	case style.ColorKindIndexed:
		idx, _ := c.Index()

		return append(params, 48, 5, int(idx))
	case style.ColorKindRGB:
		r, g, b, _ := c.RGB()

		return append(params, 48, 2, int(r), int(g), int(b))
	case style.ColorKindDefault:
		return params
	}

	return params
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

// ClearScreen emits the ANSI escape sequence to clear the entire screen.
func (f *ANSIFlusher) ClearScreen() error {
	_, err := f.w.WriteString("\x1b[2J")
	if err != nil {
		return err
	}
	// Move cursor to home
	_, err = f.w.WriteString("\x1b[H")
	f.curX = 0
	f.curY = 0

	return err
}

// HideCursor hides the cursor.
func (f *ANSIFlusher) HideCursor() error {
	_, err := f.w.WriteString("\x1b[?25l")

	return err
}

// ShowCursor shows the cursor.
func (f *ANSIFlusher) ShowCursor() error {
	_, err := f.w.WriteString("\x1b[?25h")

	return err
}

// Flush flushes the underlying buffer.
func (f *ANSIFlusher) Flush() error {
	return f.w.Flush()
}

func (f *ANSIFlusher) moveCursorTo(y, x int) error {
	if f.curX == x && f.curY == y {
		return nil
	}

	_, err := fmt.Fprintf(f.w, "\x1b[%d;%dH", y+1, x+1)
	if err != nil {
		return err
	}

	f.curX = x
	f.curY = y

	return nil
}

func (f *ANSIFlusher) emitSGR(s style.Style) error {
	params := []int{0}

	// Special case: if truly empty style, emit reset
	if s == (style.Style{}) {
		return emitSGRParams(f.w, []int{0})
	}

	// Foreground color
	if s.FG.IsDefault() {
		params = append(params, 39) // Explicit default foreground color
	} else {
		params = appendFGColor(params, s.FG)
	}

	// Background color
	if s.BG.IsDefault() {
		params = append(params, 49) // Explicit default background color
	} else {
		params = appendBGColor(params, s.BG)
	}

	// Attributes (1=bold, 2=dim, 3=italic, 4=underline, 5=blink, 7=reverse)
	if s.Attr&style.AttrBold != 0 {
		params = append(params, 1)
	}

	if s.Attr&style.AttrDim != 0 {
		params = append(params, 2)
	}

	if s.Attr&style.AttrItalic != 0 {
		params = append(params, 3)
	}

	if s.Attr&style.AttrUnderline != 0 {
		params = append(params, 4)
	}

	if s.Attr&style.AttrBlink != 0 {
		params = append(params, 5)
	}

	if s.Attr&style.AttrReverse != 0 {
		params = append(params, 7)
	}

	return emitSGRParams(f.w, params)
}

func (f *ANSIFlusher) emitRune(r rune) error {
	if r == 0 {
		r = ' '
	}

	var buf [utf8.UTFMax]byte

	n := utf8.EncodeRune(buf[:], r)
	_, err := f.w.Write(buf[:n])

	return err
}
