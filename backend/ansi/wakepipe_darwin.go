//go:build darwin

package ansi

import "golang.org/x/sys/unix"

// newWakePipe creates the nonblocking, close-on-exec self-pipe used to wake the
// blocking poll loop. macOS has no pipe2(2), so create an ordinary pipe and set
// the flags with fcntl, closing both ends if any step fails.
func newWakePipe(fds []int) error {
	if err := unix.Pipe(fds); err != nil {
		return err
	}

	for _, fd := range fds {
		if err := unix.SetNonblock(fd, true); err != nil {
			_ = unix.Close(fds[0])
			_ = unix.Close(fds[1])

			return err
		}

		unix.CloseOnExec(fd)
	}

	return nil
}
