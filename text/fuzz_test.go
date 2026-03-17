package text

import "testing"

func FuzzWrapLines(f *testing.F) {
	f.Add("hello world", 10)
	f.Add("日本語テスト", 8)
	f.Add("🌍🎉👨‍👩‍👧‍👦", 5)
	f.Add("", 10)
	f.Add("a b c d e f", 3)
	f.Add("abcdefghij\nklmnopqrst", 5)
	f.Add("word word word", 0)
	f.Add("word word word", -1)

	f.Fuzz(func(t *testing.T, s string, width int) {
		for _, mode := range []WrapMode{WrapCluster, WrapNone, WrapWord} {
			lines := WrapLines(s, WrapOptions{Width: width, Mode: mode})

			// Invariant: always returns at least one line.
			if len(lines) == 0 {
				t.Fatalf("WrapLines returned empty slice for mode=%d width=%d", mode, width)
			}

			// Invariant: lines are contiguous and cover the entire string.
			if lines[0].StartByte != 0 {
				t.Fatalf("first line starts at %d, want 0", lines[0].StartByte)
			}

			for i := 1; i < len(lines); i++ {
				if lines[i].StartByte != lines[i-1].EndByte {
					if lines[i-1].HardBreak && lines[i].StartByte == lines[i-1].EndByte+1 {
						// Hard break consumes \n.
						continue
					}

					t.Fatalf("gap between lines %d and %d: end=%d, start=%d",
						i-1, i, lines[i-1].EndByte, lines[i].StartByte)
				}
			}

			// Invariant: Cols is non-negative.
			for i, l := range lines {
				if l.Cols < 0 {
					t.Fatalf("line %d has negative Cols: %d", i, l.Cols)
				}
			}
		}
	})
}

func FuzzTruncate(f *testing.F) {
	f.Add("hello world", 5, true)
	f.Add("日本語テスト", 4, true)
	f.Add("🌍🎉", 3, false)
	f.Add("", 0, true)
	f.Add("abc", 100, false)
	f.Add("café", 3, true)

	f.Fuzz(func(t *testing.T, s string, maxCols int, ellipsis bool) {
		result := Truncate(s, maxCols, ellipsis)

		// Invariant: result width should not exceed maxCols.
		if maxCols > 0 {
			resultWidth := Width(result)
			if resultWidth > maxCols {
				t.Fatalf("Truncate(%q, %d, %v) width %d > maxCols",
					s, maxCols, ellipsis, resultWidth)
			}
		}

		// Invariant: should not panic (the fuzz engine catches this).
	})
}
