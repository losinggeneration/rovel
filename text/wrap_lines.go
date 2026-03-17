package text

type WrapMode int

const (
	WrapCluster WrapMode = iota
	WrapNone
	WrapWord
)

type WrapOptions struct {
	Width int
	Mode  WrapMode
}

type WrappedLine struct {
	StartByte int
	EndByte   int
	Cols      int
	HardBreak bool // true if terminated by '\n'
}

func WrapLines(s string, opts WrapOptions) []WrappedLine {
	if s == "" {
		return []WrappedLine{{StartByte: 0, EndByte: 0, Cols: 0, HardBreak: false}}
	}

	mode := opts.Mode
	if opts.Width <= 0 {
		mode = WrapNone
	}

	var out []WrappedLine

	lineStart := 0

	for i := 0; i <= len(s); i++ {
		atEnd := i == len(s)

		isNL := !atEnd && s[i] == '\n'
		if !atEnd && !isNL {
			continue
		}

		hardBreak := isNL
		segmentStart := lineStart
		segmentEnd := i

		segOutStart := len(out)
		emit := func(wl WrappedLine) {
			out = append(out, wl)
		}

		switch mode {
		case WrapNone:
			emit(WrappedLine{
				StartByte: segmentStart,
				EndByte:   segmentEnd,
				Cols:      WidthBetween(s, segmentStart, segmentEnd),
				HardBreak: hardBreak,
			})
		case WrapCluster:
			wrapSegmentClusters(s, segmentStart, segmentEnd, opts.Width, hardBreak, emit)
		case WrapWord:
			wrapSegmentWords(s, segmentStart, segmentEnd, opts.Width, hardBreak, emit)
		default:
			wrapSegmentClusters(s, segmentStart, segmentEnd, opts.Width, hardBreak, emit)
		}

		if len(out) > segOutStart {
			out[len(out)-1].HardBreak = hardBreak
		}

		if isNL {
			lineStart = i + 1
		}
	}

	if len(out) == 0 {
		return []WrappedLine{{StartByte: 0, EndByte: 0, Cols: 0, HardBreak: false}}
	}

	return out
}

func wrapSegmentClusters(
	s string,
	startByte, endByte int,
	width int,
	_ bool,
	emit func(WrappedLine),
) {
	if startByte == endByte {
		emit(WrappedLine{StartByte: startByte, EndByte: endByte, Cols: 0, HardBreak: false})

		return
	}

	pos := startByte
	for pos < endByte {
		lineStart := pos
		cols := 0
		lineEnd := pos

		for pos < endByte {
			next := NextCluster(s, pos)
			if next > endByte {
				next = endByte
			}

			clusterW := WidthBetween(s, pos, next)

			if width > 0 && cols == 0 && clusterW > width {
				cols = clusterW
				pos = next
				lineEnd = pos

				break
			}

			if width > 0 && cols > 0 && cols+clusterW > width {
				break
			}

			cols += clusterW
			pos = next
			lineEnd = pos
		}

		if lineEnd == lineStart {
			next := NextCluster(s, lineStart)
			if next > endByte {
				next = endByte
			}

			lineEnd = next
			cols = WidthBetween(s, lineStart, lineEnd)
			pos = lineEnd
		}

		emit(WrappedLine{
			StartByte: lineStart,
			EndByte:   lineEnd,
			Cols:      cols,
			HardBreak: false,
		})
	}
}

// wrapSegmentWords is a simple word-wrapping strategy based on whitespace
// clusters. It prefers breaking after ASCII spaces or tabs when possible.
func wrapSegmentWords(
	s string,
	startByte, endByte int,
	width int,
	_ bool,
	emit func(WrappedLine),
) {
	if startByte == endByte {
		emit(WrappedLine{StartByte: startByte, EndByte: endByte, Cols: 0, HardBreak: false})

		return
	}

	pos := startByte
	for pos < endByte {
		lineStart := pos
		cols := 0
		lineEnd := pos

		lastBreakPos := -1
		lastBreakCols := 0

		for pos < endByte {
			next := NextCluster(s, pos)
			if next > endByte {
				next = endByte
			}

			cluster := s[pos:next]
			clusterW := WidthBetween(s, pos, next)

			if width > 0 && cols == 0 && clusterW > width {
				cols = clusterW
				pos = next
				lineEnd = pos

				break
			}

			if width > 0 && cols > 0 && cols+clusterW > width {
				if lastBreakPos > lineStart {
					pos = lastBreakPos
					lineEnd = lastBreakPos
					cols = lastBreakCols

					break
				}

				cols += clusterW
				lineEnd = next
				pos = next

				break
			}

			cols += clusterW
			pos = next
			lineEnd = pos

			if cluster == " " || cluster == "\t" {
				lastBreakPos = pos
				lastBreakCols = cols
			}
		}

		if lineEnd == lineStart {
			next := NextCluster(s, lineStart)
			if next > endByte {
				next = endByte
			}

			lineEnd = next
			cols = WidthBetween(s, lineStart, lineEnd)
			pos = lineEnd
		}

		emit(WrappedLine{
			StartByte: lineStart,
			EndByte:   lineEnd,
			Cols:      cols,
			HardBreak: false,
		})
	}
}
