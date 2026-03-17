//go:build unix

package ansi

import (
	"os"
	"testing"

	"golang.org/x/sys/unix"
)

// TestRawModeIntegration tests that we can enable and restore raw mode.
// This is an integration test that requires a terminal.
func TestRawModeIntegration(t *testing.T) {
	// Skip if not running in a terminal
	if !isTerminal(int(os.Stdin.Fd())) {
		t.Skip("not a terminal")
	}

	// Get original state
	fd := int(os.Stdin.Fd())

	var orig unix.Termios
	if err := unix.IoctlSetTermios(fd, unix.TCGETS, &orig); err != nil {
		t.Fatalf("failed to get terminal state: %v", err)
	}

	// Enable raw mode
	rawOrig, err := enableRaw()
	if err != nil {
		t.Fatalf("enableRaw failed: %v", err)
	}

	// Verify raw mode is set (check that ICANON is cleared)
	var current unix.Termios
	if err := unix.IoctlSetTermios(fd, unix.TCGETS, &current); err != nil {
		t.Fatalf("failed to get terminal state after enableRaw: %v", err)
	}

	if current.Lflag&unix.ICANON != 0 {
		t.Error("ICANON not cleared in raw mode")
	}

	if current.Lflag&unix.ECHO != 0 {
		t.Error("ECHO not cleared in raw mode")
	}

	// Restore terminal
	if err := restore(rawOrig); err != nil {
		t.Fatalf("restore failed: %v", err)
	}

	// Verify original state is restored
	if err := unix.IoctlSetTermios(fd, unix.TCGETS, &current); err != nil {
		t.Fatalf("failed to get terminal state after restore: %v", err)
	}

	if current.Lflag != orig.Lflag ||
		current.Iflag != orig.Iflag ||
		current.Oflag != orig.Oflag ||
		current.Cflag != orig.Cflag {
		t.Error("terminal state not properly restored")
	}
}

// TestGetTerminalSize tests getting terminal size.
func TestGetTerminalSize(t *testing.T) {
	if !isTerminal(int(os.Stdout.Fd())) {
		t.Skip("not a terminal")
	}

	size, err := getTerminalSize()
	if err != nil {
		t.Fatalf("getTerminalSize failed: %v", err)
	}

	if size.W <= 0 || size.H <= 0 {
		t.Errorf("invalid terminal size: %dx%d", size.W, size.H)
	}

	// Reasonable bounds check (most terminals should be at least 20x10)
	if size.W < 20 || size.H < 10 {
		t.Logf("warning: unusually small terminal size: %dx%d", size.W, size.H)
	}
}
