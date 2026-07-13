package rovel

import (
	"github.com/losinggeneration/rovel/geom"
	"github.com/losinggeneration/rovel/style"
)

// Style types re-exported from style package for API convenience.
// The actual types are defined in the style package to avoid import cycles.

// Style represents foreground color, background color, and text attributes.
type Style = style.Style

// Color represents a terminal color.
type Color = style.Color

// AttrMask represents text attributes.
type AttrMask = style.AttrMask

// Color constants re-exported for convenience.
const (
	ColorDefault       = style.ColorDefault
	ColorBlack         = style.ColorBlack
	ColorRed           = style.ColorRed
	ColorGreen         = style.ColorGreen
	ColorYellow        = style.ColorYellow
	ColorBlue          = style.ColorBlue
	ColorMagenta       = style.ColorMagenta
	ColorCyan          = style.ColorCyan
	ColorWhite         = style.ColorWhite
	ColorBrightBlack   = style.ColorBrightBlack
	ColorBrightRed     = style.ColorBrightRed
	ColorBrightGreen   = style.ColorBrightGreen
	ColorBrightYellow  = style.ColorBrightYellow
	ColorBrightBlue    = style.ColorBrightBlue
	ColorBrightMagenta = style.ColorBrightMagenta
	ColorBrightCyan    = style.ColorBrightCyan
	ColorBrightWhite   = style.ColorBrightWhite
)

// Attribute constants re-exported for convenience.
const (
	AttrBold      = style.AttrBold
	AttrDim       = style.AttrDim
	AttrItalic    = style.AttrItalic
	AttrUnderline = style.AttrUnderline
	AttrBlink     = style.AttrBlink
	AttrReverse   = style.AttrReverse
)

type (
	Rect  = geom.Rect
	Point = geom.Point
	Size  = geom.Size
)
