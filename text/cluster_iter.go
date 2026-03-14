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
	if len(s) == 0 {
		return
	}
	if !utf8.ValidString(s) {
		for i := 0; i < len(s); i++ {
			if !fn(i, i+1) {
				return
			}
		}
		return
	}

	g := uniseg.NewGraphemes(s)
	for g.Next() {
		start, end := g.Positions()
		if !fn(start, end) {
			return
		}
	}
}

func forEachClusterInRange(s string, startByte, endByte int, fn func(start, end int) bool) {
	if startByte < 0 {
		startByte = 0
	}
	if endByte > len(s) {
		endByte = len(s)
	}
	if startByte >= endByte {
		return
	}

	forEachCluster(s, func(start, end int) bool {
		if end <= startByte {
			return true
		}
		if start >= endByte {
			return false
		}
		if start < startByte {
			start = startByte
		}
		if end > endByte {
			end = endByte
		}
		return fn(start, end)
	})
}
