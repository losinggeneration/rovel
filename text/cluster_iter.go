package text

import (
	"unicode/utf8"

	"github.com/rivo/uniseg"
)

// forEachCluster calls fn for each grapheme cluster in s, in order.
// The (start, end) offsets are UTF-8 byte offsets.
//
// If s is not valid UTF-8, we conservatively treat every byte as its own
// cluster so navigation and mapping remain total functions.
func forEachCluster(s string, fn func(start, end int) bool) {
	forEachClusterFrom(s, 0, fn)
}

// forEachClusterFrom is forEachCluster resumed at a byte offset that is already
// a cluster boundary. Offsets reported to fn are absolute within s.
//
// Resuming is sound because grapheme cluster breaking never spans a boundary:
// the only stateful rule (regional-indicator parity) resets there, so scanning
// from a boundary yields the same segmentation as scanning the whole string.
// That property is what lets cluster navigation avoid rescanning from byte 0,
// which is the difference between linear and quadratic work for callers that
// walk a string cluster by cluster.
func forEachClusterFrom(s string, from int, fn func(start, end int) bool) {
	if from < 0 {
		from = 0
	}

	if from >= len(s) {
		return
	}

	if !utf8.ValidString(s) {
		for i := from; i < len(s); i++ {
			if !fn(i, i+1) {
				return
			}
		}

		return
	}

	// FirstGraphemeClusterInString computes grapheme boundaries and width only.
	// The Graphemes iterator additionally derives word, sentence and line-break
	// state on every step, none of which this package reads — that bookkeeping
	// dominated the cost of every wrap and navigation call.
	rest := s[from:]
	pos := from
	state := -1

	for len(rest) > 0 {
		var cluster string

		cluster, rest, _, state = uniseg.FirstGraphemeClusterInString(rest, state)

		end := pos + len(cluster)
		if !fn(pos, end) {
			return
		}

		pos = end
	}
}

// asciiClusterBoundary reports whether i is provably a cluster boundary from
// its immediate neighbours alone, without scanning or validating the rest of s.
//
// Two adjacent ASCII bytes are always a boundary: ASCII carries no combining
// marks, joiners or variation selectors, so the only ASCII cluster spanning
// more than one byte is CRLF. The answer is the same whether or not s is valid
// UTF-8 elsewhere — under the invalid-input contract every byte is its own
// cluster, which agrees here — so this may be checked before validation.
// false means "unknown", not "not a boundary".
func asciiClusterBoundary(s string, i int) bool {
	if i <= 0 || i >= len(s) {
		return true
	}

	prev, cur := s[i-1], s[i]
	if prev >= utf8.RuneSelf || cur >= utf8.RuneSelf {
		return false
	}

	return !(prev == '\r' && cur == '\n')
}

// clusterCursor walks grapheme clusters forward, carrying uniseg's break state
// so each step costs one cluster rather than a rescan from the start of the
// string. It also validates the string once instead of once per step.
//
// Callers that need to rewind may seek to any offset they have already seen as
// a cluster boundary; seeking restarts the break state, which is sound at a
// boundary (see forEachClusterFrom).
type clusterCursor struct {
	s     string
	valid bool
	pos   int
	state int
}

func newClusterCursor(s string) *clusterCursor {
	return &clusterCursor{s: s, valid: utf8.ValidString(s), pos: -1, state: -1}
}

func (c *clusterCursor) seek(at int) {
	if at == c.pos {
		return
	}

	c.pos = at
	c.state = -1
}

// next returns the end offset of the cluster starting at the cursor position
// and advances past it.
func (c *clusterCursor) next() int {
	if c.pos >= len(c.s) {
		return len(c.s)
	}

	if !c.valid {
		c.pos++

		return c.pos
	}

	var cluster string

	cluster, _, _, c.state = uniseg.FirstGraphemeClusterInString(c.s[c.pos:], c.state)
	c.pos += len(cluster)

	return c.pos
}

// clusterEndAt returns the end offset of the cluster starting at the boundary
// start. It reads exactly one cluster rather than iterating the string.
func clusterEndAt(s string, start int) int {
	if start < 0 {
		start = 0
	}

	if start >= len(s) {
		return len(s)
	}

	cluster, _, _, _ := uniseg.FirstGraphemeClusterInString(s[start:], -1)

	return start + len(cluster)
}
