package text

import (
	cellwidth "github.com/losinggeneration/tui/internal/width"
)

func Width(s string) int {
	width := 0

	for _, r := range s {
		width += cellwidth.RuneWidth(r)
	}

	return width
}

func WidthRune(r rune) int {
	return cellwidth.RuneWidth(r)
}

func FitPrefix(s string, maxCols int) (endByte int, cols int, clipped bool) {
	if maxCols <= 0 {
		return 0, 0, len(s) > 0
	}

	width := 0

	i := 0
	for i < len(s) {
		next := NextCluster(s, i)
		if next <= i {
			break
		}

		clusterW := WidthBetween(s, i, next)
		if width == 0 && clusterW > maxCols {
			return 0, 0, true
		}

		if width+clusterW > maxCols {
			return i, width, true
		}

		width += clusterW
		i = next
	}

	return i, width, i < len(s)
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
