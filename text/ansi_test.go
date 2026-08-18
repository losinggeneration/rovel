package text

import (
	"strings"
	"testing"
)

func TestStripANSIAndANSIWidth(t *testing.T) {
	s := "\x1b[31mred\x1b[0m \x1b]8;;url\x1b\\link\x1b]8;;\x1b\\ 界"
	if StripANSI(s) != "red link 界" {
		t.Fatalf("strip=%q", StripANSI(s))
	}
	if ANSIWidth(s) != 11 {
		t.Fatalf("width=%d", ANSIWidth(s))
	}
}

func TestANSITruncatePreservesReset(t *testing.T) {
	got := ANSITruncate("\x1b[31mabcdef\x1b[0m", 4, true)
	if StripANSI(got) != "abc…" {
		t.Fatalf("got=%q strip=%q", got, StripANSI(got))
	}
	if !strings.HasSuffix(got, "\x1b[0m") {
		t.Fatalf("missing reset %q", got)
	}
}

func TestANSIWrapIgnoresANSIWidth(t *testing.T) {
	got := ANSIWrap("\x1b[31mabcd\x1b[0mef", 4)
	if len(got) != 2 || StripANSI(got[0]) != "abcd" || StripANSI(got[1]) != "ef" {
		t.Fatalf("wrap=%q", got)
	}
}

func TestANSIZeroWidthOSCAndAPC(t *testing.T) {
	s := "a\x1b_Gabc\x1b\\b\x1b]0;title\ab"
	if ANSIWidth(s) != 3 {
		t.Fatalf("width=%d strip=%q", ANSIWidth(s), StripANSI(s))
	}
}

func TestANSIWidthParityForANSIAndControlSequences(t *testing.T) {
	cases := map[string]int{
		"\x1b[1;31mbold red\x1b[0m": 8,
		"a\x1b]52;c;AAAA\ab":        2,
		"a\x1bP1$r0 q\x1b\\b":       2,
		"a\x1b_custom apc\x1b\\b":   2,
	}
	for s, want := range cases {
		if got := ANSIWidth(s); got != want {
			t.Fatalf("ANSIWidth(%q)=%d want %d strip=%q", s, got, want, StripANSI(s))
		}
	}
}

func TestANSIWidthParityForUnicodeClusters(t *testing.T) {
	cases := map[string]int{
		"e\u0301": 1,
		"👩‍👩‍👧‍👦": 2,
		"🇺🇸":      2,
		"界":       2,
		"Ａ":       2,
	}
	for s, want := range cases {
		if got := ANSIWidth(s); got != want {
			t.Fatalf("ANSIWidth(%q)=%d want %d", s, got, want)
		}
	}
}

func TestANSIWrapPreservesResetSafety(t *testing.T) {
	got := ANSIWrap("\x1b[31mabcdef", 3)
	if len(got) != 2 {
		t.Fatalf("wrap=%q", got)
	}
	for i, line := range got {
		if !strings.HasSuffix(line, "\x1b[0m") {
			t.Fatalf("line %d missing reset: %q", i, line)
		}
		if ANSIWidth(line) > 3 {
			t.Fatalf("line %d overwide: %q", i, line)
		}
	}
}

func TestANSIStrictSlicingAtVisibleCellBoundaries(t *testing.T) {
	if got := ANSISliceStrict("a界b", 0, 2); got != "a" {
		t.Fatalf("slice wide overflow got %q", got)
	}
	if got := ANSISliceStrict("a界b", 1, 3); got != "界" {
		t.Fatalf("slice wide exact got %q", got)
	}
	if got := ANSISliceStrict("e\u0301x", 0, 1); got != "e\u0301" {
		t.Fatalf("slice combining got %q", got)
	}
}

func TestANSIRegionalIndicatorRegression(t *testing.T) {
	if got := ANSIWidth("🇨🇦🇺🇸"); got != 4 {
		t.Fatalf("width=%d", got)
	}
	if got := ANSITruncate("🇨🇦x", 2, false); got != "🇨🇦" {
		t.Fatalf("truncate flag got %q", got)
	}
}

func TestANSIWrapWordsBreaksAtSpaces(t *testing.T) {
	got := ANSIWrapWords("alpha beta gamma", 10)
	if len(got) != 2 || got[0] != "alpha beta" || got[1] != "gamma" {
		t.Fatalf("wrap=%q", got)
	}
	if got := ANSIWrapWords("a  b", 2); len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("wrap collapsing spaces=%q", got)
	}
}

func TestANSIWrapWordsHardBreaksLongWords(t *testing.T) {
	got := ANSIWrapWords("abcdefghij kl", 4)
	want := []string{"abcd", "efgh", "ij", "kl"}
	if len(got) != len(want) {
		t.Fatalf("wrap=%q want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("wrap[%d]=%q want %q (all %q)", i, got[i], want[i], got)
		}
	}
}

