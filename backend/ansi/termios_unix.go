//go:build unix

package ansi

import (
	"syscall"
	"unsafe"

	"github.com/losinggeneration/tui/geom"
	"golang.org/x/sys/unix"
)

// enableRaw puts the terminal into raw mode and returns the original state.
func enableRaw(fd int) (*unix.Termios, error) {
	if !isTerminal(fd) {
		return nil, syscall.EINVAL
	}

	// Get current terminal settings
	var orig unix.Termios

	err := unix.IoctlSetTermios(fd, unix.TCGETS, &orig)
	if err != nil {
		return nil, err
	}

	// Copy and modify for raw mode
	raw := orig
	raw.Iflag &^= unix.IGNBRK | unix.BRKINT | unix.PARMRK | unix.ISTRIP |
		unix.INLCR | unix.IGNCR | unix.ICRNL | unix.IXON
	raw.Oflag &^= unix.OPOST
	raw.Lflag &^= unix.ECHO | unix.ECHONL | unix.ICANON | unix.ISIG | unix.IEXTEN
	raw.Cflag &^= unix.CSIZE | unix.PARENB
	raw.Cflag |= unix.CS8
	raw.Cc[unix.VMIN] = 1
	raw.Cc[unix.VTIME] = 0

	err = unix.IoctlSetTermios(fd, unix.TCSETS, &raw)
	if err != nil {
		return nil, err
	}

	return &orig, nil
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
		return nil, syscall.EINVAL
	}

	var orig unix.Termios

	err := unix.IoctlSetTermios(fd, unix.TCGETS, &orig)
	if err != nil {
		return nil, err
	}

	cbreak := orig
	cbreak.Lflag &^= unix.ECHO | unix.ECHONL | unix.ICANON | unix.IEXTEN
	cbreak.Cc[unix.VMIN] = 1
	cbreak.Cc[unix.VTIME] = 0

	err = unix.IoctlSetTermios(fd, unix.TCSETS, &cbreak)
	if err != nil {
		return nil, err
	}

	return &orig, nil
}

// restore restores the terminal to its original state.
func restore(fd int, orig *unix.Termios) error {
	if orig == nil {
		return nil
	}

	return unix.IoctlSetTermios(fd, unix.TCSETS, orig)
}

// getTerminalSize returns the current terminal size using the given output fd.
func getTerminalSize(fd int) (geom.Size, error) {
	var ws unix.Winsize

	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(fd),
		unix.TIOCGWINSZ,
		uintptr(unsafe.Pointer(&ws)),
	)
	if errno != 0 {
		return geom.Size{}, errno
	}

	return geom.Size{
		W: int(ws.Col),
		H: int(ws.Row),
	}, nil
}

// isTerminal returns true if fd refers to a terminal.
func isTerminal(fd int) bool {
	var termios unix.Termios

	return unix.IoctlSetTermios(fd, unix.TCGETS, &termios) == nil
}
