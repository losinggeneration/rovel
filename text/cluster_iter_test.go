package text

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/rivo/uniseg"
)

// clusterCorpus covers the shapes grapheme segmentation actually has to get
// right: ASCII, combining marks, ZWJ emoji sequences, regional indicator pairs
// (whose breaking is stateful), wide CJK, and invalid UTF-8.
var clusterCorpus = []string{
	"",
	"a",
	"hello world",
	"áb",   // combining acute
	"é̂̃f", // stacked combining marks
	"👍",
	"👨‍👩‍👧‍👦", // ZWJ family
	"🇺🇸",      // one regional indicator pair
	"🇺🇸🇬🇧",    // two RI pairs — parity matters
	"🇺🇸🇬🇧🇫🇷x", // odd trailing content after RI pairs
	"日本語のテキスト",
	"mixed 日本 👍 á 🇺🇸 end",
	"tab\tand space",
	"\r\n",
	"a\r\nb",
	"line1\r\nline2\r\n",
	"\r",
	"\n\r",
	"á\r\nb",
	"x\r\n\xff",
	"\xff\xfe invalid",
	"good\xffbad",
	strings.Repeat("ab́", 50),
	strings.Repeat("🇺🇸", 20),
}

// referenceClusters returns cluster boundaries via uniseg's Graphemes iterator,
// which is the definition the original implementation was written against.
func referenceClusters(s string) [][2]int {
	var out [][2]int
	if !utf8.ValidString(s) {
		for i := range len(s) {
			out = append(out, [2]int{i, i + 1})
		}

		return out
	}

	g := uniseg.NewGraphemes(s)
	for g.Next() {
		start, end := g.Positions()
		out = append(out, [2]int{start, end})
	}

	return out
}

// TestForEachClusterMatchesReference is the safety net for replacing the
// cluster iterator's engine: the boundaries it reports must be identical to
// uniseg's Graphemes iterator for every input in the corpus.
func TestForEachClusterMatchesReference(t *testing.T) {
	for _, s := range clusterCorpus {
		want := referenceClusters(s)

		var got [][2]int

		forEachCluster(s, func(start, end int) bool {
			got = append(got, [2]int{start, end})

			return true
		})

		if len(got) != len(want) {
			t.Fatalf("forEachCluster(%q): %d clusters, want %d\n got: %v\nwant: %v", s, len(got), len(want), got, want)
		}

		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("forEachCluster(%q): cluster %d = %v, want %v", s, i, got[i], want[i])
			}
		}
	}
}

// TestClusterNavigationMatchesReference checks NextCluster, PrevCluster and
// ClampCluster at every byte offset against boundaries derived from the
// reference iterator.
func TestClusterNavigationMatchesReference(t *testing.T) {
	for _, s := range clusterCorpus {
		clusters := referenceClusters(s)

		for off := range len(s) + 2 {
			wantClamp := clampFrom(clusters, s, off)
			if got := ClampCluster(s, off); got != wantClamp {
				t.Errorf("ClampCluster(%q, %d) = %d, want %d", s, off, got, wantClamp)
			}

			wantNext := nextFrom(clusters, s, off)
			if got := NextCluster(s, off); got != wantNext {
				t.Errorf("NextCluster(%q, %d) = %d, want %d", s, off, got, wantNext)
			}

			wantPrev := prevFrom(clusters, s, off)
			if got := PrevCluster(s, off); got != wantPrev {
				t.Errorf("PrevCluster(%q, %d) = %d, want %d", s, off, got, wantPrev)
			}
		}
	}
}

func clampFrom(clusters [][2]int, s string, off int) int {
	if off <= 0 {
		return 0
	}

	if off >= len(s) {
		return len(s)
	}

	for _, c := range clusters {
		if off == c[0] || off < c[1] {
			return c[0]
		}
	}

	return len(s)
}

func nextFrom(clusters [][2]int, s string, off int) int {
	if off <= 0 {
		off = 0
	}

	if off >= len(s) {
		return len(s)
	}

	start := clampFrom(clusters, s, off)
	for _, c := range clusters {
		if c[0] == start {
			return c[1]
		}
	}

	return len(s)
}

func prevFrom(clusters [][2]int, s string, off int) int {
	if off <= 0 {
		return 0
	}

	if off > len(s) {
		off = len(s)
	}

	start := clampFrom(clusters, s, off)
	if start <= 0 {
		return 0
	}

	prev := 0
	for _, c := range clusters {
		if c[0] == start {
			break
		}

		prev = c[0]
	}

	return prev
}

// TestWrapLinesScalesLinearly guards against cluster navigation regressing to
// rescanning from the start of the string. That regression is invisible in
// correctness tests and only shows up as wrapping cost growing with the square
// of the input: a 64KB line took ~400s before cluster iteration became
// resumable, versus low milliseconds after. The bound below is deliberately
// enormous so it cannot flake on a loaded machine — it is checking the growth
// rate, not a performance target.
func TestWrapLinesScalesLinearly(t *testing.T) {
	if testing.Short() {
		t.Skip("timing-based")
	}

	const budget = 5 * time.Second

	for _, s := range []string{
		strings.Repeat("some plausible tool output text ", 2048),
		strings.Repeat("日本語 á 👍 🇺🇸 mixed width text ", 1024),
	} {
		start := time.Now()
		_ = WrapLines(s, WrapOptions{Width: 100, Mode: WrapWord})

		if elapsed := time.Since(start); elapsed > budget {
			t.Fatalf("wrapping %d bytes took %v (budget %v): cluster iteration is rescanning, not resuming",
				len(s), elapsed, budget)
		}
	}
}
