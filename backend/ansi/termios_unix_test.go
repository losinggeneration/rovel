//go:build unix

package ansi

import (
	"os"
	"strings"
	"testing"

	"github.com/losinggeneration/tui/internal/errbuf"
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
	rawOrig, err := enableRaw(fd)
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
	if err := restore(fd, rawOrig); err != nil {
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

// TestCBreakModeIntegration tests that enableCBreak clears ECHO and ICANON
// but preserves ISIG and OPOST, and that restore brings back the original
// state.
func TestCBreakModeIntegration(t *testing.T) {
	if !isTerminal(int(os.Stdin.Fd())) {
		t.Skip("not a terminal")
	}

	fd := int(os.Stdin.Fd())

	var orig unix.Termios
	if err := unix.IoctlSetTermios(fd, unix.TCGETS, &orig); err != nil {
		t.Fatalf("failed to get terminal state: %v", err)
	}

	regOrig, err := enableCBreak(fd)
	if err != nil {
		t.Fatalf("enableCBreak failed: %v", err)
	}

	var current unix.Termios
	if err := unix.IoctlSetTermios(fd, unix.TCGETS, &current); err != nil {
		t.Fatalf("failed to get terminal state after enableCBreak: %v", err)
	}

	if current.Lflag&unix.ECHO != 0 {
		t.Error("ECHO not cleared in cbreak mode")
	}

	if current.Lflag&unix.ECHONL != 0 {
		t.Error("ECHONL not cleared in cbreak mode")
	}

	if current.Lflag&unix.ICANON != 0 {
		t.Error("ICANON not cleared in cbreak mode")
	}

	if current.Lflag&unix.IEXTEN != 0 {
		t.Error("IEXTEN not cleared in cbreak mode")
	}

	if current.Lflag&unix.ISIG == 0 {
		t.Error("ISIG should remain set in cbreak mode")
	}

	if current.Oflag&unix.OPOST == 0 {
		t.Error("OPOST should remain set in cbreak mode")
	}

	// Restore terminal
	if err := restore(fd, regOrig); err != nil {
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
		t.Error("terminal state not properly restored after cbreak mode")
	}
}

// TestGetTerminalSize tests getting terminal size.
func TestGetTerminalSize(t *testing.T) {
	if !isTerminal(int(os.Stdout.Fd())) {
		t.Skip("not a terminal")
	}

	size, err := getTerminalSize(int(os.Stdout.Fd()))
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

func TestNewRejectsNonTerminalInput(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}

	defer func() {
		if err := r.Close(); err != nil {
			t.Error("r.Close:", err)
		}

		if err := w.Close(); err != nil {
			t.Error("w.Close:", err)
		}
	}()

	_, err = New(errbuf.New(1), Options{
		Input:  r,
		Output: os.Stdout,
	})
	if err == nil {
		t.Fatal("expected error for non-terminal input")
	}

	if !strings.Contains(err.Error(), "input fd") {
		t.Fatalf("expected input terminal error, got %v", err)
	}
}

func TestNewRejectsNonTerminalOutput(t *testing.T) {
	if !isTerminal(int(os.Stdin.Fd())) {
		t.Skip("requires terminal stdin to isolate output validation")
	}

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}

	defer func() {
		if err := r.Close(); err != nil {
			t.Error("r.Close:", err)
		}

		if err := w.Close(); err != nil {
			t.Error("w.Close:", err)
		}
	}()

	_, err = New(errbuf.New(1), Options{
		Input:  os.Stdin,
		Output: w,
	})
	if err == nil {
		t.Fatal("expected error for non-terminal output")
	}

	if !strings.Contains(err.Error(), "output fd") {
		t.Fatalf("expected output terminal error, got %v", err)
	}
}
