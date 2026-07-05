//go:build linux

package ansi

import "golang.org/x/sys/unix"

// ioctl request numbers for reading and writing termios on Linux.
const (
	ioctlReadTermios  = unix.TCGETS
	ioctlWriteTermios = unix.TCSETS
)
