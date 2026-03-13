package style

import (
	"os"
	"strings"
)

// Color is a tagged uint32 representing a terminal color.
// Bits 24-31: kind (0=default, 1=basic16, 2=indexed256, 3=rgb888)
// Bits 0-23: color value
type Color uint32

const (
	kindMask  = 0xFF000000
	valueMask = 0x00FFFFFF
	kindShift = 24
)

// ColorKind represents the kind of color.
type ColorKind uint8

const (
	ColorKindDefault ColorKind = 0
	ColorKindBasic   ColorKind = 1
	ColorKindIndexed ColorKind = 2
	ColorKindRGB     ColorKind = 3
)

// ColorBasic creates a basic 16-color (ANSI) color.
// Index should be 0-15 (0=black through 15=bright white).
func ColorBasic(index uint8) Color {
	if index > 15 {
		index = 0
	}
	return Color(uint32(ColorKindBasic)<<kindShift | uint32(index))
}

// ColorIndex creates a 256-color palette color (xterm palette).
// Index should be 0-255.
func ColorIndex(index uint8) Color {
	return Color(uint32(ColorKindIndexed)<<kindShift | uint32(index))
}

// ColorRGB creates a truecolor RGB value.
// Each component should be 0-255.
func ColorRGB(r, g, b uint8) Color {
	return Color(uint32(ColorKindRGB)<<kindShift | uint32(r)<<16 | uint32(g)<<8 | uint32(b))
}

// Kind returns the color kind.
func (c Color) Kind() ColorKind {
	return ColorKind(c >> kindShift)
}

// BasicIndex returns the basic color index (0-15) if kind is basic.
func (c Color) BasicIndex() (uint8, bool) {
	if c.Kind() != ColorKindBasic {
		return 0, false
	}
	return uint8(c & valueMask), true
}

// Index returns the palette index (0-255) if kind is indexed.
func (c Color) Index() (uint8, bool) {
	if c.Kind() != ColorKindIndexed {
		return 0, false
	}
	return uint8(c & valueMask), true
}

// RGB returns the RGB components if kind is RGB.
func (c Color) RGB() (r, g, b uint8, ok bool) {
	if c.Kind() != ColorKindRGB {
		return 0, 0, 0, false
	}
	v := uint32(c & valueMask)
	return uint8(v >> 16), uint8(v >> 8), uint8(v), true
}

// IsDefault returns true if this is the default color.
func (c Color) IsDefault() bool {
	return c.Kind() == ColorKindDefault
}

// Basic 16-color constants.
// These use the tagged uint32 encoding with ColorKindBasic in bits 24-31
// and the color index 0-15 in bits 0-23.
const (
	ColorBlack         Color = (1 << 24) | iota // 0x01000000
	ColorRed                                    // 0x01000001
	ColorGreen                                  // 0x01000002
	ColorYellow                                 // 0x01000003
	ColorBlue                                   // 0x01000004
	ColorMagenta                                // 0x01000005
	ColorCyan                                   // 0x01000006
	ColorWhite                                  // 0x01000007
	ColorBrightBlack                            // 0x01000008
	ColorBrightRed                              // 0x01000009
	ColorBrightGreen                            // 0x0100000A
	ColorBrightYellow                           // 0x0100000B
	ColorBrightBlue                             // 0x0100000C
	ColorBrightMagenta                          // 0x0100000D
	ColorBrightCyan                             // 0x0100000E
	ColorBrightWhite                            // 0x0100000F
)

// ColorDefault is the default terminal color (kind=0, value=0).
const ColorDefault Color = 0

// Capability describes terminal color capabilities.
type Capability struct {
	HasBasic     bool // always true
	Has256Color  bool
	HasTrueColor bool
}

// DetectCapabilityFromEnv detects color capabilities from environment variables.
func DetectCapabilityFromEnv() Capability {
	cap := Capability{HasBasic: true}

	// Check COLORTERM for truecolor indicator
	if colorterm := os.Getenv("COLORTERM"); colorterm == "truecolor" || colorterm == "24bit" {
		cap.HasTrueColor = true
		cap.Has256Color = true
		return cap
	}

	// Check TERM for 256-color support
	term := os.Getenv("TERM")
	if strings.Contains(term, "256color") {
		cap.Has256Color = true
	}

	return cap
}

