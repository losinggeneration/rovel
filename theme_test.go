package tui

import (
	"testing"

	"github.com/losinggeneration/tui/style"
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

	// Focus should be defined and have AttrReverse
	if theme.Focus.Attr&style.AttrReverse == 0 {
		t.Error("DefaultTheme() Focus should have AttrReverse")
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
	cap := style.Capability{HasBasic: true}
	resolved := theme.Resolved(cap)

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

	cap := style.Capability{HasBasic: true}
	resolved := theme.Resolved(cap)

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

func TestThemePreservesBaseAndFocus(t *testing.T) {
	// Verify that Theme.Resolved() preserves Base and Focus structure
	// even if Palette is added later
	original := Theme{
		Base:  style.Style{FG: style.ColorRed, BG: style.ColorBlue},
		Focus: style.Style{FG: style.ColorBlue, BG: style.ColorRed, Attr: style.AttrReverse},
	}

	cap := style.Capability{HasBasic: true}
	resolved := original.Resolved(cap)

	// Base and Focus should be preserved (just colors resolved)
	if resolved.Base.Attr != original.Base.Attr {
		t.Error("Resolved() changed Base Attr")
	}

	if resolved.Focus.Attr != original.Focus.Attr {
		t.Error("Resolved() changed Focus Attr")
	}

	// Colors should be the same (both already basic)
	if resolved.Base.FG != original.Base.FG {
		t.Errorf("Resolved() changed Base FG: %v -> %v", original.Base.FG, resolved.Base.FG)
	}
}
