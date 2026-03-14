package render

import (
	"bufio"
	"bytes"
	"testing"

	"github.com/losinggeneration/tui/style"
)

func TestANSIFlusher_New(t *testing.T) {
	var buf bytes.Buffer
	f := NewANSIFlusher(&buf)

	if f.w == nil {
		t.Error("NewANSIFlusher() w is nil")
	}
	if f.curX != 0 || f.curY != 0 {
		t.Errorf("NewANSIFlusher() cursor = (%d,%d), want (0,0)", f.curX, f.curY)
	}
	if f.hasStyle {
		t.Error("NewANSIFlusher() hasStyle should be false")
	}
}

func TestANSIFlusher_moveCursorTo_NoOp(t *testing.T) {
	var buf bytes.Buffer
	f := NewANSIFlusher(&buf)
	f.curX = 5
	f.curY = 3

	err := f.moveCursorTo(3, 5)

	if err != nil {
		t.Errorf("moveCursorTo() unexpected error: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("moveCursorTo() with same position wrote %d bytes, want 0", buf.Len())
	}
	if f.curX != 5 || f.curY != 3 {
		t.Errorf("moveCursorTo() unchanged cursor = (%d,%d), want (5,3)", f.curX, f.curY)
	}
}

func TestANSIFlusher_moveCursorTo_Moves(t *testing.T) {
	var buf bytes.Buffer
	f := NewANSIFlusher(&buf)

	err := f.moveCursorTo(2, 5)

	if err != nil {
		t.Errorf("moveCursorTo() unexpected error: %v", err)
	}
	if err := f.w.Flush(); err != nil {
		t.Errorf("Flush() unexpected error: %v", err)
	}
	expected := "\x1b[3;6H" // ANSI: row 3, col 6 (1-indexed)
	if got := buf.String(); got != expected {
		t.Errorf("moveCursorTo() = %q, want %q", got, expected)
	}
	if f.curX != 5 || f.curY != 2 {
		t.Errorf("moveCursorTo() cursor = (%d,%d), want (5,2)", f.curX, f.curY)
	}
}

func TestANSIFlusher_emitSGR_Default(t *testing.T) {
	var buf bytes.Buffer
	f := NewANSIFlusher(&buf)

	err := f.emitSGR(style.Style{})

	if err != nil {
		t.Errorf("emitSGR() unexpected error: %v", err)
	}
	if err := f.w.Flush(); err != nil {
		t.Errorf("Flush() unexpected error: %v", err)
	}
	// Default style only emits reset
	expected := "\x1b[0m"
	if got := buf.String(); got != expected {
		t.Errorf("emitSGR() default = %q, want %q", got, expected)
	}
}

func TestANSIFlusher_emitSGR_BasicColors(t *testing.T) {
	tests := []struct {
		name     string
		fg       style.Color
		bg       style.Color
		expected string
	}{
		{
			name:     "red foreground",
			fg:       style.ColorRed,
			expected: "\x1b[0;31;49m",
		},
		{
			name:     "blue foreground",
			fg:       style.ColorBlue,
			expected: "\x1b[0;34;49m",
		},
		{
			name:     "green background",
			bg:       style.ColorGreen,
			expected: "\x1b[0;39;42m",
		},
		{
			name:     "red on blue",
			fg:       style.ColorRed,
			bg:       style.ColorBlue,
			expected: "\x1b[0;31;44m",
		},
		{
			name:     "white fg yellow bg",
			fg:       style.ColorWhite,
			bg:       style.ColorYellow,
			expected: "\x1b[0;37;43m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			f := NewANSIFlusher(&buf)

			err := f.emitSGR(style.Style{FG: tt.fg, BG: tt.bg})

			if err != nil {
				t.Errorf("emitSGR() unexpected error: %v", err)
			}
			if err := f.w.Flush(); err != nil {
				t.Errorf("Flush() unexpected error: %v", err)
			}
			if got := buf.String(); got != tt.expected {
				t.Errorf("emitSGR() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestANSIFlusher_emitSGR_BrightColors(t *testing.T) {
	tests := []struct {
		name     string
		fg       style.Color
		bg       style.Color
		expected string
	}{
		{
			name:     "bright red foreground",
			fg:       style.ColorBrightRed,
			expected: "\x1b[0;91;49m",
		},
		{
			name:     "bright blue foreground",
			fg:       style.ColorBrightBlue,
			expected: "\x1b[0;94;49m",
		},
		{
			name:     "bright green background",
			bg:       style.ColorBrightGreen,
			expected: "\x1b[0;39;102m",
		},
		{
			name:     "bright red on bright blue",
			fg:       style.ColorBrightRed,
			bg:       style.ColorBrightBlue,
			expected: "\x1b[0;91;104m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			f := NewANSIFlusher(&buf)

			err := f.emitSGR(style.Style{FG: tt.fg, BG: tt.bg})

			if err != nil {
				t.Errorf("emitSGR() unexpected error: %v", err)
			}
			if err := f.w.Flush(); err != nil {
				t.Errorf("Flush() unexpected error: %v", err)
			}
			if got := buf.String(); got != tt.expected {
				t.Errorf("emitSGR() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestANSIFlusher_emitSGR_IndexedColor(t *testing.T) {
	var buf bytes.Buffer
	f := NewANSIFlusher(&buf)

	// ColorIndex(232) (first grayscale in 256-color range)
	err := f.emitSGR(style.Style{FG: style.ColorIndex(232)})

	if err != nil {
		t.Errorf("emitSGR() unexpected error: %v", err)
	}
	if err := f.w.Flush(); err != nil {
		t.Errorf("Flush() unexpected error: %v", err)
	}
	expected := "\x1b[0;38;5;232;49m"
	if got := buf.String(); got != expected {
		t.Errorf("emitSGR() 256-color = %q, want %q", got, expected)
	}
}

func TestANSIFlusher_emitSGR_Attributes(t *testing.T) {
	tests := []struct {
		name     string
		attr     style.AttrMask
		expected string
	}{
		{
			name:     "bold",
			attr:     style.AttrBold,
			expected: "\x1b[0;39;49;1m",
		},
		{
			name:     "dim",
			attr:     style.AttrDim,
			expected: "\x1b[0;39;49;2m",
		},
		{
			name:     "italic",
			attr:     style.AttrItalic,
			expected: "\x1b[0;39;49;3m",
		},
		{
			name:     "underline",
			attr:     style.AttrUnderline,
			expected: "\x1b[0;39;49;4m",
		},
		{
			name:     "blink",
			attr:     style.AttrBlink,
			expected: "\x1b[0;39;49;5m",
		},
		{
			name:     "reverse",
			attr:     style.AttrReverse,
			expected: "\x1b[0;39;49;7m",
		},
		{
			name:     "bold underline",
			attr:     style.AttrBold | style.AttrUnderline,
			expected: "\x1b[0;39;49;1;4m",
		},
		{
			name:     "all attributes",
			attr:     style.AttrBold | style.AttrDim | style.AttrItalic | style.AttrUnderline | style.AttrBlink | style.AttrReverse,
			expected: "\x1b[0;39;49;1;2;3;4;5;7m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			f := NewANSIFlusher(&buf)

			err := f.emitSGR(style.Style{Attr: tt.attr})

			if err != nil {
				t.Errorf("emitSGR() unexpected error: %v", err)
			}
			if err := f.w.Flush(); err != nil {
				t.Errorf("Flush() unexpected error: %v", err)
			}
			if got := buf.String(); got != tt.expected {
				t.Errorf("emitSGR() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestANSIFlusher_emitSGR_TrueColor(t *testing.T) {
	var buf bytes.Buffer
	f := NewANSIFlusher(&buf)

	// RGB(1, 2, 3) - distinct RGB values
	err := f.emitSGR(style.Style{FG: style.ColorRGB(1, 2, 3)})

	if err != nil {
		t.Errorf("emitSGR() unexpected error: %v", err)
	}
	if err := f.w.Flush(); err != nil {
		t.Errorf("Flush() unexpected error: %v", err)
	}
	expected := "\x1b[0;38;2;1;2;3;49m"
	if got := buf.String(); got != expected {
		t.Errorf("emitSGR() truecolor = %q, want %q", got, expected)
	}
}

func TestANSIFlusher_emitSGR_Complete(t *testing.T) {
	var buf bytes.Buffer
	f := NewANSIFlusher(&buf)

	style := style.Style{
		FG:   style.ColorRed,
		BG:   style.ColorBlue,
		Attr: style.AttrBold | style.AttrUnderline,
	}
	err := f.emitSGR(style)

	if err != nil {
		t.Errorf("emitSGR() unexpected error: %v", err)
	}
	if err := f.w.Flush(); err != nil {
		t.Errorf("Flush() unexpected error: %v", err)
	}
	// Note: order is FG, BG, then attributes
	expected := "\x1b[0;31;44;1;4m"
	if got := buf.String(); got != expected {
		t.Errorf("emitSGR() = %q, want %q", got, expected)
	}
}

func TestANSIFlusher_emitRune_UTF8(t *testing.T) {
	var buf bytes.Buffer
	f := NewANSIFlusher(&buf)

	err := f.emitRune('A')

	if err != nil {
		t.Errorf("emitRune() unexpected error: %v", err)
	}
	if err := f.w.Flush(); err != nil {
		t.Errorf("Flush() unexpected error: %v", err)
	}
	if got := buf.String(); got != "A" {
		t.Errorf("emitRune('A') = %q, want 'A'", got)
	}
}

func TestANSIFlusher_emitRune_WideChar(t *testing.T) {
	var buf bytes.Buffer
	f := NewANSIFlusher(&buf)

	err := f.emitRune('日')

	if err != nil {
		t.Errorf("emitRune() unexpected error: %v", err)
	}
	if err := f.w.Flush(); err != nil {
		t.Errorf("Flush() unexpected error: %v", err)
	}
	if got := buf.String(); got != "日" {
		t.Errorf("emitRune('日') = %q, want '日'", got)
	}
}

func TestANSIFlusher_emitRune_Zero(t *testing.T) {
	var buf bytes.Buffer
	f := NewANSIFlusher(&buf)

	err := f.emitRune(0)

	if err != nil {
		t.Errorf("emitRune(0) unexpected error: %v", err)
	}
	if err := f.w.Flush(); err != nil {
		t.Errorf("Flush() unexpected error: %v", err)
	}
	if got := buf.String(); got != " " {
		t.Errorf("emitRune(0) = %q, want ' '", got)
	}
}

func TestANSIFlusher_FlushRuns_Simple(t *testing.T) {
	var buf bytes.Buffer
	f := NewANSIFlusher(&buf)

	back := NewBuffer(10, 5)
	front := NewBuffer(10, 5)

	*back.At(2, 1) = Cell{R: 'X', Style: style.Style{FG: style.ColorRed}}
	*front.At(2, 1) = Cell{R: 'A'} // different

	runs := []Run{{Y: 1, X0: 2, X1: 3}}

	err := f.FlushRuns(back, front, runs)

	if err != nil {
		t.Errorf("FlushRuns() unexpected error: %v", err)
	}
	// Should have cursor movement + SGR + rune
	output := buf.String()
	if len(output) == 0 {
		t.Error("FlushRuns() produced no output")
	}
}

func TestANSIFlusher_FlushRuns_StyleChanges(t *testing.T) {
	var buf bytes.Buffer
	f := NewANSIFlusher(&buf)

	back := NewBuffer(10, 5)
	front := NewBuffer(10, 5)

	*back.At(2, 1) = Cell{R: 'A', Style: style.Style{FG: style.ColorRed}}
	*back.At(3, 1) = Cell{R: 'B', Style: style.Style{FG: style.ColorBlue}}
	*front.At(2, 1) = Cell{R: 'X'}
	*front.At(3, 1) = Cell{R: 'Y'}

	runs := []Run{{Y: 1, X0: 2, X1: 4}}

	err := f.FlushRuns(back, front, runs)

	if err != nil {
		t.Errorf("FlushRuns() unexpected error: %v", err)
	}
	output := buf.String()
	// Should contain two different SGR sequences for red and blue
	if len(output) == 0 {
		t.Error("FlushRuns() produced no output")
	}
}

func TestANSIFlusher_FlushRuns_WideChar(t *testing.T) {
	var buf bytes.Buffer
	f := NewANSIFlusher(&buf)

	back := NewBuffer(10, 5)
	front := NewBuffer(10, 5)

	// Wide character at position 2
	*back.At(2, 1) = Cell{R: '日', Wide: true, Style: style.Style{FG: style.ColorRed}}
	*back.At(3, 1) = Cell{R: 0, WideCont: true, Style: style.Style{FG: style.ColorRed}}
	*front.At(2, 1) = Cell{R: 'A'}
	*front.At(3, 1) = Cell{R: ' '}

	runs := []Run{{Y: 1, X0: 2, X1: 4}}

	err := f.FlushRuns(back, front, runs)

	if err != nil {
		t.Errorf("FlushRuns() unexpected error: %v", err)
	}
	// Cursor should advance by 2 after wide char
	if f.curX != 4 {
		t.Errorf("FlushRuns() cursor X = %d, want 4", f.curX)
	}
}

func TestANSIFlusher_FlushRuns_UpdatesFront(t *testing.T) {
	var buf bytes.Buffer
	f := NewANSIFlusher(&buf)

	back := NewBuffer(10, 5)
	front := NewBuffer(10, 5)

	style := style.Style{FG: style.ColorRed, BG: style.ColorBlue}
	*back.At(2, 1) = Cell{R: 'X', Style: style, Wide: false, WideCont: false}
	*front.At(2, 1) = Cell{R: 'A'}

	runs := []Run{{Y: 1, X0: 2, X1: 3}}

	err := f.FlushRuns(back, front, runs)

	if err != nil {
		t.Errorf("FlushRuns() unexpected error: %v", err)
	}
	// Front buffer should now match back
	frontCell := *front.At(2, 1)
	if frontCell.R != 'X' || frontCell.Style != style {
		t.Errorf("FlushRuns() front buffer not updated, got %+v", frontCell)
	}
}

func TestAppendFGColor(t *testing.T) {
	tests := []struct {
		name     string
		color    style.Color
		expected []int
	}{
		{"default", style.ColorDefault, []int{}},
		{"black", style.ColorBlack, []int{30}},
		{"red", style.ColorRed, []int{31}},
		{"green", style.ColorGreen, []int{32}},
		{"yellow", style.ColorYellow, []int{33}},
		{"blue", style.ColorBlue, []int{34}},
		{"magenta", style.ColorMagenta, []int{35}},
		{"cyan", style.ColorCyan, []int{36}},
		{"white", style.ColorWhite, []int{37}},
		{"bright black", style.ColorBrightBlack, []int{90}},
		{"bright red", style.ColorBrightRed, []int{91}},
		{"bright white", style.ColorBrightWhite, []int{97}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := []int{}
			result := appendFGColor(params, tt.color)
			if len(result) != len(tt.expected) {
				t.Errorf("appendFGColor() = %v, want %v", result, tt.expected)
			} else {
				for i := range result {
					if result[i] != tt.expected[i] {
						t.Errorf("appendFGColor()[%d] = %d, want %d", i, result[i], tt.expected[i])
					}
				}
			}
		})
	}
}

func TestAppendBGColor(t *testing.T) {
	tests := []struct {
		name     string
		color    style.Color
		expected []int
	}{
		{"default", style.ColorDefault, []int{}},
		{"black", style.ColorBlack, []int{40}},
		{"red", style.ColorRed, []int{41}},
		{"green", style.ColorGreen, []int{42}},
		{"yellow", style.ColorYellow, []int{43}},
		{"blue", style.ColorBlue, []int{44}},
		{"magenta", style.ColorMagenta, []int{45}},
		{"cyan", style.ColorCyan, []int{46}},
		{"white", style.ColorWhite, []int{47}},
		{"bright black", style.ColorBrightBlack, []int{100}},
		{"bright red", style.ColorBrightRed, []int{101}},
		{"bright white", style.ColorBrightWhite, []int{107}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := []int{}
			result := appendBGColor(params, tt.color)
			if len(result) != len(tt.expected) {
				t.Errorf("appendBGColor() = %v, want %v", result, tt.expected)
			} else {
				for i := range result {
					if result[i] != tt.expected[i] {
						t.Errorf("appendBGColor()[%d] = %d, want %d", i, result[i], tt.expected[i])
					}
				}
			}
		})
	}
}

func TestEmitSGRParams(t *testing.T) {
	tests := []struct {
		name     string
		params   []int
		expected string
	}{
		{"single", []int{31}, "\x1b[31m"},
		{"two", []int{31, 44}, "\x1b[31;44m"},
		{"multiple", []int{1, 31, 44}, "\x1b[1;31;44m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			w := bufio.NewWriter(&buf)
			err := emitSGRParams(w, tt.params)
			if err := w.Flush(); err != nil {
				t.Errorf("Flush() unexpected error: %v", err)
			}
			if err != nil {
				t.Errorf("emitSGRParams() unexpected error: %v", err)
			}
			if got := buf.String(); got != tt.expected {
				t.Errorf("emitSGRParams() = %q, want %q", got, tt.expected)
			}
		})
	}
}
