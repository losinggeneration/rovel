package width

import runewidth "github.com/mattn/go-runewidth"

// RuneWidth returns the terminal column width of a rune (0, 1, or 2).
//
// This is a shared implementation detail used by multiple public packages to
// avoid import cycles. Public text-measurement semantics are documented in
// package text.
func RuneWidth(r rune) int {
	// Control characters and NULL have width 0.
	if r < 32 {
		return 0
	}

	// DEL is width 0.
	if r == 127 {
		return 0
	}

	// Regional Indicator symbols combine into flag grapheme clusters in many
	// terminals. Treat each code point as width 1 so the pair occupies 2 cells.
	//
	// This intentionally favors practical terminal behavior over strict
	// codepoint-based width tables.
	if r >= 0x1F1E6 && r <= 0x1F1FF {
		return 1
	}

	w := runewidth.RuneWidth(r)
	if w < 0 {
		w = 0
	}

	if w > 2 {
		w = 2
	}

	return w
}
