package rovel

import (
	"testing"

	"github.com/losinggeneration/rovel/style"
)

func TestDefaultTheme(t *testing.T) {
	theme := DefaultTheme()

	// Base should use ColorDefault for both FG and BG (to emit 39 and 49)
	if theme.Base.FG != style.ColorDefault {
		t.Errorf("DefaultTheme() Base.FG = %v, want ColorDefault", theme.Base.FG)
	}

	if theme.Base.BG != style.ColorDefault {
		t.Errorf("DefaultTheme() Base.BG = %v, want ColorDefault", theme.Base.BG)
	}

	// Palette should have reasonable defaults
	if theme.Palette.Text == (style.Style{}) {
		t.Error("DefaultTheme() has zero Text style in palette")
	}

	if theme.Palette.Focus.Attr&style.AttrReverse == 0 {
		t.Error("DefaultTheme() Palette.Focus should have AttrReverse")
	}

	// Verify semantic roles all use ColorDefault FG (monochrome)
	// This is what makes Default different from Classic!
	if theme.Palette.Success.FG != style.ColorDefault {
		t.Errorf("DefaultTheme() Palette.Success.FG should be ColorDefault, got FG=%v", theme.Palette.Success.FG)
	}

	if theme.Palette.Warning.FG != style.ColorDefault {
		t.Errorf("DefaultTheme() Palette.Warning.FG should be ColorDefault, got FG=%v", theme.Palette.Warning.FG)
	}

	if theme.Palette.Danger.FG != style.ColorDefault {
		t.Errorf("DefaultTheme() Palette.Danger.FG should be ColorDefault, got FG=%v", theme.Palette.Danger.FG)
	}

	if theme.Palette.Accent.FG != style.ColorDefault {
		t.Errorf("DefaultTheme() Palette.Accent.FG should be ColorDefault, got FG=%v", theme.Palette.Accent.FG)
	}

	// Verify all semantic roles use ColorDefault for BG too (to emit 49)
	if theme.Palette.Success.BG != style.ColorDefault {
		t.Errorf("DefaultTheme() Palette.Success.BG should be ColorDefault, got BG=%v", theme.Palette.Success.BG)
	}

	if theme.Palette.Warning.BG != style.ColorDefault {
		t.Errorf("DefaultTheme() Palette.Warning.BG should be ColorDefault, got BG=%v", theme.Palette.Warning.BG)
	}

	if theme.Palette.Danger.BG != style.ColorDefault {
		t.Errorf("DefaultTheme() Palette.Danger.BG should be ColorDefault, got BG=%v", theme.Palette.Danger.BG)
	}
}

func TestDefaultThemeClassic(t *testing.T) {
	theme := DefaultThemeClassic()

	// Classic theme should use basic colors explicitly
	if theme.Base.FG.Kind() != style.ColorKindBasic {
		t.Errorf("DefaultThemeClassic() Base.FG should use basic colors, got %v", theme.Base.FG.Kind())
	}

	if theme.Base.FG != style.ColorBasic(7) { // White
		t.Errorf("DefaultThemeClassic() Base.FG = %v, want White", theme.Base.FG)
	}

	if theme.Base.BG != style.ColorBasic(0) { // Black
		t.Errorf("DefaultThemeClassic() Base.BG = %v, want Black", theme.Base.BG)
	}
}

func TestDefaultThemeModern(t *testing.T) {
	theme := DefaultThemeModern()

	// Modern theme should use RGB colors
	if theme.Base == (style.Style{}) {
		t.Error("DefaultThemeModern() has zero Base style")
	}

	// Should use RGB colors
	if theme.Base.FG.Kind() != style.ColorKindRGB {
		t.Errorf("DefaultThemeModern() Base should use RGB colors, got %v", theme.Base.FG.Kind())
	}

	if theme.Base.BG.Kind() != style.ColorKindRGB {
		t.Errorf("DefaultThemeModern() Base BG should use RGB colors, got %v", theme.Base.BG.Kind())
	}
}

func TestThemeResolved(t *testing.T) {
	theme := DefaultThemeModern()

	// Resolve for basic-only terminal
	c := style.Capability{HasBasic: true}
	resolved := theme.Resolved(c)

	// Resolved theme should be different from original
	if resolved.Base.FG.Kind() != style.ColorKindBasic {
		t.Errorf("Theme.Resolved() didn't degrade RGB to basic, got %v", resolved.Base.FG.Kind())
	}

	// Verify Base is preserved (just colors change)
	if resolved.Base.Attr != theme.Base.Attr {
		t.Error("Theme.Resolved() changed attributes")
	}

	// Resolve for 256-color terminal
	cap256 := style.Capability{HasBasic: true, Has256Color: true}
	resolved256 := theme.Resolved(cap256)

	// Should have indexed colors, not RGB
	if resolved256.Base.FG.Kind() != style.ColorKindIndexed {
		t.Errorf("Theme.Resolved() didn't map RGB to indexed, got %v", resolved256.Base.FG.Kind())
	}
}

