package tui

import (
	"github.com/losinggeneration/tui/backend"
	"github.com/losinggeneration/tui/style"
)

// Theme defines the base style for rendering.
type Theme struct {
	Base style.Style
}

// DefaultTheme returns the default theme with sensible defaults.
func DefaultTheme() Theme {
	return Theme{
		Base: style.Style{
			FG: style.ColorWhite,
			BG: style.ColorBlack,
		},
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
