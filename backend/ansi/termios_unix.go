//go:build unix

package ansi

import (
	"os"
	"syscall"
	"unsafe"

	"github.com/losinggeneration/tui/geom"
	"golang.org/x/sys/unix"
)

// enableRaw puts the terminal into raw mode and returns the original state.
func enableRaw() (*unix.Termios, error) {
	fd := int(os.Stdin.Fd())
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

// restore restores the terminal to its original state.
func restore(orig *unix.Termios) error {
	if orig == nil {
		return nil
	}

	fd := int(os.Stdin.Fd())

	return unix.IoctlSetTermios(fd, unix.TCSETS, orig)
}

// getTerminalSize returns the current terminal size.
func getTerminalSize() (geom.Size, error) {
	var ws unix.Winsize

	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		os.Stdout.Fd(),
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

	err := unix.IoctlSetTermios(fd, unix.TCGETS, &termios)

	return err == nil
}
