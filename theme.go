package tui

import "github.com/losinggeneration/tui/style"

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
	Theme Theme
	// Backend is optional; nil means use default backend/ansi on unix builds.
	// Backend interface.Backend
}

// DefaultAppOpts returns default application options.
func DefaultAppOpts() AppOpts {
	return AppOpts{
		Theme: DefaultTheme(),
	}
}
