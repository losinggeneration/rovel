package tui

import (
	"github.com/losinggeneration/tui/backend"
	"github.com/losinggeneration/tui/style"
)

// Palette defines semantic theme roles for consistent styling.
type Palette struct {
	Surface      style.Style
	SurfaceMuted style.Style
	Text         style.Style
	TextMuted    style.Style
	Border       style.Style
	BorderMuted  style.Style
	Focus        style.Style
	Selection    style.Style
	Accent       style.Style
	Placeholder  style.Style
	Success      style.Style
	Warning      style.Style
	Danger       style.Style
	Disabled     style.Style
}

// Theme defines the styling for the application.
type Theme struct {
	Base      style.Style // Clear-cell style (Paint Contract A)
	Focus     style.Style // Kept for backward compatibility
	Aesthetic Aesthetic   // High-level look hint (classic vs modern)
	Chrome    Chrome      // Box-like chrome (borders, focus rings, etc)
	Palette   Palette
}

// Aesthetic is a high-level, non-prescriptive hint for widget "chrome"
// decisions (glyphs, outlines vs flat fills, etc). It avoids per-widget theme
// trees while still letting themes express a consistent visual direction.
type Aesthetic uint8

const (
	AestheticClassic Aesthetic = iota + 1
	AestheticModern
)

func (t Theme) effectiveAesthetic() Aesthetic {
	if t.Aesthetic != 0 {
		return t.Aesthetic
	}
	// Zero value preserves pre-P3 behavior: classic terminal chrome.
	return AestheticClassic
}

// EffectiveAesthetic returns the theme's aesthetic, defaulting to classic if
// the zero value is used.
func (t Theme) EffectiveAesthetic() Aesthetic {
	return t.effectiveAesthetic()
}

// DefaultTheme returns the default theme (monochrome using terminal default FG).
func DefaultTheme() Theme {
	// Use ColorDefault for BOTH FG and BG to emit 39 and 49
	// This tells the terminal to use its actual default colors
	monochrome := style.Style{
		FG: style.ColorDefault, // Will emit 39 (terminal default FG)
		BG: style.ColorDefault, // Will emit 49 (terminal default BG)
	}

	return Theme{
		Base:      monochrome,
		Focus:     monochrome.WithAttr(style.AttrReverse),
		Aesthetic: AestheticClassic,
		Chrome: Chrome{
			Border: FrameChrome{Glyphs: BoxGlyphsASCII, Edges: BoxEdgesAll},
			FocusRing: FrameChrome{
				Glyphs: BoxGlyphsASCII,
				Edges:  BoxEdgesAll,
			},
		},
		Palette: Palette{
			Surface:      monochrome,
			SurfaceMuted: monochrome,
			Text:         monochrome.WithAttr(style.AttrBold),                       // Bold for emphasis
			TextMuted:    monochrome,                                                // Normal for muted
			Border:       monochrome,                                                // Normal border
			BorderMuted:  monochrome.WithAttr(style.AttrDim),                        // Dim for muted border
			Focus:        monochrome.WithAttr(style.AttrReverse),                    // Reverse for focus
			Selection:    monochrome.WithAttr(style.AttrReverse),                    // Reverse for selection
			Accent:       monochrome.WithAttr(style.AttrBold),                       // Bold for accent
			Placeholder:  monochrome.WithAttr(style.AttrDim),                        // Dim for placeholder
			Success:      monochrome.WithAttr(style.AttrBold),                       // Bold for success (monochrome)
			Warning:      monochrome.WithAttr(style.AttrUnderline),                  // Underline for warning (monochrome)
			Danger:       monochrome.WithAttr(style.AttrBold | style.AttrUnderline), // Bold+underline for danger (monochrome)
			Disabled:     monochrome.WithAttr(style.AttrDim),                        // Dim for disabled
		},
	}
}

// DefaultThemeClassic returns a classic "retro terminal" aesthetic with 16 ANSI colors.
func DefaultThemeClassic() Theme {
	black := style.ColorBasic(0)
	white := style.ColorBasic(7)
	gray := style.ColorBasic(8)
	brightWhite := style.ColorBasic(15)

	base := style.Style{
		FG: white,
		BG: black,
	}

	return Theme{
		Base:      base,
		Focus:     base.WithAttr(style.AttrReverse),
		Aesthetic: AestheticClassic,
		Chrome: Chrome{
			Border: FrameChrome{Glyphs: BoxGlyphsASCII, Edges: BoxEdgesAll},
			FocusRing: FrameChrome{
				Glyphs: BoxGlyphsASCII,
				Edges:  BoxEdgesAll,
			},
		},
		Palette: Palette{
			Surface:      base,
			SurfaceMuted: base.WithBG(black),
			Text:         base,
			TextMuted:    base.WithFG(gray),
			Border:       base.WithFG(brightWhite),
			BorderMuted:  base.WithFG(white),
			Focus:        base.WithAttr(style.AttrReverse),
			Selection:    base.WithFG(black).WithBG(white),
			Accent:       base.WithFG(style.ColorBasic(14)), // Cyan (distinct from danger)
			Placeholder:  base.WithFG(gray),
			Success:      base.WithFG(style.ColorBasic(2)), // Green
			Warning:      base.WithFG(style.ColorBasic(3)), // Yellow
			Danger:       base.WithFG(style.ColorBasic(1)), // Red
			Disabled:     base.WithAttr(style.AttrDim),
		},
	}
}

