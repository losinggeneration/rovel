package text

import (
	"slices"
	"strings"

	"github.com/rivo/uniseg"
)

// StripANSI removes ANSI/terminal control sequences that do not occupy
// terminal cells. It handles CSI, OSC, DCS, APC, PM, and simple ESC sequences.
func StripANSI(s string) string {
	out := strings.Builder{}
	for i := 0; i < len(s); {
		if s[i] == 0x1b {
			if n := skipANSI(s, i); n > i {
				i = n
				continue
			}
		}
		cluster, n := nextANSICluster(s, i)
		out.WriteString(cluster)
		i = n
	}
	return out.String()
}

// ANSIWidth returns the visible terminal cell width of s after ignoring ANSI
// and terminal control sequences.
func ANSIWidth(s string) int {
	return uniseg.StringWidth(StripANSI(s))
}

// ANSITruncate returns s truncated to width visible cells while preserving ANSI
// sequences and appending a reset when truncation leaves an SGR style open.
func ANSITruncate(s string, width int, ellipsis bool) string {
	if width <= 0 {
		return ""
	}
	limit := width
	suffix := ""
	if ellipsis && ANSIWidth(s) > width {
		limit = width - 1
		suffix = "…"
	}
	if limit < 0 {
		limit = 0
	}

	out := strings.Builder{}
	w := 0
	open := false
	for i := 0; i < len(s); {
		cluster, n := nextANSICluster(s, i)
		i = n
		if cluster == "" {
			continue
		}
		if cluster[0] == 0x1b {
			out.WriteString(cluster)
			open = updateSGROpen(open, cluster)
			continue
		}
		cw := uniseg.StringWidth(cluster)
		if w+cw > limit {
			break
		}
		out.WriteString(cluster)
		w += cw
	}

	out.WriteString(suffix)
	if open {
		out.WriteString("\x1b[0m")
	}

	return out.String()
}

// ANSIWrap wraps s to width visible cells while preserving ANSI sequences and
// closing/reopening SGR styles across wrapped lines.
func ANSIWrap(s string, width int) []string {
	if width <= 0 {
		return nil
	}
	var lines []string
	cur := strings.Builder{}
	w := 0
	open := []string{}
	for i := 0; i < len(s); {
		cluster, n := nextANSICluster(s, i)
		i = n
		if cluster == "\n" {
			if len(open) > 0 {
				cur.WriteString("\x1b[0m")
			}
			lines = append(lines, cur.String())
			cur.Reset()
			for _, seq := range open {
				cur.WriteString(seq)
			}
			w = 0
			continue
		}
		if cluster != "" && cluster[0] == 0x1b {
			cur.WriteString(cluster)
			if isSGRReset(cluster) {
				open = nil
			} else if strings.HasSuffix(cluster, "m") {
				open = append(open, cluster)
			}
			continue
		}
		cw := uniseg.StringWidth(cluster)
		if w > 0 && w+cw > width {
			if len(open) > 0 {
				cur.WriteString("\x1b[0m")
			}
			lines = append(lines, cur.String())
			cur.Reset()
			for _, seq := range open {
				cur.WriteString(seq)
			}
			w = 0
		}
		cur.WriteString(cluster)
		w += cw
	}
	if len(open) > 0 {
		cur.WriteString("\x1b[0m")
	}
	lines = append(lines, cur.String())
	return lines
}

