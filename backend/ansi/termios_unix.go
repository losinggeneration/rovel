//go:build unix

package ansi

import (
	"github.com/losinggeneration/tui/geom"
	"golang.org/x/sys/unix"
)

// enableRaw puts the terminal into raw mode and returns the original state.
func enableRaw(fd int) (*unix.Termios, error) {
	if !isTerminal(fd) {
		return nil, unix.EINVAL
	}

	orig, err := unix.IoctlGetTermios(fd, ioctlReadTermios)
	if err != nil {
		return nil, err
	}

	// Copy and modify for raw mode.
	raw := *orig
	raw.Iflag &^= unix.IGNBRK | unix.BRKINT | unix.PARMRK | unix.ISTRIP |
		unix.INLCR | unix.IGNCR | unix.ICRNL | unix.IXON
	raw.Oflag &^= unix.OPOST
	raw.Lflag &^= unix.ECHO | unix.ECHONL | unix.ICANON | unix.ISIG | unix.IEXTEN
	raw.Cflag &^= unix.CSIZE | unix.PARENB
	raw.Cflag |= unix.CS8
	raw.Cc[unix.VMIN] = 1
	raw.Cc[unix.VTIME] = 0

	if err := unix.IoctlSetTermios(fd, ioctlWriteTermios, &raw); err != nil {
		return nil, err
	}

	return orig, nil
}

// enableCBreak puts the terminal into cbreak mode: character-at-a-time
// input with ECHO disabled. ICANON and IEXTEN are cleared so keys like
// Tab and arrows reach the application immediately without waiting for
// Enter. ISIG stays on so Ctrl+C delivers SIGINT, and OPOST stays on so
// output is line-processed (text remains selectable). Suitable for
// interactive CLI tools like dialog boxes, prompts, and script-driven
// TUI components.
func enableCBreak(fd int) (*unix.Termios, error) {
	if !isTerminal(fd) {
		return nil, unix.EINVAL
	}

	orig, err := unix.IoctlGetTermios(fd, ioctlReadTermios)
	if err != nil {
		return nil, err
	}

	cbreak := *orig
	cbreak.Lflag &^= unix.ECHO | unix.ECHONL | unix.ICANON | unix.IEXTEN
	cbreak.Cc[unix.VMIN] = 1
	cbreak.Cc[unix.VTIME] = 0

	if err := unix.IoctlSetTermios(fd, ioctlWriteTermios, &cbreak); err != nil {
		return nil, err
	}

	return orig, nil
}

// restore restores the terminal to its original state.
func restore(fd int, orig *unix.Termios) error {
	if orig == nil {
		return nil
	}

	return unix.IoctlSetTermios(fd, ioctlWriteTermios, orig)
}

// getTerminalSize returns the current terminal size using the given output fd.
func getTerminalSize(fd int) (geom.Size, error) {
	ws, err := unix.IoctlGetWinsize(fd, unix.TIOCGWINSZ)
	if err != nil {
		return geom.Size{}, err
	}

	return geom.Size{
		W: int(ws.Col),
		H: int(ws.Row),
	}, nil
}

// isTerminal reports whether fd refers to a terminal.
func isTerminal(fd int) bool {
	_, err := unix.IoctlGetTermios(fd, ioctlReadTermios)

	return err == nil
}

// IsTerminal reports whether the file descriptor refers to a terminal (tty).
func IsTerminal(fd int) bool {
	return isTerminal(fd)
}