// DefaultThemeModern returns a modern "flat UI" aesthetic using RGB colors.
func DefaultThemeModern() Theme {
	bgDark := style.ColorRGB(30, 30, 30)
	bgSurface := style.ColorRGB(40, 40, 45)
	bgSurfaceMuted := style.ColorRGB(35, 35, 40)
	fgDefault := style.ColorRGB(220, 220, 220)
	fgMuted := style.ColorRGB(150, 150, 150)
	fgBorder := style.ColorRGB(80, 80, 80)
	fgBorderMuted := style.ColorRGB(60, 60, 60)
	accent := style.ColorRGB(100, 150, 255)
	placeholder := style.ColorRGB(120, 120, 120)
	success := style.ColorRGB(80, 200, 120)
	warning := style.ColorRGB(220, 180, 80)
	danger := style.ColorRGB(220, 80, 80)
	disabled := style.ColorRGB(100, 100, 100)
	focusFG := style.ColorRGB(30, 30, 30)
	focusBG := style.ColorRGB(100, 150, 255)

	return Theme{
		Base: style.Style{
			FG: fgDefault,
			BG: bgDark,
		},
		Focus: style.Style{
			FG: focusFG,
			BG: focusBG,
		},
		Aesthetic: AestheticModern,
		Chrome: Chrome{
			Border: FrameChrome{Glyphs: BoxGlyphsLight, Edges: BoxEdgesAll},
			FocusRing: FrameChrome{
				Glyphs: BoxGlyphsLight,
				Edges:  BoxEdgesAll,
			},
		},
		Palette: Palette{
			Surface:      style.Style{FG: fgDefault, BG: bgSurface},
			SurfaceMuted: style.Style{FG: fgDefault, BG: bgSurfaceMuted},
			Text:         style.Style{FG: fgDefault, BG: bgDark},
			TextMuted:    style.Style{FG: fgMuted, BG: bgDark},
			Border:       style.Style{FG: fgBorder, BG: bgDark},
			BorderMuted:  style.Style{FG: fgBorderMuted, BG: bgDark},
			Focus:        style.Style{FG: focusFG, BG: focusBG},
			Selection:    style.Style{FG: focusFG, BG: focusBG},
			Accent:       style.Style{FG: accent, BG: bgDark},
			Placeholder:  style.Style{FG: placeholder, BG: bgDark},
			Success:      style.Style{FG: success, BG: bgDark},
			Warning:      style.Style{FG: warning, BG: bgDark},
			Danger:       style.Style{FG: danger, BG: bgDark},
			Disabled:     style.Style{FG: disabled, BG: bgDark},
		},
	}
}

// Resolved returns a new theme with colors resolved for the given capability.
func (t Theme) Resolved(cap style.Capability) Theme {
	resolved := t

	resolveStyle := func(s style.Style) style.Style {
		resolved := s
		resolved.FG = s.FG.Resolve(cap)
		resolved.BG = s.BG.Resolve(cap)
		return resolved
	}

	resolved.Base = resolveStyle(t.Base)
	resolved.Focus = resolveStyle(t.Focus)
	resolved.Aesthetic = t.Aesthetic
	resolved.Chrome = t.Chrome

	// Resolve all palette fields
	resolved.Palette.Surface = resolveStyle(t.Palette.Surface)
	resolved.Palette.SurfaceMuted = resolveStyle(t.Palette.SurfaceMuted)
	resolved.Palette.Text = resolveStyle(t.Palette.Text)
	resolved.Palette.TextMuted = resolveStyle(t.Palette.TextMuted)
	resolved.Palette.Border = resolveStyle(t.Palette.Border)
	resolved.Palette.BorderMuted = resolveStyle(t.Palette.BorderMuted)
	resolved.Palette.Focus = resolveStyle(t.Palette.Focus)
	resolved.Palette.Selection = resolveStyle(t.Palette.Selection)
	resolved.Palette.Accent = resolveStyle(t.Palette.Accent)
	resolved.Palette.Placeholder = resolveStyle(t.Palette.Placeholder)
	resolved.Palette.Success = resolveStyle(t.Palette.Success)
	resolved.Palette.Warning = resolveStyle(t.Palette.Warning)
	resolved.Palette.Danger = resolveStyle(t.Palette.Danger)
	resolved.Palette.Disabled = resolveStyle(t.Palette.Disabled)

	// Ensure chrome style functions degrade across capabilities.
	resolved.Chrome.Border.StyleFn = resolved.Chrome.Border.StyleFn.wrapResolver(resolveStyle)
	resolved.Chrome.FocusRing.StyleFn = resolved.Chrome.FocusRing.StyleFn.wrapResolver(resolveStyle)

	return resolved
}

// AppOpts holds application options.
type AppOpts struct {
	Theme      Theme
	Backend    backend.Backend   // Optional: custom backend, nil uses default backend/ansi
	Capability *style.Capability // nil => auto-detect from env
}

// DefaultAppOpts returns default application options.
func DefaultAppOpts() AppOpts {
	return AppOpts{
		Theme: DefaultTheme(),
	}
}