// ANSIWrapWords wraps s to width visible cells, preferring breaks at spaces
// and hard-breaking words wider than width. ANSI sequences are preserved and
// SGR styles are closed and reopened across wrapped lines, as in ANSIWrap.
// Break spaces are dropped: wrapped lines never start with a space.
func ANSIWrapWords(s string, width int) []string {
	if width <= 0 {
		return nil
	}

	var lines []string
	buf := []string{}
	open := []string{}        // SGR styles open at the current stream position
	openAtSpace := []string{} // open as of the last breakable space
	cols := 0
	lastSpace := -1
	hasWord := false
	dropSpaces := false

	// emit flushes buf[:keep] as a line (trimming trailing spaces and closing
	// the styles in carry, the open stack as of the break point) and carries
	// the remainder into the next line, dropping its leading spaces and
	// reopening the carried styles.
	emit := func(keep int, carry []string) {
		end := keep
		for end > 0 && buf[end-1] == " " {
			end--
		}
		var b strings.Builder
		for _, c := range buf[:end] {
			b.WriteString(c)
		}
		if len(carry) > 0 {
			b.WriteString("\x1b[0m")
		}
		lines = append(lines, b.String())

		rest := make([]string, 0, len(carry)+len(buf)-keep)
		rest = append(rest, carry...)
		i := keep
		for i < len(buf) && buf[i] == " " {
			i++
		}
		rest = append(rest, buf[i:]...)
		buf = rest

		// Reset the line state for the remainder. It is always escape
		// sequences plus the tail of the current word, never a space (every
		// break lands at or after the last one), so lastSpace restarts at -1
		// and only the width needs recomputing.
		cols = 0
		lastSpace = -1
		hasWord = false
		for _, c := range buf {
			if c == "" || c[0] == 0x1b {
				continue
			}
			cols += uniseg.StringWidth(c)
			hasWord = true
		}
	}

	for i := 0; i < len(s); {
		cluster, n := nextANSICluster(s, i)
		i = n
		switch {
		case cluster == "\n":
			emit(len(buf), open)
			dropSpaces = false
		case cluster == " ":
			switch {
			case dropSpaces:
				// A space at the start of a wrapped line is part of the break.
			case cols > 0 && cols+1 > width:
				emit(len(buf), open)
				dropSpaces = true
			default:
				buf = append(buf, cluster)
				cols++
				if hasWord {
					lastSpace = len(buf)
					openAtSpace = slices.Clone(open)
				}
			}
		case cluster != "" && cluster[0] == 0x1b:
			buf = append(buf, cluster)
			if isSGRReset(cluster) {
				open = nil
			} else if strings.HasSuffix(cluster, "m") {
				open = append(open, cluster)
			}
		default:
			cw := uniseg.StringWidth(cluster)
			if cols > 0 && cols+cw > width {
				if lastSpace > 0 {
					emit(lastSpace, openAtSpace)
					// emit recomputed cols for the word tail the cluster
					// will follow; a wide grapheme may still not fit after
					// it (a zero-width cluster can pad the tail). Break
					// the tail onto its own line rather than emit an
					// overwide line.
					if cols+cw > width {
						emit(len(buf), open)
					}
				} else {
					emit(len(buf), open)
				}
			}
			buf = append(buf, cluster)
			cols += cw
			hasWord = true
			dropSpaces = false
		}
	}
	emit(len(buf), open)

	return lines
}

// ANSISliceStrict returns the substring whose visible cell interval is wholly
// contained in [start,end). Wide graphemes crossing either boundary are omitted.
func ANSISliceStrict(s string, start, end int) string {
	if start < 0 {
		start = 0
	}
	if end <= start {
		return ""
	}
	out := strings.Builder{}
	w := 0
	open := false
	for i := 0; i < len(s); {
		cluster, n := nextANSICluster(s, i)
		i = n
		if cluster == "" {
			continue
		}
		if cluster[0] == 0x1b {
			if w >= start && w < end {
				out.WriteString(cluster)
				open = updateSGROpen(open, cluster)
			}
			continue
		}
		cw := uniseg.StringWidth(cluster)
		if w >= start && w+cw <= end {
			out.WriteString(cluster)
		}
		w += cw
		if w >= end {
			break
		}
	}
	if open {
		out.WriteString("\x1b[0m")
	}
	return out.String()
}

// isSGRReset reports whether seq is an SGR sequence whose net effect is a
// full attribute reset: the parameter list is empty ("\x1b[m"; parameters
// default to 0) or its final parameter is 0 ("\x1b[0m", "\x1b[00m",
// "\x1b[1;0m"), parameters being applied left to right. Anything else —
// sequences with subparameters (colons) or other non-numeric tails — is
// treated as an attribute setter.
func isSGRReset(seq string) bool {
	if !strings.HasPrefix(seq, "\x1b[") || !strings.HasSuffix(seq, "m") {
		return false
	}
	params := seq[2 : len(seq)-1]
	last := params[strings.LastIndexByte(params, ';')+1:]
	for i := 0; i < len(last); i++ {
		if last[i] < '0' || last[i] > '9' {
			return false
		}
	}
	return last == "" || strings.TrimLeft(last, "0") == ""
}

func updateSGROpen(open bool, seq string) bool {
	if isSGRReset(seq) {
		return false
	}
	if strings.HasSuffix(seq, "m") {
		return true
	}
	return open
}

func skipANSI(s string, i int) int {
	if i+1 >= len(s) || s[i] != 0x1b {
		return i
	}
	switch s[i+1] {
	case '[':
		j := i + 2
		for j < len(s) {
			if s[j] >= 0x40 && s[j] <= 0x7e {
				return j + 1
			}
			j++
		}
		return len(s)
	case ']':
		if j := strings.IndexByte(s[i+2:], 0x07); j >= 0 {
			return i + 2 + j + 1
		}
		if j := strings.Index(s[i+2:], "\x1b\\"); j >= 0 {
			return i + 2 + j + 2
		}
		return len(s)
	case 'P', '_', '^':
		if j := strings.Index(s[i+2:], "\x1b\\"); j >= 0 {
			return i + 2 + j + 2
		}
		return len(s)
	default:
		return i + 2
	}
}

func nextANSICluster(s string, i int) (string, int) {
	if i >= len(s) {
		return "", i
	}
	if s[i] == 0x1b {
		if n := skipANSI(s, i); n > i {
			return s[i:n], n
		}
	}
	cluster, rest, _, _ := uniseg.FirstGraphemeClusterInString(s[i:], -1)
	return cluster, len(s) - len(rest)
}
