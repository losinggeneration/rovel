package render

// RuneWidth returns the terminal column width of a rune (1 or 2).
// MVP: simple CJK range detection.
// Post-MVP: use internal/wcwidth for proper Unicode width.
func RuneWidth(r rune) int {
	// Control characters and NULL have width 0
	if r < 32 {
		return 0
	}

	// DEL is width 0
	if r == 127 {
		return 0
	}

	// CJK Unified Ideographs blocks
	// U+4E00–U+9FFF: CJK Unified Ideographs
	// U+3400–U+4DBF: CJK Unified Ideographs Extension A
	// U+20000–U+2A6DF: CJK Unified Ideographs Extension B (requires surrogate pair handling)
	// U+2A700–U+2B73F: CJK Unified Ideographs Extension C
	// U+2B740–U+2B81F: CJK Unified Ideographs Extension D
	// U+2B820–U+2CEAF: CJK Unified Ideographs Extension E
	// U+2CEB0–U+2EBEF: CJK Unified Ideographs Extension F
	// U+30000–U+3134F: CJK Unified Ideographs Extension G
	//
	// CJK Compatibility Ideographs
	// U+F900–U+FAFF
	//
	// Hangul Syllables
	// U+AC00–U+D7AF
	//
	// CJK Radicals Supplement
	// U+2E80–U+2EFF
	//
	// Kangxi Radicals
	// U+2F00–U+2FDF
	//
	// Ideographic Description Characters
	// U+2FF0–U+2FFF
	//
	// CJK Symbols and Punctuation
	// U+3000–U+303F
	//
	// Hiragana
	// U+3040–U+309F
	//
	// Katakana
	// U+30A0–U+30FF
	//
	// Bopomofo
	// U+3100–U+312F
	//
	// Bopomofo Extended
	// U+31A0–U+31BF
	//
	// Hangul Compatibility Jamo
	// U+3130–U+318F
	//
	// Kanbun
	// U+3190–U+319F
	//
	// Bopomofo Extended
	// U+31A0–U+31BF
	//
	// CJK Strokes
	// U+31C0–U+31EF
	//
	// Katakana Phonetic Extensions
	// U+31F0–U+31FF
	//
	// Enclosed CJK Letters and Months
	// U+3200–U+32FF
	//
	// CJK Compatibility
	// U+3300–U+33FF
	//
	// CJK Unified Ideographs Extension A
	// U+3400–U+4DBF
	//
	// Yijing Hexagram Symbols
	// U+4DC0–U+4DFF
	//
	// CJK Unified Ideographs
	// U+4E00–U+9FFF
	//
	// Yi Syllables
	// U+A000–U+A48F
	//
	// Yi Radicals
	// U+A490–U+A4CF
	//
	// Hangul Jamo Extended-B
	// U+A960–U+A97F
	//
	// Hangul Syllables
	// U+AC00–U+D7AF
	//
	// CJK Compatibility Ideographs
	// U+F900–U+FAFF
	//
	// CJK Alphabets (includes halfwidth forms)
	// U+FF00–U+FFEF

	// Check common CJK ranges
	if (r >= 0x1100 && r <= 0x115F) || // Hangul Jamo
		(r >= 0x2E80 && r <= 0x2EFF) || // CJK Radicals Supplement
		(r >= 0x2F00 && r <= 0x2FDF) || // Kangxi Radicals
		(r >= 0x2FF0 && r <= 0x2FFF) || // Ideographic Description Characters
		(r >= 0x3000 && r <= 0x303F) || // CJK Symbols and Punctuation
		(r >= 0x3040 && r <= 0x309F) || // Hiragana
		(r >= 0x30A0 && r <= 0x30FF) || // Katakana
		(r >= 0x3100 && r <= 0x312F) || // Bopomofo
		(r >= 0x3130 && r <= 0x318F) || // Hangul Compatibility Jamo
		(r >= 0x31A0 && r <= 0x31BF) || // Bopomofo Extended
		(r >= 0x31C0 && r <= 0x31EF) || // CJK Strokes
		(r >= 0x31F0 && r <= 0x31FF) || // Katakana Phonetic Extensions
		(r >= 0x3200 && r <= 0x32FF) || // Enclosed CJK Letters and Months
		(r >= 0x3300 && r <= 0x33FF) || // CJK Compatibility
		(r >= 0x3400 && r <= 0x4DBF) || // CJK Unified Ideographs Extension A
		(r >= 0x4E00 && r <= 0x9FFF) || // CJK Unified Ideographs
		(r >= 0xAC00 && r <= 0xD7AF) || // Hangul Syllables
		(r >= 0xF900 && r <= 0xFAFF) || // CJK Compatibility Ideographs
		(r >= 0xFE30 && r <= 0xFE4F) || // CJK Compatibility Forms
		(r >= 0xFF00 && r <= 0xFF60) || // Fullwidth Forms
		(r >= 0xFFE0 && r <= 0xFFE6) {
		return 2
	}

	// Default width for most characters
	return 1
}
