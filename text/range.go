package text

type Range struct {
	Start int // byte offset, inclusive
	End   int // byte offset, exclusive
}

func (r Range) Normalized() Range {
	if r.Start <= r.End {
		return r
	}

	return Range{Start: r.End, End: r.Start}
}

func (r Range) Empty() bool {
	return r.Start == r.End
}

func ClampRange(s string, r Range) Range {
	if r.Start < 0 {
		r.Start = 0
	}

	if r.End < 0 {
		r.End = 0
	}

	if r.Start > len(s) {
		r.Start = len(s)
	}

	if r.End > len(s) {
		r.End = len(s)
	}

	r = r.Normalized()
	r.Start = ClampCluster(s, r.Start)
	r.End = ClampCluster(s, r.End)

	if r.Start > r.End {
		r = Range{Start: r.End, End: r.End}
	}

	return r
}

func DeleteRange(s string, r Range) (out string, deleted Range) {
	r = ClampRange(s, r)
	if r.Empty() {
		return s, r
	}

	return s[:r.Start] + s[r.End:], r
}

func ReplaceRange(s string, r Range, insert string) (out string, inserted Range) {
	r = ClampRange(s, r)
	if insert == "" {
		out, _ := DeleteRange(s, r)

		return out, Range{Start: r.Start, End: r.Start}
	}

	out = s[:r.Start] + insert + s[r.End:]
	inserted = Range{Start: r.Start, End: r.Start + len(insert)}
	inserted = ClampRange(out, inserted)

	return out, inserted
}

func DeletePrevCluster(s string, caret int) (out string, newCaret int) {
	caret = ClampCluster(s, caret)
	if caret <= 0 {
		return s, 0
	}

	start := PrevCluster(s, caret)
	out = s[:start] + s[caret:]
	newCaret = start

	return out, newCaret
}

func DeleteNextCluster(s string, caret int) (out string, newCaret int) {
	caret = ClampCluster(s, caret)
	if caret >= len(s) {
		return s, len(s)
	}

	end := NextCluster(s, caret)
	out = s[:caret] + s[end:]
	newCaret = caret

	return out, newCaret
}
