package rovel

import "github.com/losinggeneration/rovel/backend"

// TerminalMode controls how the backend configures the terminal's input mode.
// Re-exported from backend for convenience.
type TerminalMode = backend.TerminalMode

const (
	// ModeRaw puts the terminal into full raw mode (default). Character-at-a-time
	// input, no echo, no signal keys. Required for full-featured TUI apps.
	ModeRaw = backend.ModeRaw

	// ModeCBreak puts the terminal into cbreak mode: character-at-a-time
	// input with ECHO disabled, but ISIG stays on so Ctrl+C delivers SIGINT
	// and OPOST stays on so output is line-processed. Mouse tracking and
	// bracketed paste are unavailable; text remains selectable in the
	// terminal. The screen is not cleared on startup; output is drawn
	// inline at the current cursor position.
	ModeCBreak = backend.ModeCBreak
)
