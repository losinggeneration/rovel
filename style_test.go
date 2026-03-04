package tui

import "testing"

func TestStyle_Equals(t *testing.T) {
	tests := []struct {
		name     string
		s        Style
		other    Style
		expected bool
	}{
		{
			name:     "identical styles",
			s:        Style{FG: ColorRed, BG: ColorBlue, Attr: AttrBold},
			other:    Style{FG: ColorRed, BG: ColorBlue, Attr: AttrBold},
			expected: true,
		},
		{
			name:     "different foreground",
			s:        Style{FG: ColorRed, BG: ColorBlue, Attr: AttrBold},
			other:    Style{FG: ColorGreen, BG: ColorBlue, Attr: AttrBold},
			expected: false,
		},
		{
			name:     "different background",
			s:        Style{FG: ColorRed, BG: ColorBlue, Attr: AttrBold},
			other:    Style{FG: ColorRed, BG: ColorGreen, Attr: AttrBold},
			expected: false,
		},
		{
			name:     "different attributes",
			s:        Style{FG: ColorRed, BG: ColorBlue, Attr: AttrBold},
			other:    Style{FG: ColorRed, BG: ColorBlue, Attr: AttrDim},
			expected: false,
		},
		{
			name:     "zero styles",
			s:        Style{},
			other:    Style{},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.Equals(tt.other); got != tt.expected {
				t.Errorf("Style.Equals() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestColor_Constants(t *testing.T) {
	// Verify color constants are distinct
	colors := []Color{
		ColorDefault,
		ColorBlack,
		ColorRed,
		ColorGreen,
		ColorYellow,
		ColorBlue,
		ColorMagenta,
		ColorCyan,
		ColorWhite,
		ColorBrightBlack,
		ColorBrightRed,
		ColorBrightGreen,
		ColorBrightYellow,
		ColorBrightBlue,
		ColorBrightMagenta,
		ColorBrightCyan,
		ColorBrightWhite,
	}

	seen := make(map[Color]bool)
	for _, c := range colors {
		if seen[c] {
			t.Errorf("Duplicate color value: %v", c)
		}
		seen[c] = true
	}
}

func TestAttrMask_Constants(t *testing.T) {
	// Verify each attribute is a distinct bit
	attrs := []AttrMask{
		AttrBold,
		AttrDim,
		AttrItalic,
		AttrUnderline,
		AttrBlink,
		AttrReverse,
	}

	for _, attr := range attrs {
		if attr == 0 {
			t.Errorf("Attribute %v is zero", attr)
		}
		// Check it's a single bit
		if attr&(attr-1) != 0 {
			t.Errorf("Attribute %v is not a single bit", attr)
		}
	}

	// Verify no overlapping bits
	seen := make(map[AttrMask]bool)
	for _, attr := range attrs {
		if seen[attr] {
			t.Errorf("Duplicate attribute bit: %v", attr)
		}
		seen[attr] = true
	}
}
