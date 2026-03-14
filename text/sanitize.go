package text

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// Sanitize applies a conservative policy to make text behave predictably with a
// cell-based renderer and a single-rune-per-cell paint API.
//
// Policy:
//   - normalize to NFC
//   - drop remaining combining marks (Mn/Me)
//   - drop joiners and variation selectors (ZWJ/ZWNJ, VS-1..VS-16, and IVS)
func Sanitize(s string) string {
	if s == "" {
		return ""
	}

	s = norm.NFC.String(s)

	var b strings.Builder
	b.Grow(len(s))

	for _, r := range s {
		if unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Me, r) {
			continue
		}
		// ZWJ / ZWNJ
		if r == 0x200D || r == 0x200C {
			continue
		}
		// Variation Selectors (VS1..VS16)
		if r >= 0xFE00 && r <= 0xFE0F {
			continue
		}
		// Ideographic Variation Sequences (IVS)
		if r >= 0xE0100 && r <= 0xE01EF {
			continue
		}
		b.WriteRune(r)
	}

	return b.String()
}
