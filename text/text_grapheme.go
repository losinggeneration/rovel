package text

import (
	"unicode/utf8"

	cellwidth "github.com/losinggeneration/tui/internal/width"
)

func NextCluster(s string, i int) int {
	if i >= len(s) {
		return len(s)
	}
	_, size := utf8.DecodeRuneInString(s[i:])
	if size == 0 {
		return len(s)
	}
	return i + size
}

func PrevCluster(s string, i int) int {
	if i <= 0 {
		return 0
	}

	if i > len(s) {
		i = len(s)
	}

	_, size := utf8.DecodeLastRuneInString(s[:i])
	if size == 0 {
		return 0
	}

	i -= size
	if i > 0 {
		return i
	}

	return 0
}

func ColumnOf(s string, byteOff int) int {
	if byteOff <= 0 {
		return 0
	}
	if byteOff > len(s) {
		byteOff = len(s)
	}

	width := 0
	for i := 0; i < byteOff; {
		r, size := utf8.DecodeRuneInString(s[i:])
		width += cellwidth.RuneWidth(r)
		i += size
	}
	return width
}

func OffsetAtColumn(s string, col int) int {
	if col <= 0 || len(s) == 0 {
		return 0
	}

	width := 0
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		w := cellwidth.RuneWidth(r)

		if width+w > col {
			return i
		}

		width += w
		i += size
	}

	return len(s)
}

type Line struct {
	Start int
	End   int
	Width int
}

func Wrap(s string, width int) []Line {
	if width <= 0 || len(s) == 0 {
		return nil
	}

	var lines []Line
	currentWidth := 0
	lineStart := 0

	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		runeWidth := cellwidth.RuneWidth(r)

		if r == '\n' {
			if currentWidth > 0 {
				lines = append(lines, Line{
					Start: lineStart,
					End:   i,
					Width: currentWidth,
				})
			}
			i += size
			lineStart = i
			currentWidth = 0
			continue
		}

		if currentWidth+runeWidth > width {
			if currentWidth > 0 {
				lines = append(lines, Line{
					Start: lineStart,
					End:   i,
					Width: currentWidth,
				})
				lineStart = i
				currentWidth = 0
			}

			if runeWidth > width {
				i += size
				continue
			}
		}

		currentWidth += runeWidth
		i += size
	}

	if currentWidth > 0 {
		lines = append(lines, Line{
			Start: lineStart,
			End:   len(s),
			Width: currentWidth,
		})
	}

	return lines
}
