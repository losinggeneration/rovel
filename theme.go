package tui

import (
	"github.com/losinggeneration/tui/backend"
	"github.com/losinggeneration/tui/style"
)

// Theme defines the base style for rendering.
type Theme struct {
	Base  style.Style
	Focus style.Style
}

// DefaultTheme returns the default theme with sensible defaults.
func DefaultTheme() Theme {
	base := style.Style{
		FG: style.ColorDefault,
		BG: style.ColorDefault,
	}
	focus := base
	focus.Attr |= style.AttrReverse

	return Theme{
		Base:  base,
		Focus: focus,
	}
}

// AppOpts holds application options.
type AppOpts struct {
	Theme   Theme
	Backend backend.Backend // Optional: custom backend, nil uses default backend/ansi
}

// DefaultAppOpts returns default application options.
func DefaultAppOpts() AppOpts {
	return AppOpts{
		Theme: DefaultTheme(),
	}
}
