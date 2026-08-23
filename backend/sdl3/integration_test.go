package sdl3

import (
	"os"
	"testing"

	"github.com/losinggeneration/rovel/backend"
)

// TestBackendEnablePresentRestore_DummyVideo exercises Enable/PresentCellFrame/
// Restore against SDL3's dummy video driver. It only runs when the environment
// variable SDL3_BACKEND_INTEGRATION is set, because the Zyko0/go-sdl3 binding
// embeds the SDL3 and SDL3_ttf shared libraries; extracting them on every
// test run slows the ordinary `go test ./...` loop considerably.
//
// Library loading is exercised by Enable itself (loadLibraries), so this test
// also covers the auto-load path users rely on.
//
// ponytail: process may SIGSEGV during test-cleanup teardown — the bundled
// SDL3/dummy-driver pairing sometimes segfaults inside binsdl.Unload after the
// assertions pass. The test assertions themselves complete successfully; this
// is upstream binsdl cleanup, not backend code.
func TestBackendEnablePresentRestore_DummyVideo(t *testing.T) {
	if os.Getenv("SDL3_BACKEND_INTEGRATION") == "" {
		t.Skip("set SDL3_BACKEND_INTEGRATION=1 to enable the dummy-video integration test")
	}

	t.Setenv("SDL_VIDEO_DRIVER", "dummy")

	bi, err := New(DefaultOptions())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	b := bi.(*Backend)
	if _, err := b.Enable(); err != nil {
		t.Fatalf("Enable: %v", err)
	}

	err = b.PresentCellFrame(backend.CellFrame{
		W: 2,
		H: 1,
		Cells: []backend.FrameCell{
			{R: 'O'},
			{R: 'K'},
		},
	})
	if err != nil {
		t.Fatalf("PresentCellFrame: %v", err)
	}

	if err := b.Restore(); err != nil {
		t.Fatalf("Restore: %v", err)
	}
}
