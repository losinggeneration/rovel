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
	emoji := []rune{'😀', '😂', '🎉', '🌈'}
	for _, r := range emoji {
		if got := RuneWidth(r); got != 2 {
			t.Errorf("RuneWidth(%c) = %d, want 2 (emoji)", r, got)
		}
	}
}

func TestRuneWidth_RegionalIndicator(t *testing.T) {
	// Regional Indicator symbols are treated as narrow so flag pairs occupy 2 cells.
	if got := RuneWidth('🇺'); got != 1 {
		t.Errorf("RuneWidth(🇺) = %d, want 1", got)
	}
	if got := RuneWidth('🇸'); got != 1 {
		t.Errorf("RuneWidth(🇸) = %d, want 1", got)
	}
}
