package rovel

import (
	"github.com/losinggeneration/rovel/geom"
	"github.com/losinggeneration/rovel/style"
)

// Chrome defines shared "chrome" primitives for box-like decorations across
// components. It is geometry-first: colors typically come from Theme.Palette,
// with optional per-cell styling via FrameChromeStyleFn.
type Chrome struct {
	Border    FrameChrome
	FocusRing FrameChrome
}

// FrameChrome describes how to draw a frame (borders, rings, panels, etc).
type FrameChrome struct {
	Glyphs  BoxGlyphs
	Edges   BoxEdges
	StyleFn FrameChromeStyleFn
}

// FrameChromeStyleFn can compute a per-cell style for a frame, given a base
// style provided by the caller (usually derived from Theme.Palette).
type FrameChromeStyleFn func(part BoxPart, x, y int, r geom.Rect, base style.Style) style.Style

func (fn FrameChromeStyleFn) wrapResolver(resolve func(style.Style) style.Style) FrameChromeStyleFn {
	if fn == nil {
		return nil
	}

	return func(part BoxPart, x, y int, r geom.Rect, base style.Style) style.Style {
		return resolve(fn(part, x, y, r, base))
	}
}

// Effective returns a version of fc with defaults filled in based on theme.
//
// Backward-compat: if the theme's Aesthetic is unset (zero), the default glyphs
// remain the existing Unicode-light box drawing characters.
func (fc FrameChrome) Effective(t Theme) FrameChrome {
	out := fc
	if out.Edges == 0 {
		out.Edges = BoxEdgesAll
	}

	if out.Glyphs == (BoxGlyphs{}) {
		switch t.Aesthetic {
		case AestheticClassic:
			out.Glyphs = BoxGlyphsASCII
		case AestheticModern:
			out.Glyphs = BoxGlyphsLight
		default:
			out.Glyphs = BoxGlyphsLight
		}
	}

	return out
}

// BoxStyle converts fc into a BoxStyle for Painter.BoxStyled using base as the
// default cell style.
func (fc FrameChrome) BoxStyle(base style.Style) BoxStyle {
	bs := BoxStyle{
		Glyphs: fc.Glyphs,
		Edges:  fc.Edges,
		Style:  base,
	}
	if fc.StyleFn != nil {
		bs.StyleFn = func(part BoxPart, x, y int, r geom.Rect) style.Style {
			return fc.StyleFn(part, x, y, r, base)
		}
	}

	return bs
}
