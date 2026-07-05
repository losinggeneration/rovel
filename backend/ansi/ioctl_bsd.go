//go:build darwin || dragonfly || freebsd || netbsd || openbsd

package ansi

import "golang.org/x/sys/unix"

// ioctl request numbers for reading and writing termios on macOS and the BSDs,
// which use TIOCGETA/TIOCSETA where Linux uses TCGETS/TCSETS.
const (
	ioctlReadTermios  = unix.TIOCGETA
	ioctlWriteTermios = unix.TIOCSETA
)
