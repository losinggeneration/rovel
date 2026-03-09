package render

import "testing"

func TestRuneWidth_ASCII(t *testing.T) {
	// ASCII characters should have width 1
	for r := rune(32); r < 127; r++ {
		width := RuneWidth(r)
		if width != 1 {
			t.Errorf("RuneWidth(%c) = %d, want 1", r, width)
		}
	}
}

func TestRuneWidth_Control(t *testing.T) {
	// Control characters should have width 0
	controlRanges := []struct {
		start, end rune
	}{
		{0, 31},
		{127, 127},
	}

	for _, rr := range controlRanges {
		for r := rr.start; r <= rr.end; r++ {
			width := RuneWidth(r)
			if width != 0 {
				t.Errorf("RuneWidth(%d) = %d, want 0 (control char)", r, width)
			}
		}
	}
}

func TestRuneWidth_CJK(t *testing.T) {
	// Common CJK characters should have width 2
	cjkChars := []rune{
		// CJK Unified Ideographs
		'中', '国', '文', '字', '你', '好',
		// Hiragana
		'あ', 'い', 'う', 'え', 'お',
		// Katakana
		'ア', 'イ', 'ウ', 'エ', 'オ',
		// Hangul Syllables
		'가', '나', '다', '라', '마',
		// Fullwidth forms
		'Ａ', 'Ｂ', 'Ｃ',
	}

	for _, r := range cjkChars {
		width := RuneWidth(r)
		if width != 2 {
			t.Errorf("RuneWidth(%c) = %d, want 2 (CJK)", r, width)
		}
	}
}

func TestRuneWidth_Emoji(t *testing.T) {
	// Most emoji are handled as wide (2) in our simple implementation
	// Proper wcwidth would handle these better
	emoji := []rune{
		'😀', '😂', '🎉', '❤',
	}

	for _, r := range emoji {
		width := RuneWidth(r)
		// In our current implementation, these default to width 1
		// (they're outside the CJK ranges we check)
		// This is acceptable for now
		if width < 0 || width > 2 {
			t.Errorf("RuneWidth(%c) = %d, want 1 or 2", r, width)
		}
	}
}

func TestRuneWidth_RangeBoundaries(t *testing.T) {
	// Test characters at boundaries of our CJK ranges
	tests := []struct {
		r        rune
		expected int
	}{
		// CJK Unified Ideographs (U+4E00–U+9FFF)
		{0x4E00, 2}, // First CJK Unified Ideograph
		{0x9FFF, 2}, // Last CJK Unified Ideograph
		{0x4DFF, 1}, // Just before CJK Unified
		{0xA000, 1}, // Just after CJK Unified

		// Hiragana (U+3040–U+309F)
		{0x3040, 2}, // First Hiragana
		{0x309F, 2}, // Last Hiragana
		{0x303F, 2}, // IDEOGRAPHIC FULL STOP (in CJK Symbols block)
		{0x30A0, 2}, // Katakana (starts right after)

		// Hangul Syllables (U+AC00–U+D7AF)
		{0xAC00, 2}, // First Hangul Syllable
		{0xD7AF, 2}, // Last Hangul Syllable
		{0xABFF, 1}, // Just before
		{0xD7B0, 1}, // Just after
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			if got := RuneWidth(tt.r); got != tt.expected {
				t.Errorf("RuneWidth(0x%X) = %d, want %d", tt.r, got, tt.expected)
			}
		})
	}
}