func TestThemeResolved_Palette(t *testing.T) {
	theme := DefaultThemeModern()

	c := style.Capability{HasBasic: true}
	resolved := theme.Resolved(c)

	// All palette fields should be resolved
	// Just check a few representative fields
	if resolved.Palette.Surface == (style.Style{}) {
		t.Error("Resolved theme has zero Surface style")
	}

	if resolved.Palette.Text == (style.Style{}) {
		t.Error("Resolved theme has zero Text style")
	}

	if resolved.Palette.Focus == (style.Style{}) {
		t.Error("Resolved theme has zero Focus style")
	}

	// Verify colors were degraded to basic
	if resolved.Palette.Surface.FG.Kind() != style.ColorKindBasic &&
		resolved.Palette.Surface.FG != 0 {
		t.Errorf("Palette Surface not degraded to basic: %v", resolved.Palette.Surface.FG.Kind())
	}
}

func TestThemePreservesBaseAndPaletteFocus(t *testing.T) {
	// Verify that Theme.Resolved() preserves Base and Palette.Focus structure.
	original := Theme{
		Base: style.Style{FG: style.ColorRed, BG: style.ColorBlue},
		Palette: Palette{
			Focus: style.Style{FG: style.ColorBlue, BG: style.ColorRed, Attr: style.AttrReverse},
		},
	}

	c := style.Capability{HasBasic: true}
	resolved := original.Resolved(c)

	// Base and Palette.Focus should be preserved (just colors resolved)
	if resolved.Base.Attr != original.Base.Attr {
		t.Error("Resolved() changed Base Attr")
	}

	if resolved.Palette.Focus.Attr != original.Palette.Focus.Attr {
		t.Error("Resolved() changed Palette.Focus Attr")
	}

	// Colors should be the same (both already basic)
	if resolved.Base.FG != original.Base.FG {
		t.Errorf("Resolved() changed Base FG: %v -> %v", original.Base.FG, resolved.Base.FG)
	}
}

func TestThemeChrome_Defaults(t *testing.T) {
	classic := DefaultThemeClassic()

	ec := classic.Chrome.Border.Effective(classic)
	if ec.Glyphs != BoxGlyphsLight {
		t.Errorf("DefaultThemeClassic() Border glyphs = %+v, want ASCII", ec.Glyphs)
	}

	modern := DefaultThemeModern()

	em := modern.Chrome.Border.Effective(modern)
	if em.Glyphs != BoxGlyphsLight {
		t.Errorf("DefaultThemeModern() Border glyphs = %+v, want light", em.Glyphs)
	}

	// Backward-compat: unspecified aesthetic + zero chrome defaults to light.
	legacy := Theme{
		Base: style.Style{FG: style.ColorWhite, BG: style.ColorBlack},
	}

	el := legacy.Chrome.Border.Effective(legacy)
	if el.Glyphs != BoxGlyphsLight {
		t.Errorf("legacy theme Border glyphs = %+v, want light", el.Glyphs)
	}
}

func TestThemeResolved_DefaultsPaletteFocus(t *testing.T) {
	original := Theme{
		Base: style.Style{FG: style.ColorWhite, BG: style.ColorBlack},
	}

	c := style.Capability{HasBasic: true}

	resolved := original.Resolved(c)
	if resolved.Palette.Focus.Attr&style.AttrReverse == 0 {
		t.Error("Resolved() Palette.Focus should default to AttrReverse when unset")
	}
}

func TestThemeResolved_ChromeStyleFn(t *testing.T) {
	theme := DefaultThemeModern()
	theme.Chrome.Border.StyleFn = func(part BoxPart, x, y int, r Rect, base style.Style) style.Style {
		_ = part
		_ = x
		_ = y
		_ = r
		_ = base

		return style.Style{FG: style.ColorRGB(10, 20, 30), BG: style.ColorRGB(40, 50, 60)}
	}

	c := style.Capability{HasBasic: true}
	resolved := theme.Resolved(c)

	chrome := resolved.Chrome.Border.Effective(resolved)

	bs := chrome.BoxStyle(resolved.Palette.Border)
	if bs.StyleFn == nil {
		t.Fatal("resolved Border StyleFn is nil")
	}

	out := bs.StyleFn(BoxPartTop, 0, 0, Rect{X: 0, Y: 0, W: 3, H: 3})
	if out.FG.Kind() != style.ColorKindBasic && out.FG != 0 {
		t.Errorf("resolved StyleFn FG kind = %v, want basic or default", out.FG.Kind())
	}

	if out.BG.Kind() != style.ColorKindBasic && out.BG != 0 {
		t.Errorf("resolved StyleFn BG kind = %v, want basic or default", out.BG.Kind())
	}
}
