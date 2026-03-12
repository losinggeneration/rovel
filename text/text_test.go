package text

import (
	"testing"
)

func TestWidth(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"", 0},
		{"hello", 5},
		{"Hello, World!", 13},
		{"日本語", 6},
		{"a日b", 4},
		{"\x1b[31mred\x1b[0m", 3 + 3 + 4},
	}

	for _, tt := range tests {
		got := Width(tt.input)
		if got != tt.expected {
			t.Errorf("Width(%q) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

func TestFitPrefix(t *testing.T) {
	tests := []struct {
		input       string
		maxCols     int
		wantEnd     int
		wantCols    int
		wantClipped bool
	}{
		{"hello", 10, 5, 5, false},
		{"hello", 3, 3, 3, true},
		{"日本語test", 5, 6, 4, true},
		{"", 5, 0, 0, false},
		{"a", 0, 0, 0, true},
		{"日本", 1, 0, 0, true},
		{"日", 2, 3, 2, false},
	}

	for _, tt := range tests {
		end, cols, clipped := FitPrefix(tt.input, tt.maxCols)
		if end != tt.wantEnd || cols != tt.wantCols || clipped != tt.wantClipped {
			t.Errorf("FitPrefix(%q, %d) = (%d, %d, %v), want (%d, %d, %v)",
				tt.input, tt.maxCols, end, cols, clipped,
				tt.wantEnd, tt.wantCols, tt.wantClipped)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxCols  int
		ellipsis bool
		expected string
	}{
		{"hello", 10, false, "hello"},
		{"hello world", 5, false, "hello"},
		{"hello world", 8, true, "hello w…"},
		{"hello world", 5, true, "hell…"},
		{"日本語", 4, true, "日…"},
		{"日本語", 4, false, "日本"},
		{"", 5, false, ""},
		{"a", 0, false, ""},
	}

	for _, tt := range tests {
		got := Truncate(tt.input, tt.maxCols, tt.ellipsis)
		if got != tt.expected {
			t.Errorf("Truncate(%q, %d, %v) = %q, want %q",
				tt.input, tt.maxCols, tt.ellipsis, got, tt.expected)
		}
	}
}

func TestNextCluster(t *testing.T) {
	tests := []struct {
		input string
		pos   int
		want  int
	}{
		{"hello", 0, 1},
		{"hello", 4, 5},
		{"hello", 5, 5},
		{"日abc", 0, 3},
		{"日abc", 3, 4},
		{"", 0, 0},
		{"a", 10, 1},
	}

	for _, tt := range tests {
		got := NextCluster(tt.input, tt.pos)
		if got != tt.want {
			t.Errorf("NextCluster(%q, %d) = %d, want %d", tt.input, tt.pos, got, tt.want)
		}
	}
}

func TestPrevCluster(t *testing.T) {
	tests := []struct {
		input string
		pos   int
		want  int
	}{
		{"hello", 5, 4},
		{"hello", 1, 0},
		{"hello", 0, 0},
		{"日abc", 3, 0},
		{"日abc", 4, 3},
		{"", 0, 0},
		{"a", 10, 0},
	}

	for _, tt := range tests {
		got := PrevCluster(tt.input, tt.pos)
		if got != tt.want {
			t.Errorf("PrevCluster(%q, %d) = %d, want %d", tt.input, tt.pos, got, tt.want)
		}
	}
}

func TestColumnOf(t *testing.T) {
	tests := []struct {
		input   string
		byteOff int
		want    int
	}{
		{"hello", 0, 0},
		{"hello", 3, 3},
		{"hello", 5, 5},
		{"日abc", 0, 0},
		{"日abc", 3, 2},
		{"日abc", 4, 3},
		{"", 0, 0},
		{"a", 10, 1},
	}

	for _, tt := range tests {
		got := ColumnOf(tt.input, tt.byteOff)
		if got != tt.want {
			t.Errorf("ColumnOf(%q, %d) = %d, want %d", tt.input, tt.byteOff, got, tt.want)
		}
	}
}

func TestOffsetAtColumn(t *testing.T) {
	tests := []struct {
		input string
		col   int
		want  int
	}{
		{"hello", 0, 0},
		{"hello", 3, 3},
		{"hello", 10, 5},
		{"日abc", 0, 0},
		{"日abc", 2, 3},
		{"日abc", 3, 4},
		{"", 5, 0},
	}

	for _, tt := range tests {
		got := OffsetAtColumn(tt.input, tt.col)
		if got != tt.want {
			t.Errorf("OffsetAtColumn(%q, %d) = %d, want %d", tt.input, tt.col, got, tt.want)
		}
	}
}

func TestWrap(t *testing.T) {
	tests := []struct {
		input  string
		width  int
		expect []Line
	}{
		{"", 10, nil},
		{"hello", 10, []Line{{Start: 0, End: 5, Width: 5}}},
		{"hello world", 5, []Line{
			{Start: 0, End: 5, Width: 5},
			{Start: 5, End: 10, Width: 5},
			{Start: 10, End: 11, Width: 1},
		}},
		{"a\nb\nc", 10, []Line{
			{Start: 0, End: 1, Width: 1},
			{Start: 2, End: 3, Width: 1},
			{Start: 4, End: 5, Width: 1},
		}},
		{"日日日", 4, []Line{
			{Start: 0, End: 6, Width: 4},
			{Start: 6, End: 9, Width: 2},
		}},
	}

	for _, tt := range tests {
		got := Wrap(tt.input, tt.width)
		if len(got) != len(tt.expect) {
			t.Errorf("Wrap(%q, %d) returned %d lines, want %d", tt.input, tt.width, len(got), len(tt.expect))
			continue
		}
		for i, line := range got {
			if line != tt.expect[i] {
				t.Errorf("Wrap(%q, %d)[%d] = %+v, want %+v", tt.input, tt.width, i, line, tt.expect[i])
			}
		}
	}
}
