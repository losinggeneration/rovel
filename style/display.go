package style

// RGBA is a concrete display color for non-terminal presentation backends.
type RGBA struct {
	R uint8
	G uint8
	B uint8
	A uint8
}

// DisplayRGBA resolves a Color to a concrete RGBA value for surface
// presentation. Default colors map to fallback.
func (c Color) DisplayRGBA(fallback RGBA) RGBA {
	switch c.Kind() {
	case ColorKindDefault:
		return fallback
	case ColorKindBasic:
		idx, _ := c.BasicIndex()
		r, g, b := xterm256RGB(int(idx))

		return RGBA{R: r, G: g, B: b, A: 0xFF}
	case ColorKindIndexed:
		idx, _ := c.Index()
		r, g, b := xterm256RGB(int(idx))

		return RGBA{R: r, G: g, B: b, A: 0xFF}
	case ColorKindRGB:
		r, g, b, _ := c.RGB()

		return RGBA{R: r, G: g, B: b, A: 0xFF}
	default:
		return fallback
	}
}

// ResolvedForDisplay resolves default colors against base and applies reverse
// so presentation backends can obtain effective display colors directly.
func (s Style) ResolvedForDisplay(base Style) (fg, bg Color) {
	resolved := Merge(base, s)
	fg = resolved.FG
	bg = resolved.BG

	if resolved.Attr&AttrReverse != 0 {
		fg, bg = bg, fg
	}

	return fg, bg
}

// DisplayRGBA resolves effective foreground and background RGBA values for a
// style against a base style and display defaults.
func (s Style) DisplayRGBA(base Style, defaultFG, defaultBG RGBA) (fg, bg RGBA) {
	fgColor, bgColor := s.ResolvedForDisplay(base)

	return fgColor.DisplayRGBA(defaultFG), bgColor.DisplayRGBA(defaultBG)
}
