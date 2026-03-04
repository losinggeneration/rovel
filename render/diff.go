package render

type Run struct {
	Y  int
	X0 int // inclusive
	X1 int // exclusive
}

func DiffRuns(back, front *Buffer, dmg *Damage) []Run {
	// Validate buffer sizes match damage dimensions
	if back == nil || front == nil {
		return nil
	}
	if dmg != nil && (back.W != dmg.W || back.H != dmg.H ||
		front.W != dmg.W || front.H != dmg.H) {
		// Mismatch: return empty - defensive programming
		return nil
	}
	// TODO: consider merging adjacent runs per row.

	if dmg == nil || dmg.IsEmpty() {
		return nil
	}

	runs := make([]Run, 0, 64)

	for y := 0; y < dmg.H; y++ {
		spans := dmg.Rows[y]
		if len(spans) == 0 {
			continue
		}

		for _, sp := range spans {
			x := sp.X0
			for x < sp.X1 {
				if !cellsDiffer(*back.At(x, y), *front.At(x, y)) {
					x++
					continue
				}

				runX0 := x
				runX1 := x + 1

				for runX1 < sp.X1 &&
					cellsDiffer(*back.At(runX1, y), *front.At(runX1, y)) {
					runX1++
				}

				runX0, runX1 = expandForWide(back, front, y, runX0, runX1)

				if runX0 < 0 {
					runX0 = 0
				}
				if runX1 > dmg.W {
					runX1 = dmg.W
				}
				if runX0 < runX1 {
					runs = append(runs, Run{
						Y:  y,
						X0: runX0,
						X1: runX1,
					})
				}

				x = runX1
			}
		}
	}

	// TODO: coalesce runs on same row that overlap/are adjacent after wide-expansion.
	return coalesceRuns(runs)
}

func cellsDiffer(a, b Cell) bool {
	// Keep in sync with what Flush writes and what Painter sets.
	// TODO: if you add fields to Cell that affect terminal output, include them.
	return a.R != b.R ||
		a.Style != b.Style ||
		a.Wide != b.Wide ||
		a.WideCont != b.WideCont
}

func expandForWide(back, front *Buffer, y, x0, x1 int) (int, int) {
	w := back.W

	if x0 > 0 {
		if back.At(x0, y).WideCont || front.At(x0, y).WideCont {
			x0--
		}
	}

	// Conservative scan for continuation cells within the run.
	for x := x0; x < x1; x++ {
		if x > 0 && (back.At(x, y).WideCont || front.At(x, y).WideCont) {
			if x-1 < x0 {
				x0 = x - 1
			}
		}
	}

	if x1 < w {
		if back.At(x1-1, y).Wide || front.At(x1-1, y).Wide {
			x1++
		}
	}

	return x0, x1
}

func coalesceRuns(runs []Run) []Run {
	// MVP: simple O(n log n) sort by (Y, X0) and merge.
	// TODO: implement; for now assume produced in sorted order by scan.
	if len(runs) <= 1 {
		return runs
	}

	out := runs[:0]
	cur := runs[0]

	for i := 1; i < len(runs); i++ {
		r := runs[i]
		if r.Y == cur.Y && r.X0 <= cur.X1 {
			if r.X1 > cur.X1 {
				cur.X1 = r.X1
			}
			continue
		}
		out = append(out, cur)
		cur = r
	}

	out = append(out, cur)
	return out
}
