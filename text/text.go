package text

import (
	"unicode/utf8"

	"github.com/losinggeneration/tui/render"
)

func Width(s string) int {
	width := 0
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		width += render.RuneWidth(r)
		i += size
	}
	return width
}

func WidthRune(r rune) int {
	return render.RuneWidth(r)
}

func FitPrefix(s string, maxCols int) (endByte int, cols int, clipped bool) {
	if maxCols <= 0 {
		return 0, 0, len(s) > 0
	}

	width := 0
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		w := render.RuneWidth(r)

		if width+w > maxCols {
			if width == 0 {
				return 0, 0, len(s) > 0
			}
			return i, width, true
		}

		width += w
		i += size
	}

	return len(s), width, false
}

func Truncate(s string, maxCols int, ellipsis bool) string {
	if maxCols <= 0 {
		return ""
	}

	ellipsisWidth := 0
	if ellipsis {
		ellipsisWidth = 1
		if maxCols < ellipsisWidth {
			return ""
		}
	}

	endByte, _, clipped := FitPrefix(s, maxCols-ellipsisWidth)
	if !clipped {
		return s[:endByte]
	}

	if ellipsis {
		return s[:endByte] + "…"
	}
	return s[:endByte]
}
