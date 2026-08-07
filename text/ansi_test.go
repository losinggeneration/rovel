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
