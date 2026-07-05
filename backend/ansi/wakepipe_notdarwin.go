//go:build unix && !darwin

package ansi

import "golang.org/x/sys/unix"

// newWakePipe creates the nonblocking, close-on-exec self-pipe used to wake the
// blocking poll loop. Platforms with pipe2(2) create it atomically.
func newWakePipe(fds []int) error {
	return unix.Pipe2(fds, unix.O_NONBLOCK|unix.O_CLOEXEC)
}
