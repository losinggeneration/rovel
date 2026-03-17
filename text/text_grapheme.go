package text

import (
	"unicode/utf8"

	cellwidth "github.com/losinggeneration/tui/internal/width"
)

func NextCluster(s string, i int) int {
	if i <= 0 {
		i = 0
	}

	if i >= len(s) {
		return len(s)
	}

	if !utf8.ValidString(s) {
		if i+1 > len(s) {
			return len(s)
		}

		return i + 1
	}

	i = ClampCluster(s, i)
	if i >= len(s) {
		return len(s)
	}

	next := len(s)
	found := false

	forEachCluster(s, func(start, end int) bool {
		if start == i {
			next = end
			found = true

			return false
		}

		return true
	})

	if !found {
		return len(s)
	}

	return next
}

func PrevCluster(s string, i int) int {
	if i <= 0 {
		return 0
	}

	if i > len(s) {
		i = len(s)
	}

	if !utf8.ValidString(s) {
		if i-1 < 0 {
			return 0
		}

		return i - 1
	}

	i = ClampCluster(s, i)
	if i <= 0 {
		return 0
	}

	prev := 0
	found := false

	forEachCluster(s, func(start, end int) bool {
		_ = end

		if start == i {
			found = true

			return false
		}

		prev = start

		return true
	})

	if i == len(s) {
		// i == len(s) is a legal boundary but not a cluster start.
		last := 0

		forEachCluster(s, func(start, end int) bool {
			_ = end
			last = start

			return true
		})

		return last
	}

	if !found {
		return 0
	}

	return prev
}

func ColumnOf(s string, byteOff int) int {
	byteOff = ClampCluster(s, byteOff)

	return WidthBetween(s, 0, byteOff)
}

type ColumnBias int

const (
	BiasLeft ColumnBias = iota
	BiasRight
)

func OffsetAtColumn(s string, col int) int {
	return OffsetAtColumnBias(s, col, BiasLeft)
}

func OffsetAtColumnBias(s string, col int, bias ColumnBias) int {
	if col <= 0 || len(s) == 0 {
		return 0
	}

	if col < 0 {
		col = 0
	}

	width := 0
	out := 0
	done := false

	forEachCluster(s, func(start, end int) bool {
		clusterW := WidthBetween(s, start, end)
		if width+clusterW > col {
			if bias == BiasRight {
				out = end
			} else {
				out = start
			}

			done = true

			return false
		}

		if width+clusterW == col {
			out = end
			done = true

			return false
		}

		width += clusterW
		out = end

		return true
	})

	if done {
		return out
	}

	return len(s)
}

type Line struct {
	Start int
	End   int
	Width int
}

func Wrap(s string, width int) []Line {
	wrapped := WrapLines(s, WrapOptions{Width: width, Mode: WrapCluster})
	if len(wrapped) == 0 {
		return nil
	}

	out := make([]Line, 0, len(wrapped))
	for _, wl := range wrapped {
		out = append(out, Line{
			Start: wl.StartByte,
			End:   wl.EndByte,
			Width: wl.Cols,
		})
	}

	return out
}

func ClampCluster(s string, byteOff int) int {
	if byteOff <= 0 {
		return 0
	}

	if byteOff >= len(s) {
		return len(s)
	}

	if !utf8.ValidString(s) {
		return byteOff
	}

	out := 0
	found := false

	forEachCluster(s, func(start, end int) bool {
		if byteOff == start {
			out = start
			found = true

			return false
		}

		if byteOff < end {
			out = start
			found = true

			return false
		}

		return true
	})

	if found {
		return out
	}

	return len(s)
}

func IsClusterBoundary(s string, byteOff int) bool {
	if byteOff < 0 || byteOff > len(s) {
		return false
	}

	if byteOff == 0 || byteOff == len(s) {
		return true
	}

	if !utf8.ValidString(s) {
		return true
	}

	isBoundary := false

	forEachCluster(s, func(start, end int) bool {
		_ = end

		if start == byteOff {
			isBoundary = true

			return false
		}

		return true
	})

	return isBoundary
}

func WidthBetween(s string, startByte, endByte int) int {
	if startByte < 0 {
		startByte = 0
	}

	if endByte > len(s) {
		endByte = len(s)
	}

	if startByte >= endByte {
		return 0
	}

	width := 0

	for i := startByte; i < endByte; {
		r, size := utf8.DecodeRuneInString(s[i:])
		width += cellwidth.RuneWidth(r)

		if size <= 0 {
			break
		}

		i += size
	}

	return width
}
