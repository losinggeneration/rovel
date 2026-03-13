// Package style provides color and style types for the tui library.
//
// These types are shared across render, backend, and the root tui package
// to avoid import cycles.
//
// This package is unstable before v0.1.0.
package style

// Color is now defined in color.go as a tagged uint32.
// This file provides the Style type and derivation helpers.

// AttrMask represents text attributes (bold, dim, italic, etc.).
type AttrMask uint16

const (
	AttrBold AttrMask = 1 << iota
	AttrDim
	AttrItalic
	AttrUnderline
	AttrBlink
	AttrReverse
)

// Style represents foreground color, background color, and text attributes.
type Style struct {
	FG   Color
	BG   Color
	Attr AttrMask
}

// WithFG returns a new style with the foreground color set.
func (s Style) WithFG(c Color) Style {
	s.FG = c
	return s
}

// WithBG returns a new style with the background color set.
func (s Style) WithBG(c Color) Style {
	s.BG = c
	return s
}

// WithAttr returns a new style with the attribute added.
func (s Style) WithAttr(a AttrMask) Style {
	s.Attr |= a
	return s
}

// WithoutAttr returns a new style with the attribute removed.
func (s Style) WithoutAttr(a AttrMask) Style {
	s.Attr &^= a
	return s
}

// Merge combines base and override styles.
// Non-default colors from override replace base colors.
// Attributes are OR'd together.
func Merge(base, override Style) Style {
	result := base
	if override.FG != 0 { // non-default
		result.FG = override.FG
	}
	if override.BG != 0 {
		result.BG = override.BG
	}
	result.Attr |= override.Attr
	return result
}

// Equals returns true if two styles are identical.
func (s Style) Equals(other Style) bool {
	return s.FG == other.FG && s.BG == other.BG && s.Attr == other.Attr
}