func TestANSIWrapWordsPreservesSGRAcrossLines(t *testing.T) {
	got := ANSIWrapWords("\x1b[31malpha beta\x1b[0m", 6)
	if len(got) != 2 {
		t.Fatalf("wrap=%q", got)
	}
	if got[0] != "\x1b[31malpha\x1b[0m" || got[1] != "\x1b[31mbeta\x1b[0m" {
		t.Fatalf("wrap=%q", got)
	}
	for i, line := range got {
		if ANSIWidth(line) > 6 {
			t.Fatalf("line %d overwide: %q", i, line)
		}
	}
}

func TestANSIWrapWordsHardBreaksAfterSoftBreak(t *testing.T) {
	// Regression: after a soft break the overflowing cluster was appended
	// without rechecking fit, so a zero-width cluster (a tab) padding the
	// word tail could push a wide grapheme onto an overwide line.
	got := ANSIWrapWords("\t c界", 2)
	want := []string{"\t", "c", "界"}
	if len(got) != len(want) {
		t.Fatalf("wrap=%q want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("wrap[%d]=%q want %q (all %q)", i, got[i], want[i], got)
		}
	}
}

func TestANSIWrapWordsWidthContract(t *testing.T) {
	cases := []struct {
		in    string
		width int
	}{
		{"\t c界", 2},
		{"a\t b界\tc界", 3},
		{"\t \t\x1b[0mc界\t\x1b[1m", 2},
		{"aaaa bbbb cccc dddd", 5},
		{"\x1b[31mab cd ef\x1b[0m", 5},
		{"ab cd ", 2},
		{"hello   world", 5},
		{"a界b c", 2},
		{"界界界 界界界", 4},
	}
	for _, c := range cases {
		for i, line := range ANSIWrapWords(c.in, c.width) {
			if w := ANSIWidth(line); w > c.width {
				t.Errorf("ANSIWrapWords(%q, %d) line %d: %d cells wide: %q",
					c.in, c.width, i, w, line)
			}
		}
	}
}

func TestSGRResetVariants(t *testing.T) {
	// Reset forms with an empty or trailing-zero parameter list must clear
	// the open-style stack like "\x1b[0m" instead of being carried into
	// reopened lines.
	resets := []string{"\x1b[m", "\x1b[0m", "\x1b[00m", "\x1b[1;0m"}
	for _, reset := range resets {
		got := ANSIWrapWords("\x1b[31mab "+reset+"cd ef", 3)
		want := []string{"\x1b[31mab\x1b[0m", "\x1b[31m" + reset + "cd", "ef"}
		if len(got) != len(want) {
			t.Fatalf("reset %q: wrap=%q want %q", reset, got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("reset %q: wrap[%d]=%q want %q (all %q)", reset, i, got[i], want[i], got)
			}
		}
	}

	// "\x1b[0;31m" resets and then sets red: a net setter, not a reset.
	if got, want := ANSITruncate("\x1b[1mabc\x1b[0;31mdef", 5, false), "\x1b[1mabc\x1b[0;31mde\x1b[0m"; got != want {
		t.Fatalf("net setter truncate=%q want %q", got, want)
	}
	// A reset inside the kept prefix leaves nothing open: no trailing reset.
	if got, want := ANSITruncate("\x1b[31mabc\x1b[mdef", 5, false), "\x1b[31mabc\x1b[mde"; got != want {
		t.Fatalf("reset truncate=%q want %q", got, want)
	}

	got := ANSIWrap("\x1b[31mab \x1b[mcd", 3)
	want := []string{"\x1b[31mab \x1b[m", "cd"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("wrap=%q want %q", got, want)
	}
}

func TestANSIWrapWordsClosesStyleAtBreakPoint(t *testing.T) {
	// The reset sits after the break space: line 1 must still be closed with
	// the styles that were open AT the break, and line 2 reopens them before
	// the reset cancels them. Previously line 1 was left unbalanced.
	got := ANSIWrapWords("\x1b[31mab \x1b[0mcd ef", 3)
	want := []string{"\x1b[31mab\x1b[0m", "\x1b[31m\x1b[0mcd", "ef"}
	if len(got) != len(want) {
		t.Fatalf("wrap=%q want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("wrap[%d]=%q want %q (all %q)", i, got[i], want[i], got)
		}
	}
}

func TestANSIWrapWordsEdgeCases(t *testing.T) {
	if got := ANSIWrapWords("x", 0); got != nil {
		t.Fatalf("width 0 wrap=%q", got)
	}
	if got := ANSIWrapWords("", 5); len(got) != 1 || got[0] != "" {
		t.Fatalf("empty wrap=%q", got)
	}
	if got := ANSIWrapWords("fits fine", 20); len(got) != 1 || got[0] != "fits fine" {
		t.Fatalf("short wrap=%q", got)
	}
	if got := ANSIWrapWords("ab\ncd ef", 5); len(got) != 2 || got[0] != "ab" || got[1] != "cd ef" {
		t.Fatalf("newline wrap=%q", got)
	}
	if got := ANSIWrapWords("界a b", 3); len(got) != 2 || StripANSI(got[0]) != "界a" || got[1] != "b" {
		t.Fatalf("wide cluster wrap=%q", got)
	}
}