// Resolve resolves color to supported capability level.
func (c Color) Resolve(cap Capability) Color {
	kind := c.Kind()

	// Already supported or default
	if kind == ColorKindDefault || kind == ColorKindBasic {
		return c
	}
	if kind == ColorKindIndexed && cap.Has256Color {
		return c
	}
	if kind == ColorKindRGB && cap.HasTrueColor {
		return c
	}

	// Need fallback
	if kind == ColorKindRGB {
		if cap.Has256Color {
			return c.rgbToIndexed()
		}
		return c.toBasic()
	}

	if kind == ColorKindIndexed {
		return c.toBasic()
	}

	return c
}

// rgbToIndexed maps RGB to nearest xterm-256 color using Euclidean distance.
func (c Color) rgbToIndexed() Color {
	r, g, b, _ := c.RGB()

	bestIdx := 0
	bestDist := uint32(0xFFFFFFFF)

	for i := 0; i < 256; i++ {
		xr, xg, xb := xterm256RGB(i)
		dr := uint32(r) - uint32(xr)
		dg := uint32(g) - uint32(xg)
		db := uint32(b) - uint32(xb)
		dist := dr*dr + dg*dg + db*db

		if dist < bestDist {
			bestDist = dist
			bestIdx = i
		}
	}

	return ColorIndex(uint8(bestIdx))
}

// toBasic maps indexed or RGB to nearest basic-16 color.
func (c Color) toBasic() Color {
	var r, g, b uint8

	if c.Kind() == ColorKindRGB {
		r, g, b, _ = c.RGB()
	} else if c.Kind() == ColorKindIndexed {
		idx, _ := c.Index()
		r, g, b = xterm256RGB(int(idx))
	} else {
		return c
	}

	// Simplified RGB to basic-16 mapping
	if r > 128 && g > 128 && b > 128 {
		if r > 200 && g > 200 && b > 200 {
			return ColorBrightWhite
		}
		return ColorWhite
	}

	if r > 128 {
		if g > 128 {
			return ColorYellow
		}
		if b > 128 {
			return ColorMagenta
		}
		return ColorRed
	}

	if g > 128 {
		if b > 128 {
			return ColorCyan
		}
		return ColorGreen
	}

	if b > 128 {
		return ColorBlue
	}

	return ColorBlack
}

// xterm256RGB returns the RGB values for an xterm-256 palette index.
// Precomputed table for performance.
func xterm256RGB(i int) (r, g, b uint8) {
	// Color cube 16-231 (6x6x6)
	if i >= 16 && i <= 231 {
		idx := i - 16
		r = uint8((idx/36)*40 + 55)
		g = uint8(((idx/6)%6)*40 + 55)
		b = uint8((idx%6)*40 + 55)
		return r, g, b
	}

	// Grayscale 232-255
	if i >= 232 {
		gray := 8 + (i-232)*10
		return uint8(gray), uint8(gray), uint8(gray)
	}

	// Basic colors 0-15 (standard xterm values)
	basic := [...]struct {
		r, g, b uint8
	}{
		{0, 0, 0},       // 0: black
		{205, 0, 0},     // 1: red
		{0, 205, 0},     // 2: green
		{205, 205, 0},   // 3: yellow
		{0, 0, 238},     // 4: blue
		{205, 0, 205},   // 5: magenta
		{0, 205, 205},   // 6: cyan
		{229, 229, 229}, // 7: white
		{127, 127, 127}, // 8: bright black
		{255, 0, 0},     // 9: bright red
		{0, 255, 0},     // 10: bright green
		{255, 255, 0},   // 11: bright yellow
		{92, 92, 255},   // 12: bright blue
		{255, 0, 255},   // 13: bright magenta
		{0, 255, 255},   // 14: bright cyan
		{255, 255, 255}, // 15: bright white
	}

	if i >= 0 && i < 16 {
		c := basic[i]
		return c.r, c.g, c.b
	}

	return 128, 128, 128
}
