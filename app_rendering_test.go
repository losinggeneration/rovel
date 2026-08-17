package rovel

import (
	"bytes"
	"testing"

	"github.com/losinggeneration/rovel/backend"
	"github.com/losinggeneration/rovel/geom"
)

// fakeANSITransport models the real backend's buffering: writes land in a
// userspace buffer and only reach the terminal (visible) when Flush pushes
// them out.
type fakeANSITransport struct {
	pending []byte
	visible []byte
	flushes int
}

func (f *fakeANSITransport) Write(p []byte) (int, error) {
	f.pending = append(f.pending, p...)

	return len(p), nil
}

func (f *fakeANSITransport) Flush() error {
	f.flushes++
	f.visible = append(f.visible, f.pending...)
	f.pending = nil

	return nil
}

// RestoreScreen must push the teardown all the way to the terminal, not just
// into the backend's buffered writer. The suspend path depends on this: the
// process stops immediately after RestoreScreen returns, so teardown bytes
// still buffered in userspace are invisible until resume — the terminal keeps
// showing the app's frozen UI and the suspend looks like it never happened.
func TestRestoreScreenPushesTeardownToTerminal(t *testing.T) {
	ft := &fakeANSITransport{}
	p := newANSIPresenter(ft, backend.ModeRaw, false)

	if err := p.InitScreen(geom.Size{W: 4, H: 2}); err != nil {
		t.Fatalf("InitScreen: %v", err)
	}

	if err := p.RestoreScreen(); err != nil {
		t.Fatalf("RestoreScreen: %v", err)
	}

	if len(ft.pending) != 0 {
		t.Errorf("%d teardown bytes still buffered in the transport; they would not reach the screen until some later Flush", len(ft.pending))
	}

	if !bytes.Contains(ft.visible, []byte("\x1b[0m")) {
		t.Error("teardown did not reset SGR (ESC[0m never reached the terminal)")
	}

	if !bytes.Contains(ft.visible, []byte("\x1b[?25h")) {
		t.Error("teardown did not show the cursor (ESC[?25h never reached the terminal)")
	}
}

// The teardown must leave the terminal in a predictable state for whoever
// writes next: the bottom line erased to the terminal default background and
// the cursor parked at its first column. The shell's job-control notice after
// a suspend, and the app's exit message after a quit, then always land in the
// same place instead of wherever the last frame happened to leave the cursor
// (typically mid-frame, on a line still styled with the app's background).
func TestRestoreScreenParksCursorOnCleanBottomLine(t *testing.T) {
	ft := &fakeANSITransport{}
	p := newANSIPresenter(ft, backend.ModeRaw, false)

	if err := p.InitScreen(geom.Size{W: 4, H: 2}); err != nil {
		t.Fatalf("InitScreen: %v", err)
	}

	if err := p.RestoreScreen(); err != nil {
		t.Fatalf("RestoreScreen: %v", err)
	}

	if len(ft.pending) != 0 {
		t.Fatalf("%d teardown bytes still buffered in the transport", len(ft.pending))
	}

	want := "\x1b[0m\x1b[2;1H\x1b[K\x1b[?25h"
	if !bytes.Contains(ft.visible, []byte(want)) {
		t.Errorf("teardown = %q, want it to contain %q (SGR reset, park at row 2 col 1, erase line, show cursor)", ft.visible, want)
	}
}
