package render

import (
	"bufio"
	"bytes"
	"testing"

	"github.com/losinggeneration/tui"
)

func TestANSIFlusher_New(t *testing.T) {
	var buf bytes.Buffer
	f := NewANSIFlusher(&buf)

	if f.W == nil {
		t.Error("NewANSIFlusher() W is nil")
	}
	if f.CurX != 0 || f.CurY != 0 {
		t.Errorf("NewANSIFlusher() cursor = (%d,%d), want (0,0)", f.CurX, f.CurY)
	}
	if f.HasStyle {
		t.Error("NewANSIFlusher() HasStyle should be false")
	}
}

func TestANSIFlusher_moveCursorTo_NoOp(t *testing.T) {
	var buf bytes.Buffer
	f := NewANSIFlusher(&buf)
	f.CurX = 5
	f.CurY = 3

	err := f.moveCursorTo(3, 5)

	if err != nil {
		t.Errorf("moveCursorTo() unexpected error: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("moveCursorTo() with same position wrote %d bytes, want 0", buf.Len())
	}
	if f.CurX != 5 || f.CurY != 3 {
		t.Errorf("moveCursorTo() unchanged cursor = (%d,%d), want (5,3)", f.CurX, f.CurY)
	}
}

func TestANSIFlusher_moveCursorTo_Moves(t *testing.T) {
	var buf bytes.Buffer
	f := NewANSIFlusher(&buf)

	err := f.moveCursorTo(2, 5)

	if err != nil {
		t.Errorf("moveCursorTo() unexpected error: %v", err)
	}
	f.W.Flush()
	expected := "\x1b[3;6H" // ANSI: row 3, col 6 (1-indexed)
	if got := buf.String(); got != expected {
		t.Errorf("moveCursorTo() = %q, want %q", got, expected)
	}
	if f.CurX != 5 || f.CurY != 2 {
		t.Errorf("moveCursorTo() cursor = (%d,%d), want (5,2)", f.CurX, f.CurY)
	}
}

func TestANSIFlusher_emitSGR_Default(t *testing.T) {
	var buf bytes.Buffer
	f := NewANSIFlusher(&buf)

	err := f.emitSGR(tui.Style{})

	if err != nil {
		t.Errorf("emitSGR() unexpected error: %v", err)
	}
	f.W.Flush()
	// Default style only emits reset
	expected := "\x1b[0m"
	if got := buf.String(); got != expected {
		t.Errorf("emitSGR() default = %q, want %q", got, expected)
	}
}

func TestANSIFlusher_emitSGR_BasicColors(t *testing.T) {
	tests := []struct {
		name     string
		fg       tui.Color
		bg       tui.Color
		expected string
	}{
		{
			name:     "red foreground",
			fg:       tui.ColorRed,
			expected: "\x1b[0;31m",
		},
		{
			name:     "blue foreground",
			fg:       tui.ColorBlue,
			expected: "\x1b[0;34m",
		},
		{
			name:     "green background",
			bg:       tui.ColorGreen,
			expected: "\x1b[0;42m",
		},
		{
			name:     "red on blue",
			fg:       tui.ColorRed,
			bg:       tui.ColorBlue,
			expected: "\x1b[0;31;44m",
		},
		{
			name:     "white fg yellow bg",
			fg:       tui.ColorWhite,
			bg:       tui.ColorYellow,
			expected: "\x1b[0;37;43m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			f := NewANSIFlusher(&buf)

			err := f.emitSGR(tui.Style{FG: tt.fg, BG: tt.bg})

			if err != nil {
				t.Errorf("emitSGR() unexpected error: %v", err)
			}
			f.W.Flush()
			if got := buf.String(); got != tt.expected {
				t.Errorf("emitSGR() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestANSIFlusher_emitSGR_BrightColors(t *testing.T) {
	tests := []struct {
		name     string
		fg       tui.Color
		bg       tui.Color
		expected string
	}{
		{
			name:     "bright red foreground",
			fg:       tui.ColorBrightRed,
			expected: "\x1b[0;91m",
		},
		{
			name:     "bright blue foreground",
			fg:       tui.ColorBrightBlue,
			expected: "\x1b[0;94m",
		},
		{
			name:     "bright green background",
			bg:       tui.ColorBrightGreen,
			expected: "\x1b[0;102m",
		},
		{
			name:     "bright red on bright blue",
			fg:       tui.ColorBrightRed,
			bg:       tui.ColorBrightBlue,
			expected: "\x1b[0;91;104m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			f := NewANSIFlusher(&buf)

			err := f.emitSGR(tui.Style{FG: tt.fg, BG: tt.bg})

			if err != nil {
				t.Errorf("emitSGR() unexpected error: %v", err)
			}
			f.W.Flush()
			if got := buf.String(); got != tt.expected {
				t.Errorf("emitSGR() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestANSIFlusher_emitSGR_256Color(t *testing.T) {
	var buf bytes.Buffer
	f := NewANSIFlusher(&buf)

	// Color 232 (first of 256-color range)
	err := f.emitSGR(tui.Style{FG: 232})

	if err != nil {
		t.Errorf("emitSGR() unexpected error: %v", err)
	}
	f.W.Flush()
	expected := "\x1b[0;38;5;232m"
	if got := buf.String(); got != expected {
		t.Errorf("emitSGR() 256-color = %q, want %q", got, expected)
	}
}

func TestANSIFlusher_emitSGR_Attributes(t *testing.T) {
	tests := []struct {
		name     string
		attr     tui.AttrMask
		expected string
	}{
		{
			name:     "bold",
			attr:     tui.AttrBold,
			expected: "\x1b[0;1m",
		},
		{
			name:     "dim",
			attr:     tui.AttrDim,
			expected: "\x1b[0;2m",
		},
		{
			name:     "italic",
			attr:     tui.AttrItalic,
			expected: "\x1b[0;3m",
		},
		{
			name:     "underline",
			attr:     tui.AttrUnderline,
			expected: "\x1b[0;4m",
		},
		{
			name:     "blink",
			attr:     tui.AttrBlink,
			expected: "\x1b[0;5m",
		},
		{
			name:     "reverse",
			attr:     tui.AttrReverse,
			expected: "\x1b[0;7m",
		},
		{
			name:     "bold underline",
			attr:     tui.AttrBold | tui.AttrUnderline,
			expected: "\x1b[0;1;4m",
		},
		{
			name:     "all attributes",
			attr:     tui.AttrBold | tui.AttrDim | tui.AttrItalic | tui.AttrUnderline | tui.AttrBlink | tui.AttrReverse,
			expected: "\x1b[0;1;2;3;4;5;7m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			f := NewANSIFlusher(&buf)

			err := f.emitSGR(tui.Style{Attr: tt.attr})

			if err != nil {
				t.Errorf("emitSGR() unexpected error: %v", err)
			}
			f.W.Flush()
			if got := buf.String(); got != tt.expected {
				t.Errorf("emitSGR() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestANSIFlusher_emitSGR_TrueColor(t *testing.T) {
	t.Skip("truecolor requires larger Color type - uint16 cannot hold full RGB")
}

func TestANSIFlusher_emitSGR_Complete(t *testing.T) {
	var buf bytes.Buffer
	f := NewANSIFlusher(&buf)

	style := tui.Style{
		FG:   tui.ColorRed,
		BG:   tui.ColorBlue,
		Attr: tui.AttrBold | tui.AttrUnderline,
	}
	err := f.emitSGR(style)

	if err != nil {
		t.Errorf("emitSGR() unexpected error: %v", err)
	}
	f.W.Flush()
	// Note: order may vary (FG, BG, then attributes)
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
	f.W.Flush()
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
	f.W.Flush()
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
	f.W.Flush()
	if got := buf.String(); got != " " {
		t.Errorf("emitRune(0) = %q, want ' '", got)
	}
}

func TestANSIFlusher_FlushRuns_Simple(t *testing.T) {
	var buf bytes.Buffer
	f := NewANSIFlusher(&buf)

	back := NewBuffer(10, 5)
	front := NewBuffer(10, 5)

	*back.At(2, 1) = Cell{R: 'X', Style: tui.Style{FG: tui.ColorRed}}
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

	*back.At(2, 1) = Cell{R: 'A', Style: tui.Style{FG: tui.ColorRed}}
	*back.At(3, 1) = Cell{R: 'B', Style: tui.Style{FG: tui.ColorBlue}}
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
	*back.At(2, 1) = Cell{R: '日', Wide: true, Style: tui.Style{FG: tui.ColorRed}}
	*back.At(3, 1) = Cell{R: 0, WideCont: true, Style: tui.Style{FG: tui.ColorRed}}
	*front.At(2, 1) = Cell{R: 'A'}
	*front.At(3, 1) = Cell{R: ' '}

	runs := []Run{{Y: 1, X0: 2, X1: 4}}

	err := f.FlushRuns(back, front, runs)

	if err != nil {
		t.Errorf("FlushRuns() unexpected error: %v", err)
	}
	// Cursor should advance by 2 after wide char
	if f.CurX != 4 {
		t.Errorf("FlushRuns() cursor X = %d, want 4", f.CurX)
	}
}

func TestANSIFlusher_FlushRuns_UpdatesFront(t *testing.T) {
	var buf bytes.Buffer
	f := NewANSIFlusher(&buf)

	back := NewBuffer(10, 5)
	front := NewBuffer(10, 5)

	style := tui.Style{FG: tui.ColorRed, BG: tui.ColorBlue}
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
		color    tui.Color
		expected []int
	}{
		{"default", tui.ColorDefault, []int{}},
		{"black", tui.ColorBlack, []int{30}},
		{"red", tui.ColorRed, []int{31}},
		{"green", tui.ColorGreen, []int{32}},
		{"yellow", tui.ColorYellow, []int{33}},
		{"blue", tui.ColorBlue, []int{34}},
		{"magenta", tui.ColorMagenta, []int{35}},
		{"cyan", tui.ColorCyan, []int{36}},
		{"white", tui.ColorWhite, []int{37}},
		{"bright black", tui.ColorBrightBlack, []int{90}},
		{"bright red", tui.ColorBrightRed, []int{91}},
		{"bright white", tui.ColorBrightWhite, []int{97}},
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
		color    tui.Color
		expected []int
	}{
		{"default", tui.ColorDefault, []int{}},
		{"black", tui.ColorBlack, []int{40}},
		{"red", tui.ColorRed, []int{41}},
		{"green", tui.ColorGreen, []int{42}},
		{"yellow", tui.ColorYellow, []int{43}},
		{"blue", tui.ColorBlue, []int{44}},
		{"magenta", tui.ColorMagenta, []int{45}},
		{"cyan", tui.ColorCyan, []int{46}},
		{"white", tui.ColorWhite, []int{47}},
		{"bright black", tui.ColorBrightBlack, []int{100}},
		{"bright red", tui.ColorBrightRed, []int{101}},
		{"bright white", tui.ColorBrightWhite, []int{107}},
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
			w.Flush()
			if err != nil {
				t.Errorf("emitSGRParams() unexpected error: %v", err)
			}
			if got := buf.String(); got != tt.expected {
				t.Errorf("emitSGRParams() = %q, want %q", got, tt.expected)
			}
		})
	}
}
