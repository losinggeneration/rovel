package text

import (
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
			if strings.HasSuffix(cluster, "m") {
				if cluster == "\x1b[0m" {
					open = nil
				} else {
					open = append(open, cluster)
				}
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

func updateSGROpen(open bool, seq string) bool {
	if strings.HasSuffix(seq, "m") {
		if seq == "\x1b[0m" {
			return false
		}
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
