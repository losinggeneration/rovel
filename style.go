package tui

// Color represents a terminal color.
type Color uint16

const (
	// ColorDefault represents the terminal's default color.
	ColorDefault Color = iota

	// Basic 16 colors (ANSI standard colors).
	ColorBlack
	ColorRed
	ColorGreen
	ColorYellow
	ColorBlue
	ColorMagenta
	ColorCyan
	ColorWhite

	// Bright variants (high-intensity colors).
	ColorBrightBlack
	ColorBrightRed
	ColorBrightGreen
	ColorBrightYellow
	ColorBrightBlue
	ColorBrightMagenta
	ColorBrightCyan
	ColorBrightWhite

	// 256-color palette range (216 colors + 24 grays).
	// Color256 begins the 256-color palette.
	Color256 Color = 232

	// TrueColor marker for RGB colors.
	ColorTrue Color = 1 << 15
)

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

// Equals returns true if two styles are identical.
func (s Style) Equals(other Style) bool {
	return s.FG == other.FG && s.BG == other.BG && s.Attr == other.Attr
}
